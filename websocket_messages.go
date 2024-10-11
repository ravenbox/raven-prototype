package raven

import (
	"bytes"
	"encoding/json"
)

type WebsocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func EncodeWebsocketMessage(payload WebsocketMessagePayload) (WebsocketMessage, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return WebsocketMessage{"", nil}, err
	}
	return WebsocketMessage{
		Type:    payload.MessageType(),
		Payload: nil,
	}, nil
}

type WebsocketMessagePayload interface {
	MessageType() string
}

func Match[T WebsocketMessagePayload](m WebsocketMessage, fn func(T)) error {
	t := *new(T)
	if m.Type != t.MessageType() {
		return nil
	}
	var parsed T
	if err := json.Unmarshal(m.Payload, &parsed); err != nil {
		return err
	}
	fn(parsed)
	return nil
}
