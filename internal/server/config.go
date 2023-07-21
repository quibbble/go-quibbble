package server

import (
	"github.com/quibbble/go-quibbble/internal/datastore"
	"github.com/quibbble/go-quibbble/pkg/http"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

type Config struct {
	Environment string
	Log         logger.Config
	Router      http.RouterConfig
	Server      http.ServerConfig
	Datastore   datastore.DatastoreConfig
	Network     NetworkOptions
}
