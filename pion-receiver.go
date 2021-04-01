package main

import (
	"encoding/json"
	"time"

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"


	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
)

func main() {

	http.HandleFunc("/connectreceiver", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)

		ans := make(chan string)

		go func() {

			offer := webrtc.SessionDescription{}
			json.Unmarshal(body, &offer)

			m := webrtc.MediaEngine{}
			// Setup the codecs you want to use.
			// We'll use H264 but you can also define your own
			if err := m.RegisterCodec(webrtc.RTPCodecParameters{
				RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/h264", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
				PayloadType:        102,
			}, webrtc.RTPCodecTypeVideo); err != nil {
				panic(err)
			}

			// Create the API object with the MediaEngine
			api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

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

			// iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

			// Create Track that we send video back to browser on
			outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "pion")
			if err != nil {
				panic(err)
			}

			// Add this newly created track to the PeerConnection
			rtpSender, err := peerConnection.AddTrack(outputTrack)
			if err != nil {
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

			outFile := h264writer.NewWith(os.Stdout)

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
				}
			})

			// Set the handler for ICE connection state
			// This will notify you when the peer has connected/disconnected
			peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				log.Println("Connection State has changed %s \n", connectionState.String())

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
