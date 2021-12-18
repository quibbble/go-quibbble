package server

import (
	"github.com/quibbble/go-boardgame-networking-internal/pkg/http"
	"github.com/quibbble/go-boardgame-networking-internal/pkg/logger"
)

type Config struct {
	Environment string
	Log         logger.Config
	Router      http.RouterConfig
	Server      http.ServerConfig
	Network     NetworkOptions
}
