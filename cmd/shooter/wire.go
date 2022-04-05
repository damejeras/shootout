//go:build wireinject
// +build wireinject

package main

import (
	"log"

	"github.com/JeremyLoy/config"
	"github.com/damejeras/hometask/internal/app"
	"github.com/damejeras/hometask/internal/control"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

func InitShooter(cfg *app.ShooterConfig) (*control.Shooter, error) {
	wire.Build(
		log.Default,
		initRedisClient,
		control.NewShooter,
	)

	return nil, nil
}

func initConfig() (*app.ShooterConfig, error) {
	var cfg app.ShooterConfig
	if err := config.FromEnv().To(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func initRedisClient(cfg *app.ShooterConfig) (*redis.Client, error) {
	wire.Build(
		redis.NewClient,
		initRedisConfig,
	)

	return nil, nil
}

func initRedisConfig(cfg *app.ShooterConfig) *redis.Options {
	return &redis.Options{
		Addr: cfg.RedisAddr,
	}
}
