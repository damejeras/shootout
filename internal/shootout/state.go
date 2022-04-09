package shootout

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/damejeras/shootout/internal/app"
	"github.com/damejeras/shootout/internal/infrastructure"
)

var (
	ErrNotStarted          = fmt.Errorf("shootout not started yet")
	ErrAlreadyStarted      = fmt.Errorf("shootout already started")
	ErrUnacceptablePayload = fmt.Errorf("unacceptable payload")
	ErrFinished            = fmt.Errorf("shootout is finished")
	ErrInvalidRegistration = fmt.Errorf("invalid registration event")
)

type State struct {
	started, finished bool
	playerNumber      int

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
		return infrastructure.NewEvent(infrastructure.EventHeartbeat, nil)
	}

	if len(s.competitors) == 1 {
		s.finished = true
	}

	return infrastructure.NewEvent(infrastructure.EventRound, &Round{
		Competitors: s.competitors,
	})
}

func (s *State) Handle(event *infrastructure.Event) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	switch event.Type {
	case infrastructure.EventRegistration:
		return s.handleRegistration(event)
	case infrastructure.EventShot:
		return s.handleShot(event)
	default:
		return nil
	}
}

func (s *State) handleRegistration(event *infrastructure.Event) error {
	if s.started {
		return ErrAlreadyStarted
	}

	var competitor Competitor
	if err := json.Unmarshal(event.Data, &competitor); err != nil {
		return fmt.Errorf("unmarshal registration payload: %w", err)
	}

	if competitor.IsZero() {
		return ErrInvalidRegistration
	}

	s.competitors[competitor.ID] = &competitor

	if len(s.competitors) == s.playerNumber {
		s.started = true
	}

	return nil
}

func (s *State) handleShot(event *infrastructure.Event) error {
	if !s.started {
		return ErrNotStarted
	}

	var shot Shot
	if err := json.Unmarshal(event.Data, &shot); err != nil {
		return fmt.Errorf("unmarshal shot payload: %w", err)
	}

	if shot.From == "" || shot.To == "" {
		return ErrUnacceptablePayload
	}

	_, ok := s.competitors[shot.From]
	if !ok {
		// too late, sorry
		return nil
	}

	_, ok = s.competitors[shot.To]
	if !ok {
		// too late, sorry
		return nil
	}

	s.competitors[shot.To].Health -= s.competitors[shot.From].Damage

	log.Printf(
		"🔫 %s inflicted %d damage for %s 🔫",
		s.competitors[shot.From].Name,
		s.competitors[shot.From].Damage,
		s.competitors[shot.To].Name,
	)

	if s.competitors[shot.To].Health < 1 {
		delete(s.competitors, shot.To)
	}

	return nil
}
