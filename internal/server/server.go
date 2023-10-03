package server

import (
	"context"
	"os"
	"sync"
	"time"

	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-quibbble/internal/datastore"
	networking "github.com/quibbble/go-quibbble/internal/networking"
	"github.com/quibbble/go-quibbble/pkg/http"
	"github.com/quibbble/go-quibbble/pkg/logger"
	"github.com/unrolled/render"
)

type Server struct {
	cfg      Config
	server   *http.Server
	network  *networking.GameNetwork
	errCh    chan error
	shutdown sync.Once
}

func NewServer(cfg Config) (*Server, error) {
	g := make([]bg.BoardGameBuilder, 0)
	a := make([]networking.NetworkAdapter, 0)
	for _, game := range cfg.Network.Games {
		g = append(g, games[game])
	}

	gameStore, err := datastore.NewCockroachClient(&cfg.Datastore.Cockroach)
	if err != nil {
		return nil, err
	}

	network := networking.NewGameNetwork(networking.GameNetworkOptions{
		Games:      g,
		Adapters:   a,
		GameExpiry: cfg.Network.GameExpiry,
		GameStore:  gameStore,
	})
	handler := NewHandler(render.New(), network, gameStore)
	r := NewRouter(cfg.Router)
	r = AddRoutes(r, handler)
	return &Server{
		cfg:     cfg,
		server:  http.NewServer(cfg.Server, r),
		network: network,
		errCh:   make(chan error),
	}, nil
}

func (s *Server) Start() {
	go s.server.Start(s.errCh)
	for err := range s.errCh {
		if err != nil {
			logger.Log.Error().Caller().Err(err).Msg("fatal error")
			s.Shutdown(true)
		}
	}
}

func (s *Server) Shutdown(errored bool) {
	s.shutdown.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Log.Info().Msg("attempting graceful shutdown")
		graceful := make(chan bool)
		go func(graceful <-chan bool) {
			for {
				select {
				case <-ctx.Done():
					logger.Log.Panic().Msg("timeout so shutdown ungracefully")
				case <-graceful:
					return
				}
			}
		}(graceful)
		if err := s.network.Close(ctx); err != nil {
			logger.Log.Error().Caller().Err(err).Msg("failed to close out games gracefully")
		} else {
			logger.Log.Info().Msg("closed all games gracefully")
		}
		if err := s.server.Shutdown(ctx); err != nil {
			logger.Log.Error().Caller().Err(err).Msg("failed to shutdown server gracefully")
		} else {
			logger.Log.Info().Msg("closed the server gracefully")
		}
		close(s.errCh)
		close(graceful)
		if errored {
			logger.Log.Info().Msg("shutdown gracefully but error detected")
			os.Exit(1)
		}
	})
}
