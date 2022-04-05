package main

import "log"

func main() {
	cfg, err := initConfig()
	if err != nil {
		log.Fatal(err)
	}

	arbiter, err := InitArbiter(cfg)
	if err != nil {
		log.Fatal(err)
	}

	arbiter.Run()
}
