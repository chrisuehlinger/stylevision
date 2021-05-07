package main

import (
	"encoding/json"
	"time"

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
)

func main() {
	inCodec := os.Getenv("IN_CODEC")
	mimeType := "video/" + inCodec
	var availableCodecs []webrtc.RTPCodecParameters
	if inCodec == "h264" {
		availableCodecs = h264codecs
	} else if inCodec == "vp8" {
		availableCodecs = vp8codecs
	} else if inCodec == "vp9" {
		availableCodecs = vp9codecs
	}

	http.HandleFunc("/connectreceiver", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)

		ans := make(chan string)

		go func() {

			offer := webrtc.SessionDescription{}
			json.Unmarshal(body, &offer)

			m := &webrtc.MediaEngine{}
			for _, codec := range availableCodecs {
				if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
					panic(err)
				}
			}

			// Create a InterceptorRegistry. This is the user configurable RTP/RTCP Pipeline.
			// This provides NACKs, RTCP Reports and other features. If you use `webrtc.NewPeerConnection`
			// this is enabled by default. If you are manually managing You MUST create a InterceptorRegistry
			// for each PeerConnection.
			i := &interceptor.Registry{}

			// Use the default set of Interceptors
			if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
				panic(err)
			}

			// Create the API object with the MediaEngine
			api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

			// Prepare the configuration
			config := webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{
						URLs: []string{"stun:stun.l.google.com:19302"},
					},
				},
			}

			// Create a new RTCPeerConnection
			peerConnection, err := api.NewPeerConnection(config)
			if err != nil {
				panic(err)
			}

			// Create Track that we send video back to browser on
			outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: mimeType}, "video", "pion")
			if err != nil {
				panic(err)
			}

			// Add this newly created track to the PeerConnection
			rtpSender, err := peerConnection.AddTrack(outputTrack)
			if err != nil {
				panic(err)
			}

			if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
				panic(err)
			}

			// Read incoming RTCP packets
			// Before these packets are retuned they are processed by interceptors. For things
			// like NACK this needs to be called.
			go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
						return
					}
				}
			}()

			var outFile media.Writer
			if inCodec == "h264" {
				outFile = h264writer.NewWith(os.Stdout)
			} else if inCodec == "vp8" || inCodec == "vp9" {
				outFile, err = ivfwriter.NewWith(os.Stdout)
				if err != nil {
					panic(err)
				}
			}

			peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
				go func() {
					ticker := time.NewTicker(time.Second * 1)
					for range ticker.C {
						if rtcpErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}}); rtcpErr != nil {
							log.Println(rtcpErr)
						}
						if writeErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 4 * 1024 * 1024, SenderSSRC: uint32(track.SSRC())}}); writeErr != nil {
							log.Println(writeErr)
						}
					}
				}()

				for {
					// Read RTP packets being sent to Pion
					rtp, _, readErr := track.ReadRTP()
					if readErr != nil {
						panic(readErr)
					}

					if len(rtp.Payload) <= 2 {
						log.Println("OHNO! SMALL PACKET!")
					} else {
						if writeErr := outFile.WriteRTP(rtp); writeErr != nil {
							panic(writeErr)
						}
					}

					if writeErr := outputTrack.WriteRTP(rtp); writeErr != nil {
						panic(writeErr)
					}
					time.Sleep(time.Millisecond)
				}
			})

			// Set the handler for ICE connection state
			// This will notify you when the peer has connected/disconnected
			peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				fmt.Println("ReceiverConnection State has changed %s \n", connectionState.String())

				if connectionState == webrtc.ICEConnectionStateConnected {
					log.Println("Ctrl+C the remote client to stop the demo")
				} else if connectionState == webrtc.ICEConnectionStateFailed ||
					connectionState == webrtc.ICEConnectionStateDisconnected {

					closeErr := outFile.Close()
					if closeErr != nil {
						panic(closeErr)
					}

					log.Println("Done writing media files")
					os.Exit(0)
				}
			})

			// Set the remote SessionDescription
			if err = peerConnection.SetRemoteDescription(offer); err != nil {
				panic(err)
			}

			// Create answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				panic(err)
			}

			// Create channel that is blocked until ICE Gathering is complete
			gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

			// Sets the LocalDescription, and starts our UDP listeners
			if err = peerConnection.SetLocalDescription(answer); err != nil {
				panic(err)
			}

			// Block until ICE Gathering is complete, disabling trickle ICE
			// we do this because we only can exchange one signaling message
			// in a production application you should exchange ICE Candidates via OnICECandidate
			<-gatherComplete

			// Output the answer
			b, err := json.Marshal(*peerConnection.LocalDescription())
			if err != nil {
				panic(err)
			}
			ans <- string(b)

			// Block forever
			select {}
			log.Println("SHOULDNT GET HERE")
		}()

		msg := <-ans
		fmt.Fprintf(w, msg)
	})

	http.HandleFunc("/connectsender", func(w http.ResponseWriter, r *http.Request) {
		log.Println("PROXY GOING")
		resp, err := http.Post("http://localhost:9090/connectsender", "application/json", r.Body)
		if err != nil {
			print(err)
		}
		log.Println("PROXY RETURNING")

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			print(err)
		}
		fmt.Fprintf(w, string(body))
	})

	fs := http.FileServer(http.Dir("./www"))
	http.Handle("/", fs)

	if os.Getenv("ENV") == "development" {
		log.Println("IN DEVELOPMENT")
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("IN PRODUCTION")
		err := http.ListenAndServeTLS("0.0.0.0:443", "/etc/letsencrypt/live/show.dargervision.xyz/cert.pem", "/etc/letsencrypt/live/show.dargervision.xyz/privkey.pem", nil)
		if err != nil {
			panic(err)
		}
	}
	log.Println("END")
}

var videoRTCPFeedback = []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}

var (
	h264codecs = []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        102,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=102", nil},
			PayloadType:        121,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=127", nil},
			PayloadType:        120,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", videoRTCPFeedback},
			PayloadType:        125,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=125", nil},
			PayloadType:        107,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", videoRTCPFeedback},
			PayloadType:        108,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=108", nil},
			PayloadType:        109,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=127", nil},
			PayloadType:        120,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/H264", 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032", videoRTCPFeedback},
			PayloadType:        123,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=123", nil},
			PayloadType:        118,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/ulpfec", 90000, 0, "", nil},
			PayloadType:        116,
		},
	}

	vp8codecs = []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/VP8", 90000, 0, "", videoRTCPFeedback},
			PayloadType:        96,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=96", nil},
			PayloadType:        97,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/ulpfec", 90000, 0, "", nil},
			PayloadType:        116,
		},
	}

	vp9codecs = []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/VP9", 90000, 0, "profile-id=0", videoRTCPFeedback},
			PayloadType:        98,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=98", nil},
			PayloadType:        99,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/VP9", 90000, 0, "profile-id=1", videoRTCPFeedback},
			PayloadType:        100,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=100", nil},
			PayloadType:        101,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/ulpfec", 90000, 0, "", nil},
			PayloadType:        116,
		},
	}
)
