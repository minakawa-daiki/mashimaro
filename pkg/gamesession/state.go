package gamesession

type State string

const (
	StateUnknown           State = "Unknown"
	StateWaitingForSession State = "WaitingForSession"
	StateSignaling         State = "Signaling"
	StateGameProvisioning  State = "Provisioning"
)
