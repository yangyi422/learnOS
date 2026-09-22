package main

import (
	"errors"
	"log"

	"learnos/internal/app"
	"learnos/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		if errors.Is(err, app.ErrRestartAfterRestore) {
			log.Print("database restore requested; restart required")
			return
		}
		log.Fatalf("run application: %v", err)
	}
}
