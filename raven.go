package raven

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/ravenbox/raven-prototype/pkg/negotiation"
	"github.com/ravenbox/raven-prototype/pkg/sfu"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Raven struct {
	SFU *sfu.SFU

	users map[string]*user
	mu    sync.Mutex
}

type UserRegisterRequest struct {
	Name string `json:"name"`
}

func (ra *Raven) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	var regReq UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading http to ws:", err)
		return
	}
	sendCh := make(chan WebsocketMessagePayload, 32)
	u := &user{
		ws:       conn,
		wsSendCh: sendCh,
	}
	ra.mu.Lock()
	ra.users[regReq.Name] = u
	ra.mu.Unlock()

	go u.readWs()
	go u.writeWs()
}

type user struct {
	ws       *websocket.Conn
	wsSendCh chan WebsocketMessagePayload

	webrtc     *webrtc.PeerConnection
	negotiator *negotiation.Negotiator
	signaler   *userSignaler
}

func (u *user) readWs() {
	defer u.ws.Close()
	u.ws.SetReadLimit(maxMessageSize)
	u.ws.SetReadDeadline(time.Now().Add(pongWait))
	u.ws.SetPongHandler(func(string) error {
		u.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		var msg WebsocketMessage
		err := u.ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("error:", err)
			}
			break
		}
		err = u.route(msg)
		if err != nil {
			log.Println("unknown message type")
		}
	}
}

func (u *user) writeWs() {
	readFrom := u.wsSendCh
	for msg := range readFrom {
		wsMsg, err := EncodeWebsocketMessage(msg)
		if err != nil {
			log.Println("error:", err)
			continue
		}
		err = u.ws.WriteJSON(wsMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("error:", err)
			}
			break
		}
	}
}

func (u *user) route(msg WebsocketMessage) error {
	if err := Match(msg, u.wsCreateWebRTCPeer); err != nil {
		return err
	}
	if err := Match(msg, u.wsGetSignal); err != nil {
		return err
	}
	return nil
}

type msgCreateWebRTCPeer struct{}

func (msgCreateWebRTCPeer) MessageType() string { return "create_webrtc_peer" }

func (u *user) wsCreateWebRTCPeer(_ msgCreateWebRTCPeer) {
	if u.webrtc == nil {
		pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			log.Println("error:", err)
			return
		}
		u.webrtc = pc
	}
	if u.signaler == nil {
		u.signaler = &userSignaler{
			sendChan: u.wsSendCh,
		}
	}
	if u.negotiator == nil {
		neg := negotiation.NewRegisteredNegotiator(
			u.webrtc, u.signaler, negotiation.Polite)
		u.negotiator = neg
	}
}

type msgSignal negotiation.SignalBody

func (msgSignal) MessageType() string { return "signal" }

func (u *user) wsGetSignal(msg msgSignal) {
	body := negotiation.SignalBody(msg)
	u.signaler.callOnMessage(body)
}

type userSignaler struct {
	sendChan chan WebsocketMessagePayload

	// these fields will be set by negotiator
	onMessageCallback func(negotiation.SignalBody)
	onErrorCallback   func(error)
}

var _ negotiation.Signaler = (*userSignaler)(nil)

func (s *userSignaler) Send(body negotiation.SignalBody) error {
	if s.sendChan == nil {
		return nil
	}
	s.sendChan <- msgSignal(body)
	return nil
}

func (s *userSignaler) OnMessage(fn func(negotiation.SignalBody)) {
	s.onMessageCallback = fn
}

func (s *userSignaler) OnError(fn func(error)) {
	s.onErrorCallback = fn
}

func (s *userSignaler) Close() error {
	panic("not implemented")
}

func (s *userSignaler) callOnMessage(body negotiation.SignalBody) {
	if s.onMessageCallback != nil {
		s.onMessageCallback(body)
	}
}

func (s *userSignaler) callOnError(err error) {
	if s.onErrorCallback != nil {
		s.onErrorCallback(err)
	}
}
