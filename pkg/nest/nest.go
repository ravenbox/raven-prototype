package nest

import (
	"fmt"
	"log"
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

func NewNest() *Nest {
	return &Nest{
		peers:         make(map[string]*webrtc.PeerConnection),
		inboundTracks: make(map[string]struct{}),
		subscribes:    make(map[string]map[string]chan *rtp.Packet),
	}
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
	if _, exists := n.subscribes[trackID]; !exists {
		n.subscribes[trackID] = make(map[string]chan *rtp.Packet)
	}
	if _, exists := n.subscribes[trackID][peerID]; !exists {
		n.subscribes[trackID][peerID] = make(chan *rtp.Packet)
	}
	n.subscribes[trackID][peerID] = ch
	go func() {
		for packet := range ch {
			log.Println("Writing", packet)
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
	fmt.Printf("Track has started, of type %d: %s \n", tr.PayloadType(), tr.Codec().MimeType)
	trackID := strings.Join([]string{
		peerID, tr.Codec().MimeType,
	}, "#")
	fmt.Println("Remote track id:", trackID)
	n.mu.Lock()
	n.inboundTracks[trackID] = struct{}{}
	n.mu.Unlock()
	for {
		packet, _, err := tr.ReadRTP()
		if err != nil {
			//TODO:
			panic(err)
		}
		log.Println("Readed", packet)

		for _, ch := range n.subscribes[trackID] {
			select {
			case ch <- packet:
			default:
			}
		}
	}
}
