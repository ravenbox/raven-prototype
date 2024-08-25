package negotiation_test

import (
	"slices"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/ravenbox/raven-prototype/pkg/negotiation"
)

func Test_BasicNegotiation(t *testing.T) {
	t.Parallel()

	config := webrtc.Configuration{}
	pc1, err := webrtc.NewPeerConnection(config)
	if err != nil {
		t.Fatal("Failed to create webrtc PeerConnection:", err)
	}

	pc2, err := webrtc.NewPeerConnection(config)
	if err != nil {
		t.Fatal("Failed to create webrtc PeerConnection:", err)
	}

	sig1, sig2 := negotiation.DummySignalersPipeline(
		&negotiation.DummySignalerInterceptor{
			BeforeSend: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler1 send:", sb.Type())
				return nil
			},
			BeforeRecv: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler1 recv:", sb.Type())
				return nil
			},
		},
		&negotiation.DummySignalerInterceptor{
			BeforeSend: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler2 send:", sb.Type())
				return nil
			},
			BeforeRecv: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler2 recv:", sb.Type())
				return nil
			},
		},
	)

	neg1 := negotiation.NewRegisteredNegotiator(pc1, sig1, negotiation.Polite)
	defer neg1.Close()
	neg2 := negotiation.NewRegisteredNegotiator(pc2, sig2)
	defer neg2.Close()

	testBuffer := []byte("A buffer for test")

	ordered := false
	maxRetransmits := uint16(0)
	options := &webrtc.DataChannelInit{
		Ordered:        &ordered,
		MaxRetransmits: &maxRetransmits,
	}
	dc, err := neg1.PeerConn.CreateDataChannel("data", options)
	dc.OnOpen(func() {
		err := dc.Send(testBuffer)
		if err != nil {
			t.Fatal("Error sending buffer:", err)
		}
	})

	recvCh := make(chan []byte)
	neg2.PeerConn.OnDataChannel(func(dc *webrtc.DataChannel) {
		// Register the OnMessage to handle incoming messages
		dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
			recvCh <- dcMsg.Data
		})
	})

	select {
	case recvBuffer := <-recvCh:
		if !slices.Equal(recvBuffer, testBuffer) {
			t.Fatal("received data but incorrect")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("receive timeout.")
	}
}
