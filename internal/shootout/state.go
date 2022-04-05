package shootout

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/infrastructure"
)

var (
	ErrAlreadyStarted      = fmt.Errorf("shootout already started")
	ErrUnacceptablePayload = fmt.Errorf("unacceptable payload")
	ErrFinished            = fmt.Errorf("shootout is finished")
)

type State struct {
	started, finished   bool
	playerNumber, round int

	competitors map[string]*Competitor
	lock        *sync.Mutex
}

func NewState(cfg *app.ArbiterConfig) *State {
	return &State{
		playerNumber: cfg.Competitors,
		competitors:  make(map[string]*Competitor),
		lock:         new(sync.Mutex),
	}
}

func (s *State) Emit() (*infrastructure.Event, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.finished {
		return nil, ErrFinished
	}

	if !s.started {
		return infrastructure.NewEvent(infrastructure.TypeHeartbeat, nil)
	}

	s.round++

	if len(s.competitors) == 1 {
		s.finished = true
	}

	return infrastructure.NewEvent(infrastructure.TypeRound, &Round{
		ID:          s.round,
		Competitors: s.competitors,
	})
}

func (s *State) Handle(event *infrastructure.Event) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch event.Type {
	case infrastructure.TypeRegistration:
		return s.handleRegistration(event)
	case infrastructure.TypeShot:
		return s.handleShot(event)
	default:
		return nil
	}
}

func (s *State) handleRegistration(event *infrastructure.Event) error {
	if s.started {
		return ErrAlreadyStarted
	}

	if s.finished {
		return ErrFinished
	}

	var competitor Competitor
	if err := json.Unmarshal(event.Data, &competitor); err != nil {
		return fmt.Errorf("unmarshal registration payload: %w", err)
	}

	s.competitors[competitor.ID] = &competitor

	if len(s.competitors) == s.playerNumber {
		s.started = true
	}

	return nil
}

func (s *State) handleShot(event *infrastructure.Event) error {
	var shot Shot
	if err := json.Unmarshal(event.Data, &shot); err != nil {
		return fmt.Errorf("unmarshal shot payload: %w", err)
	}

	if shot.From == "" || shot.To == "" {
		return ErrUnacceptablePayload
	}

	_, ok := s.competitors[shot.From]
	if !ok {
		return ErrUnacceptablePayload
	}

	_, ok = s.competitors[shot.To]
	if !ok {
		return ErrUnacceptablePayload
	}

	s.competitors[shot.To].Health = s.competitors[shot.From].Damage

	if s.competitors[shot.To].Health < 1 {
		delete(s.competitors, shot.To)
	}

	return nil
}
