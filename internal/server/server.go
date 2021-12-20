package server

import (
	"context"
	"os"
	"sync"
	"time"

	bg "github.com/quibbble/go-boardgame"
	networking "github.com/quibbble/go-boardgame-networking-internal"
	"github.com/quibbble/go-boardgame-networking-internal/pkg/http"
	"github.com/rs/zerolog"
	"github.com/unrolled/render"
)

type Server struct {
	cfg      Config
	log      zerolog.Logger
	server   *http.Server
	errCh    chan error
	shutdown sync.Once
}

func NewServer(cfg Config, log zerolog.Logger) (*Server, error) {
	g := make([]bg.BoardGameWithBGNBuilder, 0)
	a := make([]networking.NetworkAdapter, 0)
	for _, game := range cfg.Network.Games {
		g = append(g, games[game])
	}
	for _, adapter := range cfg.Network.Adapters {
		a = append(a, adapters[adapter])
	}
	network := networking.NewGameNetwork(networking.GameNetworkOptions{
		Games:      g,
		Adapters:   a,
		GameExpiry: cfg.Network.GameExpiry,
	})
	handler := NewHandler(log, render.New(), network)
	r := NewRouter(cfg.Router)
	r = AddRoutes(r, handler)
	return &Server{
		cfg:    cfg,
		log:    log,
		server: http.NewServer(cfg.Server, r, log),
		errCh:  make(chan error),
	}, nil
}

func (s *Server) Start() {
	go s.server.Start(s.errCh)
	for err := range s.errCh {
		if err != nil {
			s.log.Error().Caller().Err(err).Msg("fatal error")
			s.Shutdown(true)
		}
	}
}

func (s *Server) Shutdown(errored bool) {
	s.shutdown.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.log.Info().Msg("attempting graceful shutdown")
		graceful := make(chan bool)
		go func(graceful <-chan bool) {
			for {
				select {
				case <-ctx.Done():
					s.log.Panic().Msg("timeout so shutdown ungracefully")
				case <-graceful:
					return
				}
			}
		}(graceful)
		if err := s.server.Shutdown(ctx); err != nil {
			s.log.Error().Caller().Err(err).Msg("failed to shutdown server gracefully")
		}
		close(s.errCh)
		close(graceful)
		if errored {
			s.log.Info().Msg("shutdown gracefully but error detected")
			os.Exit(1)
		}
	})
}
