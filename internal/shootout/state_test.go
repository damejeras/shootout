package shootout

import (
	"encoding/json"
	"testing"

	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/infrastructure"
)

func TestState(t *testing.T) {
	state := NewState(&app.ArbiterConfig{
		Competitors: 2,
	})

	event, err := state.Emit()
	if err != nil {
		t.Fatalf("unexpected first event emission err: %v", err)
	}

	if event.Type != infrastructure.EventHeartbeat {
		t.Fatalf("first event emission should be of type %q, got %q", infrastructure.EventHeartbeat, event.Type)
	}

	firstRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_1",
		Name:   "Test1",
		Health: 3,
		Damage: 1,
	})
	if err := state.Handle(firstRegistration); err != nil {
		t.Fatalf("unexpected first registration err: %v", err)
	}

	secondRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_2",
		Name:   "Test2",
		Health: 1,
		Damage: 1,
	})
	if err := state.Handle(secondRegistration); err != nil {
		t.Fatalf("unexpected second registration err: %v", err)
	}

	event, err = state.Emit()
	if err != nil {
		t.Fatalf("unexpected second emission err: %v", err)
	}

	if event.Type != infrastructure.EventRound {
		t.Fatalf("expected second emission event type to be %q, got %q", infrastructure.EventRound, event.Type)
	}

	firstShotEvent, _ := infrastructure.NewEvent(infrastructure.EventShot, &Shot{
		From: "test_1",
		To:   "test_2",
	})
	if err := state.Handle(firstShotEvent); err != nil {
		t.Fatalf("unexpected first shot err: %v", err)
	}

	secondShot, _ := infrastructure.NewEvent(infrastructure.EventShot, &Shot{
		From: "test_2",
		To:   "test_1",
	})
	if err := state.Handle(secondShot); err != nil {
		t.Fatalf("unexpected second shot err: %v", err)
	}

	event, err = state.Emit()
	if err != nil {
		t.Fatalf("unexpected third emission err: %v", err)
	}

	var round Round
	if err := json.Unmarshal(event.Data, &round); err != nil {
		t.Fatalf("can not unmarshal round event: %v", err)
	}

	if len(round.Competitors) != 1 {
		t.Fatalf("expected to receive round event with 1 competitor, got %d", len(round.Competitors))
	}

	event, err = state.Emit()
	if err != ErrFinished {
		t.Fatalf("fourth emission expected to be ErrFinished, got: %v", err)
	}
}

func TestStateRegistration(t *testing.T) {
	state := NewState(&app.ArbiterConfig{
		Competitors: 1,
	})

	event, err := state.Emit()
	if err != nil {
		t.Fatalf("unexpected first event emission err: %v", err)
	}

	if event.Type != infrastructure.EventHeartbeat {
		t.Fatalf("first event emission should be of type %q, got %q", infrastructure.EventHeartbeat, event.Type)
	}

	firstRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_1",
		Name:   "Test1",
		Health: 3,
		Damage: 1,
	})
	if err := state.Handle(firstRegistration); err != nil {
		t.Fatalf("unexpected first registration err: %v", err)
	}

	event, err = state.Emit()
	if err != nil {
		t.Fatalf("unexpected second emission err: %v", err)
	}

	if event.Type != infrastructure.EventRound {
		t.Fatalf("expected second emission event type to be %q, got %q", infrastructure.EventRound, event.Type)
	}

	secondRegistration, err := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_2",
		Name:   "Test2",
		Health: 1,
		Damage: 1,
	})
	if err := state.Handle(secondRegistration); err != ErrAlreadyStarted {
		t.Fatalf("unexpected second registration err: %v", err)
	}
}

func TestStateInvalidRegistration(t *testing.T) {
	state := NewState(&app.ArbiterConfig{
		Competitors: 1,
	})

	event, err := state.Emit()
	if err != nil {
		t.Fatalf("unexpected first event emission err: %v", err)
	}

	if event.Type != infrastructure.EventHeartbeat {
		t.Fatalf("first event emission should be of type %q, got %q", infrastructure.EventHeartbeat, event.Type)
	}

	firstRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{})
	if err := state.Handle(firstRegistration); err != ErrInvalidRegistration {
		t.Fatalf("unexpected first registration err: %v", err)
	}
}

