package main

import "log"

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

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
