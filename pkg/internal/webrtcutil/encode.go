package webrtcutil

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

func EncodeSDP(sdp *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(sdp)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func DecodeSDP(encoded string) (*webrtc.SessionDescription, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal(b, &offer); err != nil {
		return nil, err
	}
	return &offer, nil
}

func EncodeICECandidate(candidate *webrtc.ICECandidateInit) (string, error) {
	b, err := json.Marshal(candidate)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeICECandidate(s string) (*webrtc.ICECandidateInit, error) {
	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal([]byte(s), &candidate); err != nil {
		return nil, err
	}
	return &candidate, nil
}
