package main

import (
	"context"
	"io"
	"time"

	// "encoding/hex"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	// "time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
)

func main() {
	fmt.Println("START")
	outCodec := os.Getenv("OUT_CODEC")
	mimeType := "video/" + outCodec
	var payloadType webrtc.PayloadType
	if outCodec == "h264" {
		payloadType = 102
	} else if outCodec == "vp8" {
		payloadType = 96
	}

	// Create a MediaEngine object to configure the supported codec
	m := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: mimeType, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        payloadType,
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

	writeVP8 := func() {
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
	} else if outCodec == "vp8" {
		go writeVP8()
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())

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
