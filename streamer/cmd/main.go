package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/castaneai/mashimaro/streamer"
	"github.com/pion/webrtc/v2/pkg/media"
	"log"
	"math/rand"
	"os"

	"github.com/pion/webrtc/v2"
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

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.PopulateFromSDP(*offer); err != nil {
		log.Fatalf("failed to populate from SDP: %+v", err)
	}

	var payloadType uint8
	for _, videoCodec := range mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
		if videoCodec.Name == webrtc.H264 {
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
		out, err := streamer.StartStreamingJPEGRTP(9999)
		if err != nil {
			log.Printf("failed to start polling over RTP: %+v", err)
			return
		}
		for data := range out {
			if err := videoTrack.WriteSample(media.Sample{
				Data:    data,
				Samples: 90000,
			}); err != nil {
				log.Printf("failed to write video sample: %+v", err)
			}
		}
	}()

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})
	if err = peerConnection.SetRemoteDescription(*offer); err != nil {
		panic(err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

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
