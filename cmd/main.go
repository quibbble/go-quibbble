package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/quibbble/go-boardgame-networking-internal/internal/server"
	"github.com/quibbble/go-boardgame-networking-internal/pkg/config"
	"github.com/quibbble/go-boardgame-networking-internal/pkg/logger"
)

const (
	service = "network"
	prefix  = "NETWORK"
)

func main() {
	cfg := server.Config{}
	if err := config.NewConfig(service, prefix, &cfg); err != nil {
		panic(err)
	}

	// override port - for heroku deployment
	port := os.Getenv("PORT")
	if port != "" {
		cfg.Server.Port = port
	}

	log, err := logger.NewLogger(cfg.Log, cfg.Environment)
	if err != nil {
		panic(err)
	}

	log.Info().Msgf("%s service is starting", service)
	s, err := server.NewServer(cfg, log)
	if err != nil {
		panic(err)
	}
	go s.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)

	stopped := <-stop
	log.Info().Msg(fmt.Sprintf("%s signal received", stopped.String()))
	s.Shutdown(false)

	log.Info().Msgf("%s service has stopped", service)
}
