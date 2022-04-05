package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/infrastructure"
	"github.com/damejeras/hometask/internal/shootout"
	"github.com/go-redis/redis/v8"
)

const (
	competitorPubSub = "competitor_events"
)

var (
	ErrUnexpectedEvent = fmt.Errorf("unexpected event received")
	ErrNoTarget        = fmt.Errorf("no target found")
)

type Shooter struct {
	ID          string
	cfg         *app.ShooterConfig
	ctx         context.Context
	cancel      context.CancelFunc
	shotChan    chan *shootout.Shot
	redisClient *redis.Client
	logger      *log.Logger
}

func (s *Shooter) Run() {
	go s.dispatchShots()

	sub := s.redisClient.Subscribe(s.ctx, arbiterPubSub)

	for {
		select {
		case msg := <-sub.Channel():
			if err := s.handleArbiterMessage(msg); err != nil {
				s.logger.Printf("handle message from arbiter: %v", err)
				s.cancel()
			}
		case <-s.ctx.Done():
			if err := sub.Close(); err != nil {
				s.logger.Printf("close arbiter pub/sub: %v", err)
			}

			close(s.shotChan)

			return
		}
	}
}

func (s *Shooter) handleArbiterMessage(msg *redis.Message) error {
	var event infrastructure.Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	s.logger.Printf("event %q received", event.Type)

	switch event.Type {
	case infrastructure.TypeHeartbeat:
		if s.ID == "" {
			return s.register()
		}

		return nil
	case infrastructure.TypeRound:
		if s.ID == "" {
			return ErrUnexpectedEvent
		}

		var competitors map[string]*shootout.Competitor
		if err := json.Unmarshal(event.Data, &competitors); err != nil {
			return fmt.Errorf("unmarshal competitors: %w", err)
		}

		_, ok := competitors[s.ID]
		if len(competitors) == 1 && ok {
			fmt.Println("I WON")
			s.cancel()
			return nil
		}

		if !ok {
			fmt.Printf("IM DEAD")
			s.cancel()
			return nil
		}

		for target := range competitors {
			if target != s.ID {
				s.shotChan <- &shootout.Shot{
					From: s.ID,
					To:   target,
				}

				return nil
			}
		}

		return ErrNoTarget
	default:
		return fmt.Errorf("unknown event %q received", event.Type)
	}
}

func (s *Shooter) register() error {
	arbiterURL, err := url.Parse(s.cfg.ArbiterAddr)
	if err != nil {
		return fmt.Errorf("parse arbiter url: %w", err)
	}

	arbiterURL.Path = "/register"

	payload, err := json.Marshal(&registrationRequest{
		Name:   s.cfg.Name,
		Health: s.cfg.Health,
		Damage: s.cfg.Damage,
	})
	if err != nil {
		return fmt.Errorf("marshal registration request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, arbiterURL.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create registration request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send registration request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected registration response code %d", resp.StatusCode)
	}

	var competitor shootout.Competitor
	if err := json.NewDecoder(resp.Body).Decode(&competitor); err != nil {
		return fmt.Errorf("decode registration response: %w", err)
	}
	defer resp.Body.Close()

	s.ID = competitor.ID

	return nil
}

func (s *Shooter) dispatchShots() {
	for shot := range s.shotChan {
		event, err := infrastructure.NewEvent(infrastructure.TypeShot, shot)
		if err != nil {
			s.logger.Printf("create shot event: %v", err)
			s.cancel()
			return
		}

		payload, err := json.Marshal(event)
		if err != nil {
			s.logger.Printf("marshal shot event: %v", err)
			s.cancel()
			return
		}

		if err := s.redisClient.Publish(s.ctx, competitorPubSub, payload); err != nil {
			s.logger.Printf("publish shot event: %v", err)
			s.cancel()
			return
		}
	}
}