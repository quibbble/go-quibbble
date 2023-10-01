package http

import (
	"fmt"
	"net/http"

	"github.com/quibbble/go-quibbble/pkg/logger"
)

type Server struct {
	*http.Server
	port string
}

func NewServer(cfg ServerConfig, router http.Handler) *Server {
	return &Server{
		&http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: router,
		},
		cfg.Port,
	}
}

func (s *Server) Start(errCh chan<- error) {
	logger.Log.Info().Msg(fmt.Sprintf("server started on 0.0.0.0:%s", s.port))
	err := s.ListenAndServe()
	if err != http.ErrServerClosed {
		logger.Log.Error().Caller().Err(err).Msg("server stopped unexpectedly")
		errCh <- err
	} else {
		logger.Log.Info().Msg("server stopped")
	}
}
