package infrastructure

import (
	"encoding/json"
	"fmt"
)

const (
	TypeHeartbeat    EventType = "heartbeat"
	TypeRegistration           = "registration"
	TypeRound                  = "round"
	TypeShot                   = "shot"
)

type EventType string

type Event struct {
	Type EventType       `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

func NewEvent(eventType EventType, data interface{}) (*Event, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %v", err)
	}

	return &Event{
		Type: eventType,
		Data: payload,
	}, nil
}
