package negotiation_test

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/ravenbox/raven-prototype/pkg/negotiation"
	"github.com/ravenbox/raven-prototype/pkg/utils"

	"github.com/pion/webrtc/v4"
)

func Test_BasicNegotiation(t *testing.T) {
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
	if err != nil {
		t.Fatal("Could not create data channel:", err)
	}

	fg := utils.FailGroup()
	fg.Check(func(fail, done func()) {
		dc.OnOpen(func() {
			defer done()
			err := dc.Send(testBuffer)
			if err != nil {
				t.Log("Error sending buffer:", err)
				fail()
			}
		})
	})

	fg.Check(func(fail, done func()) {
		neg2.PeerConn.OnDataChannel(func(dc *webrtc.DataChannel) {
			// Register the OnMessage to handle incoming messages
			dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
				defer done()
				if !slices.Equal(dcMsg.Data, testBuffer) {
					t.Log("received data but incorrect")
					fail()
				}
			})
		})
	})

	select {
	case passed := <-fg.Wait():
		if !passed {
			t.FailNow()
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("receive timeout.")
	}
}

func Test_BasicNegotiationWithNetworkLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	config := webrtc.Configuration{}
	pc1, err := webrtc.NewPeerConnection(config)
	if err != nil {
		t.Fatal("Failed to create webrtc PeerConnection:", err)
	}

	pc2, err := webrtc.NewPeerConnection(config)
	if err != nil {
		t.Fatal("Failed to create webrtc PeerConnection:", err)
	}

	randomLatency := func() time.Duration {
		const (
			maxLatency = 120 // ms
			minLatency = 80  // ms
		)
		return time.Duration(rand.Intn(maxLatency-minLatency)+minLatency) * time.Millisecond
	}

	sig1, sig2 := negotiation.DummySignalersPipeline(
		&negotiation.DummySignalerInterceptor{
			BeforeSend: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler1 send:", sb.Type())
				return nil
			},
			BeforeRecv: func(sb *negotiation.SignalBody) error {
				d := randomLatency()
				t.Logf("Signaler1 recv: %s\tBlocked for: %v", sb.Type(), d)
				<-time.After(d)
				return nil
			},
		},
		&negotiation.DummySignalerInterceptor{
			BeforeSend: func(sb *negotiation.SignalBody) error {
				t.Log("Signaler2 send:", sb.Type())
				return nil
			},
			BeforeRecv: func(sb *negotiation.SignalBody) error {
				d := randomLatency()
				t.Logf("Signaler2 recv: %s\tBlocked for: %v", sb.Type(), d)
				<-time.After(d)
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
	if err != nil {
		t.Fatal("Could not create data channel:", err)
	}

	fg := utils.FailGroup()
	fg.Check(func(fail, done func()) {
		dc.OnOpen(func() {
			defer done()
			err := dc.Send(testBuffer)
			if err != nil {
				t.Log("Error sending buffer:", err)
				fail()
			}
		})
	})

	fg.Check(func(fail, done func()) {
		neg2.PeerConn.OnDataChannel(func(dc *webrtc.DataChannel) {
			// Register the OnMessage to handle incoming messages
			dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
				defer done()
				if !slices.Equal(dcMsg.Data, testBuffer) {
					t.Log("received data but incorrect")
					fail()
				}
			})
		})
	})

	select {
	case passed := <-fg.Wait():
		if !passed {
			t.FailNow()
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receive timeout.")
	}
}

func Test_MultipleDataTracksNegotiation(t *testing.T) {
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

	fg := utils.FailGroup()

	ordered := true
	maxRetransmits := uint16(0)
	tt := map[uint16]struct {
		testBuffer           []byte
		waitBeforeNextCreate time.Duration
		recvCh               chan []byte
	}{
		1000: {testBuffer: []byte("sample buffer for test case 1"), waitBeforeNextCreate: 150 * time.Millisecond},
		1001: {testBuffer: []byte("sample buffer for test case 2"), waitBeforeNextCreate: 200 * time.Millisecond},
		1002: {testBuffer: []byte("sample buffer for test case 3")},
	}
	for k, v := range tt {
		v.recvCh = make(chan []byte, 1)
		tt[k] = v
	}

	neg2.PeerConn.OnDataChannel(func(dc *webrtc.DataChannel) {
		// Register the OnMessage to handle incoming messages
		dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
			id := dc.ID()
			if id == nil {
				t.Log("received message on data channel with empty id. ignored.")
				return
			}
			tt[*id].recvCh <- dcMsg.Data
		})
	})

	for id, tc := range tt {
		// Copy tc for closure captures
		id := id
		tc := tc

		options := &webrtc.DataChannelInit{
			ID:             &id,
			Ordered:        &ordered,
			MaxRetransmits: &maxRetransmits,
		}
		label := fmt.Sprintf("data-%d", id)
		dc, err := neg1.PeerConn.CreateDataChannel(label, options)
		if err != nil {
			t.Fatal("Could not create data channel:", err)
		}
		fg.Check(func(fail, done func()) {
			dc.OnOpen(func() {
				defer done()
				err := dc.Send(tc.testBuffer)
				if err != nil {
					t.Log("Error sending buffer:", err)
					fail()
				}
			})
		})

		// Check data arrival after creating data channel with
		// a specified timeout.
		fg.Check(func(fail, done func()) {
			go func() {
				defer done()
				const timeout = 1 * time.Second
				select {
				case recvBuffer := <-tc.recvCh:
					if !slices.Equal(recvBuffer, tc.testBuffer) {
						t.Log("received data but incorrect")
						fail()
					}
				case <-time.After(timeout):
					t.Log("receive timeout on test case", id)
					fail()
				}
			}()
		})

		if d := tc.waitBeforeNextCreate; d > 0 {
			<-time.After(tc.waitBeforeNextCreate)
		}
	}

	const timeout = 10 * time.Second
	select {
	case passed := <-fg.Wait():
		if !passed {
			t.FailNow()
		}
	case <-time.After(timeout):
		t.Fatal("test timeout")
	}
}
