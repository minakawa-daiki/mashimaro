package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/castaneai/mashimaro/streamer"

	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/pion/webrtc/v2/pkg/media/ivfreader"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatalf("Usage: ./streamer <base64_encoded_SDP>")
	}
	offer, err := readOffer(os.Args[1])
	if err != nil {
		log.Fatalf("failed to read offer: %+v", err)
	}

	// We make our own mediaEngine so we can place the sender's codecs in it.  This because we must use the
	// dynamic media type from the sender in our answer. This is not required if we are the offerer
	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.PopulateFromSDP(*offer); err != nil {
		log.Fatalf("failed to populate from SDP: %+v", err)
	}

	// Search for VP8 Payload type. If the offer doesn't support VP8 exit since
	// since they won't be able to decode anything we send them
	var payloadType uint8
	for _, videoCodec := range mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
		if videoCodec.Name == "VP8" {
			payloadType = videoCodec.PayloadType
			break
		}
	}
	if payloadType == 0 {
		panic("Remote peer does not support VP8")
	}

	// Create a new RTCPeerConnection
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Create a video track
	videoTrack, err := peerConnection.NewTrack(payloadType, rand.Uint32(), "video", "pion")
	if err != nil {
		panic(err)
	}
	if _, err = peerConnection.AddTrack(videoTrack); err != nil {
		panic(err)
	}

	go func() {
		out, err := streamer.StartPollingRawOverTCP(1282, 747, 32)
		if err != nil {
			log.Printf("failed to start polling over TCP: %+v", err)
			return
		}
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			for {
				if _, err := pw.Write(<-out); err != nil {
					log.Printf("failed to write to pipe: %+v", err)
				}
			}
		}()
		_, header, err := ivfreader.NewWith(pr)
		if err != nil {
			log.Fatalf("failed to new ivf reader: %+v", err)
		}

		// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
		// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
		sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
		for {
			frame, err := readIVFFrame(pr)
			if err != nil {
				log.Fatalf("failed to read IVF frame: %+v", err)
			}

			time.Sleep(sleepTime)
			if err := videoTrack.WriteSample(media.Sample{Data: frame, Samples: 90000}); err != nil {
				panic(err)
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(*offer); err != nil {
		panic(err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Output the answer in base64 so we can paste it in browser
	ans, err := encodeAnswer(&answer)
	if err != nil {
		log.Fatalf("failed to encode answer: %+v", err)
	}
	fmt.Println(ans)

	// Block forever
	select {}
}

func readOffer(in string) (*webrtc.SessionDescription, error) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return nil, err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(b, &offer); err != nil {
		return nil, err
	}
	return &offer, nil
}

func encodeAnswer(answer *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(answer)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

const (
	ivfFrameHeaderSize = 12
)

func readIVFFrame(r io.Reader) ([]byte, error) {
	buffer := make([]byte, ivfFrameHeaderSize)
	var header *ivfreader.IVFFrameHeader

	if _, err := io.ReadFull(r, buffer); err != nil {
		return nil, err
	}
	header = &ivfreader.IVFFrameHeader{
		FrameSize: binary.LittleEndian.Uint32(buffer[:4]),
		Timestamp: binary.LittleEndian.Uint64(buffer[4:12]),
	}

	payload := make([]byte, header.FrameSize)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
