package main

import "log"

func main() {
	cfg, err := initConfig()
	if err != nil {
		log.Fatalf("init config: %v", err)
	}

	shooter, err := InitShooter(cfg)
	if err != nil {
		log.Fatalf("init shooter: %v", err)
	}

	shooter.Run()
}
