package negotiation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pion/webrtc/v4"
)

type Signaler interface {
	Send(SignalBody) error
	OnMessage(func(SignalBody))
	OnError(func(error))
	Close() error
}

type SignalBody struct {
	Candidate   *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Description *webrtc.SessionDescription `json:"description,omitempty"`
}

func (s SignalBody) Type() string {
	o := []string{}
	if s.Candidate != nil {
		o = append(o, "ICECandidate")
	}
	if s.Description != nil {
		o = append(o, "Description:"+s.Description.Type.String())
	}
	if len(o) == 0 {
		o = append(o, "nil")
	}
	return fmt.Sprintf("<%s>", strings.Join(o, ","))
}

// DummySignalersPipeline is used only for test purposes.
// It creates an in-memory pipeline with two Signaler endpoint.
func DummySignalersPipeline(aInterceptor, bInterceptor *DummySignalerInterceptor) (a, b *dummySignalerPipelineEndpoint) {
	if aInterceptor == nil {
		aInterceptor = &DummySignalerInterceptor{}
	}
	if bInterceptor == nil {
		bInterceptor = &DummySignalerInterceptor{}
	}

	chA2B := make(chan SignalBody, 32)
	chB2A := make(chan SignalBody, 32)
	a = &dummySignalerPipelineEndpoint{
		DummySignalerInterceptor: *aInterceptor,
		output:                   chA2B,
		input:                    chB2A,
		err:                      make(chan error),
		closed:                   make(chan struct{}),
	}
	a.Start()
	b = &dummySignalerPipelineEndpoint{
		DummySignalerInterceptor: *bInterceptor,
		output:                   chB2A,
		input:                    chA2B,
		err:                      make(chan error),
		closed:                   make(chan struct{}),
	}
	b.Start()
	return
}

// DummySignalerInterceptor is a collection of callbacks for
// intercepting and manipulating SignalBody
type DummySignalerInterceptor struct {
	// BeforeSend blocks Send method. Use BeforeRecv for
	// simulating network latency and similar stuff.
	BeforeSend func(*SignalBody) error
	// BeforeRecv interrupts reading loop. It can be used to
	// simulate network latency and similar stuff.
	BeforeRecv func(*SignalBody) error
}

type dummySignalerPipelineEndpoint struct {
	DummySignalerInterceptor

	onMessageCallback func(SignalBody)
	onErrorCallback   func(error)

	output chan<- SignalBody
	input  <-chan SignalBody
	err    chan error
	closed chan struct{}
}

var _ Signaler = (*dummySignalerPipelineEndpoint)(nil)

func (d *dummySignalerPipelineEndpoint) Start() {
	if d.output == nil {
		panic("nil output chan")
	}
	if d.input == nil {
		panic("nil input chan")
	}
	if d.err == nil {
		panic("nil err chan")
	}

	// Recv loop:
	go func() {
		for body := range d.input {
			if d.BeforeRecv != nil {
				if err := d.BeforeRecv(&body); err != nil {
					d.FireError(err)
				}
			}
			if d.onMessageCallback != nil {
				d.onMessageCallback(body)
			}
		}
	}()

	// Error loop:
	go func() {
		for err := range d.err {
			if d.onErrorCallback != nil {
				d.onErrorCallback(err)
			}
		}
	}()
}

// FireError fires a custom error. It blocks until consument of
// error by OnError callback.
func (d *dummySignalerPipelineEndpoint) FireError(err error) {
	select {
	case <-d.closed:
	default:
		d.err <- err
	}
}

func (d *dummySignalerPipelineEndpoint) Send(body SignalBody) error {
	if d.BeforeSend != nil {
		err := d.BeforeSend(&body)
		if err != nil {
			return err
		}
	}
	select {
	case <-d.closed:
		return errors.New("send on closed signaler")
	default:
		d.output <- body
	}
	return nil
}

func (d *dummySignalerPipelineEndpoint) OnMessage(callback func(SignalBody)) {
	d.onMessageCallback = callback
}

func (d *dummySignalerPipelineEndpoint) OnError(callback func(error)) {
	d.onErrorCallback = callback
}

func (d *dummySignalerPipelineEndpoint) Close() error {
	close(d.closed)
	return nil
}

type SignalerCallbacks struct {
	onMessageCallback func(SignalBody)
	onErrorCallback   func(error)
}

func (s *SignalerCallbacks) OnMessage(callback func(SignalBody)) {
	s.onMessageCallback = callback
}

func (s *SignalerCallbacks) OnError(callback func(error)) {
	s.onErrorCallback = callback
}

type ChanSignaler interface {
	Signaler
	CallOnMessage(SignalBody)
	CallOnError(error)
}

type chanSignaler[T any] struct {
	SignalerCallbacks
	SignalChan  chan T
	transformer func(SignalBody) T
}

var _ Signaler = (*chanSignaler[struct{}])(nil)
var _ ChanSignaler = (*chanSignaler[struct{}])(nil)

func NewChanSignaler[T any](ch chan T, transformer func(SignalBody) T) ChanSignaler {
	if ch == nil {
		panic("ch must not be nil")
	}
	if transformer == nil {
		panic("transformer callback must not be nil")
	}
	return &chanSignaler[T]{
		SignalerCallbacks: SignalerCallbacks{},
		SignalChan:        ch,
		transformer:       transformer,
	}
}

func (s *chanSignaler[T]) Send(body SignalBody) error {
	s.SignalChan <- s.transformer(body)
	return nil
}

func (s *chanSignaler[T]) Close() error {
	close(s.SignalChan)
	return nil
}

func (s *chanSignaler[T]) CallOnMessage(body SignalBody) {
	if s.onMessageCallback != nil {
		s.onMessageCallback(body)
	}
}

func (s *chanSignaler[T]) CallOnError(err error) {
	if s.onErrorCallback != nil {
		s.onErrorCallback(err)
	}
}
