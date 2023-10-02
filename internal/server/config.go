package server

import (
	"encoding/json"
	"fmt"

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

func (c Config) Str() string {
	c.Datastore.Cockroach.Host = "***"
	c.Datastore.Cockroach.Password = "***"
	var str string
	if c.Environment == "local" {
		raw, _ := json.MarshalIndent(c, "", "  ")
		str = string(raw)
	} else {
		str = fmt.Sprintf("%+v", c)
	}
	return str
}
