package nest

import (
	"strings"
	"sync"

	"github.com/ravenbox/raven-prototype/pkg/utils"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Nest struct {
	peers         map[string]*webrtc.PeerConnection
	inboundTracks map[string]struct{}
	subscribes    map[string]map[string]chan *rtp.Packet

	mu sync.RWMutex
}

func (n *Nest) RegisterPeer(id string, peer *webrtc.PeerConnection) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[id] = peer
	peer.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		n.onNewRemoteTrack(id, tr, r)
	})
}

func (n *Nest) SubscribeToTrack(peerID, trackID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	peer := n.peers[peerID]
	// TODO: codec stuff
	outboundTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		return err
	}
	rtpSender, err := peer.AddTrack(outboundTrack)
	if err != nil {
		return err
	}
	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	ch := make(chan *rtp.Packet, 32)
	n.subscribes[trackID][peerID] = ch
	go func() {
		for packet := range ch {
			if err := outboundTrack.WriteRTP(packet); err != nil {
				//TODO: wut?
				panic(err)
			}
		}
	}()

	return nil
}

func (n *Nest) PeerKeys() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return utils.MapKeys(n.peers)
}

func (n *Nest) Tracks() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return utils.MapKeys(n.inboundTracks)
}

func (n *Nest) onNewRemoteTrack(peerID string, tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	trackID := strings.Join([]string{
		peerID, tr.StreamID(), tr.ID(),
	}, "#")
	n.mu.Lock()
	n.inboundTracks[trackID] = struct{}{}
	n.mu.Unlock()
	for {
		packet, _, err := tr.ReadRTP()
		if err != nil {
			//TODO:
			panic(err)
		}

		for _, ch := range n.subscribes[trackID] {
			select {
			case ch <- packet:
			default:
			}
		}
	}
}
