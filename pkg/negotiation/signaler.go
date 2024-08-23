package negotiation

import (
	"github.com/pion/webrtc/v4"
)

type SignalBody struct {
	Candidate   *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Description *webrtc.SessionDescription `json:"description,omitempty"`
}

type Signaler interface {
	Send(SignalBody) error
	OnMessage(func(SignalBody))
	OnError(func(error))
	Close() error
}
