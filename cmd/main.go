package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/quibbble/go-quibbble/internal/server"
	"github.com/quibbble/go-quibbble/pkg/config"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

const service = "quibbble"

func main() {
	cfg := server.Config{}
	if err := config.NewConfig(service, strings.ToUpper(service), &cfg); err != nil {
		panic(err)
	}

	log, err := logger.NewLogger(cfg.Log, cfg.Environment)
	if err != nil {
		panic(err)
	}
	logger.Log = log

	logger.Log.Info().Msgf("%s service is starting with config %+v", service, cfg.Str())
	s, err := server.NewServer(cfg)
	if err != nil {
		panic(err)
	}
	go s.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)

	stopped := <-stop
	logger.Log.Info().Msg(fmt.Sprintf("%s signal received", stopped.String()))
	s.Shutdown(false)

	logger.Log.Info().Msgf("%s service has stopped", service)
}
