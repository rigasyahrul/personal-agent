package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/app"
	"github.com/rigasyahrul/personal-agent/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	application, err := app.New(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("close application: %v", err)
		}
	}()

	log.Printf("listening on %s", cfg.Addr)
	err = http.ListenAndServe(cfg.Addr, application.Handler())
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
