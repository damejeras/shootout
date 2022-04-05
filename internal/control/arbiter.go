package control

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/infrastructure"
	"github.com/damejeras/hometask/internal/shootout"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	shutdownTimeout  = time.Minute
	arbiterPubSub    = "arbiter_events"
	competitorPubSub = "competitor_events"
)

type Arbiter struct {
	cfg         app.ArbiterConfig
	ctx         context.Context
	cancel      context.CancelFunc
	state       *shootout.State
	logger      *log.Logger
	redisClient *redis.Client
}

func (a *Arbiter) Run() {
	server := infrastructure.SingleEndpointHTTPServer(a.cfg.Port, "/register", a.handleRegistration)
	ticker := time.Tick(time.Second)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			a.logger.Printf("arbiter HTTP server listen: %v", err)
		}
	}()

	go func() {
		subscription := a.redisClient.Subscribe(a.ctx, competitorPubSub)
		for {
			select {
			case msg := <-subscription.Channel():
				a.handleMessage(msg)
			case <-a.ctx.Done():
				if err := subscription.Close(); err != nil {
					a.logger.Printf("close competitor events channel: %v", err)
				}

				return
			}
		}
	}()

	for {
		select {
		case <-ticker:
			a.beat()
		case <-a.ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.TODO(), shutdownTimeout)
			if err := server.Shutdown(shutdownCtx); err != nil {
				a.logger.Printf("arbiter HTTP server shutdown: %v", err)
			}

			cancel()
			return
		}
	}
}

func (a *Arbiter) handleMessage(msg *redis.Message) {
	var event infrastructure.Event
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		a.logger.Printf("unmarshal competitor event: %v", err)
		a.cancel()
		return
	}

	err := a.state.Handle(&event)
	if err != nil && err != shootout.ErrUnacceptablePayload {
		a.logger.Printf("handle competitor event: %v", err)
		a.cancel()
		return
	}

	if err != nil {
		a.logger.Printf("competitor event: %v", err)
	}
}

func (a *Arbiter) beat() {
	event, err := a.state.Emit()
	if err != nil {
		a.logger.Printf("emit event: %v", err)
		a.cancel()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		a.logger.Printf("marshal event: %v", err)
		a.cancel()
		return
	}

	if err := a.redisClient.Publish(a.ctx, arbiterPubSub, payload).Err(); err != nil {
		a.logger.Printf("publish event: %v", err)
		a.cancel()
		return
	}
}

func (a *Arbiter) handleRegistration(w http.ResponseWriter, r *http.Request) {
	var request registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.logger.Printf("decode request body: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if request.Name == "" || request.Health == 0 || request.Damage == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	competitor := shootout.Competitor{
		ID:     uuid.NewString(),
		Name:   request.Name,
		Health: request.Health,
		Damage: request.Damage,
	}

	event, err := infrastructure.NewEvent(infrastructure.TypeRegistration, &competitor)
	if err != nil {
		a.logger.Printf("create registration event: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err = a.state.Handle(event); err != nil {
		if err == shootout.ErrFinished || err == shootout.ErrAlreadyStarted {
			http.Error(w, "conflict", http.StatusConflict)
			return
		}

		a.logger.Printf("create registration event: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(competitor); err != nil {
		a.logger.Printf("encode registration response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

type registrationRequest struct {
	Name   string `json:"name"`
	Health int    `json:"health"`
	Damage int    `json:"damage"`
}
