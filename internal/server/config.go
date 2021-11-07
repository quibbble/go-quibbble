package server

import (
	"github.com/quibbble/go-boardgame-networking/pkg/http"
	"github.com/quibbble/go-boardgame-networking/pkg/logger"
)

type Config struct {
	Environment string
	Log         logger.Config
	Router      http.RouterConfig
	Server      http.ServerConfig
	Network     NetworkOptions
}
