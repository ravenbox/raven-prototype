// Implementation of WebRTC perfect negotiation pattern.
// See https://w3c.github.io/webrtc-pc/#perfect-negotiation-example
package negotiation

import (
	"github.com/ravenbox/raven-prototype/pkg/utils"

	"github.com/pion/webrtc/v4"
)

// Negotiator is created and registered per webrtc connections.
type Negotiator struct {
	PeerConn *webrtc.PeerConnection
	Signaler Signaler
	Polite   bool
	OnError  func(error)

	makingOffer                  bool
	ignoreOffer                  bool
	isSettingRemoteAnswerPending bool

	registered bool
	mu         utils.Mutex
}

type negotiatorOption func(*Negotiator)

var (
	Polite negotiatorOption = func(n *Negotiator) {
		n.Polite = true
	}
	OnError = func(fn func(error)) negotiatorOption {
		return func(n *Negotiator) {
			n.OnError = fn
		}
	}
)

func NewNegotiator(
	peerConn *webrtc.PeerConnection, signaler Signaler, opts ...negotiatorOption) *Negotiator {
	n := Negotiator{
		PeerConn: peerConn,
		Signaler: signaler,
	}
	for _, opt := range opts {
		opt(&n)
	}
	return &n
}

func NewRegisteredNegotiator(
	peerConn *webrtc.PeerConnection, signaler Signaler, opts ...negotiatorOption) *Negotiator {
	n := NewNegotiator(peerConn, signaler, opts...)
	n.Register()
	return n
}

func (n *Negotiator) Register() {
	if n.registered {
		panic("negotiator already registered")
	}
	if n.PeerConn == nil {
		panic("PeerConn could not be nil")
	}
	if n.Signaler == nil {
		panic("Signaler could not be nil")
	}
	n.registered = true

	n.PeerConn.OnICECandidate(n.onICECandidate)
	n.PeerConn.OnNegotiationNeeded(n.onNegotiationNeeded)
	n.Signaler.OnMessage(n.onMessage)
	n.Signaler.OnError(n.onSignalerError)
}

func (n *Negotiator) onICECandidate(c *webrtc.ICECandidate) {
	if c == nil {
		return
	}
	cInit := c.ToJSON()
	err := n.Signaler.Send(SignalBody{
		Candidate: &cInit,
	})
	if err != nil {
		n.handleError(err)
	}
}

func (n *Negotiator) onNegotiationNeeded() {
	n.mu.Tx(func() { n.makingOffer = true })
	offer, err := n.PeerConn.CreateOffer(nil)
	if err != nil {
		n.handleError(err)
		return
	}
	err = n.PeerConn.SetLocalDescription(offer)
	if err != nil {
		n.handleError(err)
		return
	}
	err = n.Signaler.Send(SignalBody{
		Description: n.PeerConn.LocalDescription(),
	})
	if err != nil {
		n.handleError(err)
		return
	}
	n.mu.Tx(func() { n.makingOffer = false })
	n.makingOffer = false
}

func (n *Negotiator) onMessage(s SignalBody) {
	if description := s.Description; description != nil {
		var (
			readyForOffer  bool
			offerCollision bool
		)
		n.mu.Tx(func() {
			readyForOffer = !n.makingOffer &&
				(n.PeerConn.SignalingState() == webrtc.SignalingStateStable ||
					n.isSettingRemoteAnswerPending)
			offerCollision = description.Type == webrtc.SDPTypeOffer && !readyForOffer
		})
		if ignore := !n.Polite && offerCollision; ignore {
			return
		}

		n.mu.Tx(func() {
			n.isSettingRemoteAnswerPending = description.Type == webrtc.SDPTypeAnswer
		})
		n.PeerConn.SetRemoteDescription(*description)
		n.mu.Tx(func() {
			n.isSettingRemoteAnswerPending = false
		})

		if description.Type == webrtc.SDPTypeOffer {
			answer, err := n.PeerConn.CreateAnswer(nil)
			if err != nil {
				n.handleError(err)
				return
			}
			err = n.PeerConn.SetLocalDescription(answer)
			if err != nil {
				n.handleError(err)
				return
			}
			err = n.Signaler.Send(SignalBody{
				Description: &answer,
			})
			if err != nil {
				n.handleError(err)
				return
			}
		}
	}
	if candidate := s.Candidate; candidate != nil {
		err := n.PeerConn.AddICECandidate(*candidate)
		if err != nil {
			n.handleError(err)
			return
		}
	}
}

func (n *Negotiator) onSignalerError(err error) {
	n.handleError(err)
}

func (n *Negotiator) handleError(err error) {
	if err == nil {
		return
	}
	if n.OnError != nil {
		n.OnError(err)
	}
}
