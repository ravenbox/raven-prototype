package sfu

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/ravenbox/raven-prototype/pkg/utils"
)

type SFU struct {
	peers         map[*webrtc.PeerConnection]struct{}
	inboundTracks map[string]*inboundTrack
	mu            sync.RWMutex
}

type inboundTrack struct {
	subscribers map[*webrtc.PeerConnection]chan *rtp.Packet
	mu          sync.RWMutex
}

func NewSFU() *SFU {
	return &SFU{
		peers:         make(map[*webrtc.PeerConnection]struct{}),
		inboundTracks: make(map[string]*inboundTrack),
	}
}

func (n *SFU) RegisterPeer(peer *webrtc.PeerConnection) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[peer] = struct{}{}
	peer.OnTrack(n.newRemoteTrack)
}

func (n *SFU) Peers() []*webrtc.PeerConnection {
	n.mu.Lock()
	defer n.mu.Unlock()
	return utils.MapPointerKeys(n.peers)
}

func (n *SFU) Tracks() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return utils.MapKeys(n.inboundTracks)
}

func (n *SFU) Subscribe(peer *webrtc.PeerConnection, trackID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	_, exists := n.peers[peer]
	if !exists {
		return errors.New("peer not registered")
	}
	track, exists := n.inboundTracks[trackID]
	if !exists {
		return errors.New("tack does not exists")
	}

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
	track.mu.Lock()
	track.subscribers[peer] = ch
	track.mu.Unlock()
	go func() {
		for packet := range ch {
			if err := outboundTrack.WriteRTP(packet); err != nil {
				log.Println("Error writing RTP packet:", err)
				return
			}
		}
	}()

	return nil
}

func (n *SFU) newRemoteTrack(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
	trackID := fmt.Sprintf("%s#%s", tr.StreamID(), tr.ID())
	log.Printf("Track %s has started, of type %d: %s \n",
		trackID, tr.PayloadType(), tr.Codec().MimeType)
	track := &inboundTrack{
		subscribers: make(map[*webrtc.PeerConnection]chan *rtp.Packet),
	}
	n.mu.Lock()
	n.inboundTracks[trackID] = track
	n.mu.Unlock()

	for {
		packet, _, err := tr.ReadRTP()
		if err != nil {
			log.Println("Error reading RTP packet:", err)
			return
		}
		track.mu.RLock()
		subscribers := track.subscribers
		track.mu.RUnlock()
		for _, ch := range subscribers {
			select {
			case ch <- packet:
			default:
				// drop packet in case of congestion
			}
		}
	}
}
