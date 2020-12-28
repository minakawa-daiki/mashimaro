package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/castaneai/mashimaro/pkg/p2p"

	"github.com/pion/webrtc/v3"
)

func main() {
	mgr := p2p.NewManager()
	ctx := context.Background()
	go mgr.Start(ctx)

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		rb, err := ioutil.ReadAll(r.Body)
		if err != nil {
			respondError(w, err)
			return
		}
		offer, err := decodeOffer(string(rb))
		if err != nil {
			respondError(w, err)
			return
		}
		pc, err := newPeerConnection(offer)
		if err != nil {
			respondError(w, err)
			return
		}
		peer, err := p2p.NewPeer(pc)
		if err != nil {
			respondError(w, err)
			return
		}
		mgr.AddPeer(peer)

		answer, err := createAnswer(pc, offer)
		if err != nil {
			respondError(w, err)
			return
		}
		resb, err := encodeAnswer(answer)
		if err != nil {
			respondError(w, err)
			return
		}
		if _, err := w.Write([]byte(resb)); err != nil {
			respondError(w, err)
			return
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("static/index.html")
		if err != nil {
			respondError(w, err)
			return
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("content-type", "text/html")
		if _, err := w.Write(b); err != nil {
			respondError(w, err)
			return
		}
	})

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		pi, err := strconv.Atoi(p)
		if err != nil {
			log.Fatalf("failed to atoi port number: %+v", err)
		}
		port = pi
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("http server listening on %s...", addr)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func respondError(w http.ResponseWriter, err error) {
	log.Printf("error: %+v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func createAnswer(pc *webrtc.PeerConnection, offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	log.Printf("new peer connected")
	if err := pc.SetRemoteDescription(*offer); err != nil {
		return nil, fmt.Errorf("failed to set remote desc: %+v", err)
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create answer: %+v", err)
	}
	log.Printf("create answer")
	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, fmt.Errorf("failed to local desc: %+v", err)
	}
	return &answer, nil
}

func newPeerConnection(offer *webrtc.SessionDescription) (*webrtc.PeerConnection, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, err
	}
	return pc, nil
}

func decodeOffer(in string) (*webrtc.SessionDescription, error) {
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