func TestStateShotEvent(t *testing.T) {
	state := NewState(&app.ArbiterConfig{
		Competitors: 2,
	})

	event, err := state.Emit()
	if err != nil {
		t.Fatalf("unexpected first event emission err: %v", err)
	}

	if event.Type != infrastructure.EventHeartbeat {
		t.Fatalf("first event emission should be of type %q, got %q", infrastructure.EventHeartbeat, event.Type)
	}

	shotEvent, _ := infrastructure.NewEvent(infrastructure.EventShot, &Shot{})
	if err := state.Handle(shotEvent); err != ErrNotStarted {
		t.Fatalf("expected not started error, got: %v", err)
	}

	firstRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_1",
		Name:   "Test1",
		Health: 3,
		Damage: 1,
	})
	if err := state.Handle(firstRegistration); err != nil {
		t.Fatalf("unexpected first registration err: %v", err)
	}

	secondRegistration, _ := infrastructure.NewEvent(infrastructure.EventRegistration, &Competitor{
		ID:     "test_2",
		Name:   "Test2",
		Health: 1,
		Damage: 1,
	})
	if err := state.Handle(secondRegistration); err != nil {
		t.Fatalf("unexpected second registration err: %v", err)
	}

	shotEvent, _ = infrastructure.NewEvent(infrastructure.EventShot, &Shot{})
	if err := state.Handle(shotEvent); err != ErrUnacceptablePayload {
		t.Fatalf("expected unacceptable payload eror, got: %v", err)
	}

	shotEvent, _ = infrastructure.NewEvent(infrastructure.EventShot, &Shot{
		From: "unexisting",
		To:   "test_1",
	})
	if err := state.Handle(shotEvent); err != nil {
		t.Fatalf("unexpected error when handling shot from unexisting shooter: %v", err)
	}

	event, err = state.Emit()
	if err != nil {
		t.Fatalf("unexpected err when emiting event after shot: %v", err)
	}

	if event.Type != infrastructure.EventRound {
		t.Fatalf("expected event type %q, got %q", infrastructure.EventRound, event.Type)
	}

	var round Round
	if err := json.Unmarshal(event.Data, &round); err != nil {
		t.Fatalf("unexpected error when unmarshaling round event: %v", err)
	}

	if round.Competitors["test_1"].Health != 3 || round.Competitors["test_2"].Health != 1 {
		t.Fatalf("unexpected change in competitors' health")
	}

	shotEvent, _ = infrastructure.NewEvent(infrastructure.EventShot, &Shot{
		From: "test_2",
		To:   "test_1",
	})
	if err := state.Handle(shotEvent); err != nil {
		t.Fatalf("unexpected error when handling proper shot: %v", err)
	}

	event, err = state.Emit()
	if err != nil {
		t.Fatalf("unexpected err when emiting event after shot: %v", err)
	}

	var round2 Round
	if err := json.Unmarshal(event.Data, &round2); err != nil {
		t.Fatalf("unexpected error when unmarshaling round event: %v", err)
	}

	if round2.Competitors["test_1"].Health != 2 || round2.Competitors["test_2"].Health != 1 {
		t.Fatalf("incorrect change in competitors' health")
	}

	shotEvent, _ = infrastructure.NewEvent(infrastructure.EventShot, &Shot{
		From: "test_2",
		To:   "nonexisting",
	})
	if err := state.Handle(shotEvent); err != nil {
		t.Fatalf("unexpected error when handling shot to nonexisting target: %v", err)
	}

	var round3 Round
	if err := json.Unmarshal(event.Data, &round3); err != nil {
		t.Fatalf("unexpected error when unmarshaling round event: %v", err)
	}

	if round2.Competitors["test_1"].Health != 2 || round2.Competitors["test_2"].Health != 1 {
		t.Fatalf("unexpected change in competitors' health")
	}
}
