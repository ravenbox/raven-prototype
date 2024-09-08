package raven

import (
	"encoding/json"
)

type WebsocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WebsocketMessageType interface {
	MessageType() string
}

func Match[T WebsocketMessageType](m WebsocketMessage, fn func(T)) error {
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
