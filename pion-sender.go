package main

import (
	"context"
	"io"
	"time"

	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
)

func main() {
	outCodec := os.Getenv("OUT_CODEC")
	mimeType := "video/" + outCodec
	var availableCodecs []webrtc.RTPCodecParameters
	if outCodec == "h264" {
		availableCodecs = h264codecs
	} else if outCodec == "vp8" {
		availableCodecs = vp8codecs
	} else if outCodec == "vp9" {
		availableCodecs = vp9codecs
	}

	// Create a MediaEngine object to configure the supported codec
	m := &webrtc.MediaEngine{}

	// Setup the codecs you want to use.
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

	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	// Create Track that we send video back to browser on
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: mimeType}, "video", "pion")
	if err != nil {
		panic(err)
	}

	// Add this newly created track to the PeerConnection
	rtpSender, err := peerConnection.AddTrack(videoTrack)
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
				// } else {
				// fmt.Printf("RTCP! %d %s\n", n, hex.EncodeToString(rtcpBuf))
			}
		}
	}()

	writeH264 := func() {
		reader, readErr := h264reader.NewReader(os.Stdin)
		if readErr != nil {
			panic(readErr)
		}

		// Wait for connection established

		for {
			nal, readErr := reader.NextNAL()
			if readErr == io.EOF {
				fmt.Printf("All video frames parsed and sent")
				os.Exit(0)
			}

			if readErr != nil {
				panic(readErr)
			}

			select {
			case <-iceConnectedCtx.Done():
				if writeErr := videoTrack.WriteSample(media.Sample{Data: nal.Data, Duration: time.Second}); writeErr != nil {
					panic(writeErr)
				}
			case <-time.After(time.Millisecond):
				continue
			}

		}
	}

	writeVP := func() {
		reader, _, readErr := ivfreader.NewWith(os.Stdin)
		if readErr != nil {
			panic(readErr)
		}

		// Wait for connection established

		for {
			frame, _, readErr := reader.ParseNextFrame()
			if readErr == io.EOF {
				fmt.Printf("All video frames parsed and sent")
				os.Exit(0)
			}

			if readErr != nil {
				panic(readErr)
			}

			select {
			case <-iceConnectedCtx.Done():
				if writeErr := videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); writeErr != nil {
					panic(writeErr)
				}
			case <-time.After(time.Millisecond):
				continue
			}

		}
	}

	if outCodec == "h264" {
		go writeH264()
	} else if outCodec == "vp8" || outCodec == "vp9" {
		go writeVP()
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Sender Connection State has changed %s \n", connectionState.String())

		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
			log.Println("Ctrl+C the remote client to stop the demo")

		} else if connectionState == webrtc.ICEConnectionStateFailed ||
			connectionState == webrtc.ICEConnectionStateDisconnected {

			// closeErr := ivfFile.Close()
			// if closeErr != nil {
			// 	panic(closeErr)
			// }

			log.Println("Done")
			os.Exit(0)
		}
	})

	http.HandleFunc("/connectsender", func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("CONNECT REQUEST")
		body, _ := ioutil.ReadAll(r.Body)

		ans := make(chan string)

		go func() {
			offer := webrtc.SessionDescription{}
			json.Unmarshal(body, &offer)

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
			// fmt.Println("Answering...")

			// fmt.Println(string(b))
			ans <- string(b)

			// Block forever
			select {}
			fmt.Println("SHOULDNT GET HERE")
		}()

		msg := <-ans
		fmt.Fprintf(w, msg)
	})

	httpErr := http.ListenAndServe("0.0.0.0:9090", nil)
	if httpErr != nil {
		log.Fatal(httpErr)
	}
	fmt.Println("END")

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
