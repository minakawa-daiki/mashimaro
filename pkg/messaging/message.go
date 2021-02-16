package messaging

import "encoding/json"

type MessageType string

const (
	MessageTypeMove      MessageType = "move"
	MessageTypeMouseDown MessageType = "mousedown"
	MessageTypeMouseUp   MessageType = "mouseup"
	MessageTypeKeyDown   MessageType = "keydown"
	MessageTypeKeyUp     MessageType = "keyup"
	MessageTypeExitGame  MessageType = "exitGame"
)

type Message struct {
	Type MessageType     `json:"type"`
	Body json.RawMessage `json:"body"`
}

type MoveMessage struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MouseDownMessage struct {
	Button int `json:"button"`
}

type MouseUpMessage struct {
	Button int `json:"button"`
}

type KeyDownMessage struct {
	Key int `json:"key"`
}

type KeyUpMessage struct {
	Key int `json:"key"`
}
