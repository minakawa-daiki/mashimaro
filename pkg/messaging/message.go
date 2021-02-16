package messaging

import "encoding/json"

type MessageType string

const (
	MessageTypeMove     MessageType = "move"
	MessageTypeExitGame MessageType = "exitGame"
)

type Message struct {
	Type MessageType     `json:"type"`
	Body json.RawMessage `json:"body"`
}

type MoveMessage struct {
	X int `json:"x"`
	Y int `json:"y"`
}
