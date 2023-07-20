package server

import (
	"time"

	networking "github.com/quibbble/go-quibbble/internal/networking"
	"github.com/quibbble/go-quibbble/internal/networking/adapters"
)

var teams = []string{
	"red", "blue", "green", "yellow", "orange", "pink", "purple", "teal",
}

type NetworkOptions struct {
	Games      []string
	Adapters   Adapters
	GameExpiry time.Duration
}

type Adapters struct {
	RedisAdapter adapters.RedisAdapterConfig
}

type CreateGameRequest struct {
	*networking.NetworkingCreateGameOptions
	Teams       int
	MoreOptions interface{}
}

type LoadGameRequest struct {
	*networking.NetworkingCreateGameOptions
	BGN string
}
