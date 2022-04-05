//go:build wireinject
// +build wireinject

package main

import (
	"log"

	"github.com/JeremyLoy/config"
	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/control"
	"github.com/damejeras/hometask/internal/shootout"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

func InitArbiter(cfg *app.ArbiterConfig) (*control.Arbiter, error) {
	wire.Build(
		log.Default,
		initRedisClient,
		control.NewArbiter,
		shootout.NewState,
	)

	return nil, nil
}

func initConfig() (*app.ArbiterConfig, error) {
	var cfg app.ArbiterConfig
	if err := config.FromEnv().To(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initRedisClient(cfg *app.ArbiterConfig) (*redis.Client, error) {
	wire.Build(
		redis.NewClient,
		initRedisConfig,
	)

	return nil, nil
}

func initRedisConfig(cfg *app.ArbiterConfig) *redis.Options {
	return &redis.Options{
		Addr: cfg.RedisAddr,
	}
}
