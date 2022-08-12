package server

import (
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/quibbble/go-quibbble/pkg/http"
	pkgMiddleware "github.com/quibbble/go-quibbble/pkg/middleware"
	"github.com/rs/zerolog"
	"github.com/urfave/negroni"
)

func NewRouter(cfg http.RouterConfig, log zerolog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(pkgMiddleware.RequestLogger(log))
	r.Use(middleware.Timeout(time.Duration(cfg.TimeoutSec) * time.Second))
	r.Use(httprate.LimitAll(cfg.RequestPerSecLimit, time.Second))
	if !cfg.DisableCors {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.AllowedOrigins,
			AllowedMethods:   cfg.AllowedMethods,
			AllowedHeaders:   cfg.AllowedHeaders,
			AllowCredentials: true,
		}))
	}
	return r
}

func AddRoutes(r *chi.Mux, networkHandler *Handler) *chi.Mux {
	r.Route("/game", func(r chi.Router) {
		r.Post("/create", negroni.New(negroni.WrapFunc(networkHandler.CreateGame)).ServeHTTP)
		r.Post("/load", negroni.New(negroni.WrapFunc(networkHandler.LoadGame)).ServeHTTP)
		r.Get("/join", negroni.New(negroni.WrapFunc(networkHandler.JoinGame)).ServeHTTP)
		r.Get("/bgn", negroni.New(negroni.WrapFunc(networkHandler.GetBGN)).ServeHTTP)
		r.Get("/stats", negroni.New(negroni.WrapFunc(networkHandler.GetStats)).ServeHTTP)
	})
	r.Get("/health", negroni.New(negroni.WrapFunc(networkHandler.Health)).ServeHTTP)

	// add pprof
	r.Mount("/debug", middleware.Profiler())

	return r
}
