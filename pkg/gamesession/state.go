package gamesession

import "fmt"

type State int

func (s State) String() string {
	switch s {
	case StateWaitingForSession:
		return "StateWaitingForSession"
	case StateSignaling:
		return "StateSignaling"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

const (
	StateUnknown State = iota
	StateWaitingForSession
	StateSignaling
	StateGameProvisioning
)
