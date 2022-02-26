package http

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

type Server struct {
	*http.Server
	port string
	log  zerolog.Logger
}

func NewServer(cfg ServerConfig, router http.Handler, log zerolog.Logger) *Server {
	return &Server{
		&http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: router,
		},
		cfg.Port,
		log,
	}
}

func (s *Server) Start(errCh chan<- error) {
	s.log.Info().Msg(fmt.Sprintf("server started on 0.0.0.0:%s", s.port))
	err := s.ListenAndServe()
	if err != http.ErrServerClosed {
		s.log.Error().Caller().Err(err).Msg("server stopped unexpectedly")
		errCh <- err
	} else {
		s.log.Info().Msg("server stopped")
	}
}
