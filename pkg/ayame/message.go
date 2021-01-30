package ayame

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

type receivedMessage struct {
	Type    string `json:"type"`
	Payload []byte
}

func (j *receivedMessage) UnmarshalJSON(bytes []byte) error {
	var t struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(bytes, &t); err != nil {
		return err
	}
	j.Type = t.Type
	j.Payload = bytes
	return nil
}

type pingPongMessage struct {
	Type string `json:"type"`
}

type registerMessage struct {
	Type     string `json:"type"`
	RoomID   string `json:"roomId"`
	ClientID string `json:"clientId"`
}

type acceptMessage struct {
	Type          string `json:"type"`
	IsExistClient bool   `json:"isExistClient"`
}

type rejectMessage struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type candidateMessage struct {
	Type         string                   `json:"type"`
	ICECandidate *webrtc.ICECandidateInit `json:"ice,omitempty"`
}
