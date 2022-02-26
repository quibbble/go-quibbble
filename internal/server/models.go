package server

import (
	"time"

	networking "github.com/quibbble/go-quibbble"
)

var teams = []string{
	"red", "blue", "green", "yellow", "orange", "pink", "purple", "teal",
}

type NetworkOptions struct {
	Games      []string
	Adapters   []string
	GameExpiry time.Duration
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
