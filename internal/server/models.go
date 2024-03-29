package server

import (
	"time"

	networking "github.com/quibbble/go-quibbble/internal/networking"
)

var teams = []string{
	"red", "blue", "green", "yellow", "orange", "pink", "purple", "teal",
}

type NetworkOptions struct {
	Games      []string
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

type StatsResponse struct {
	GamesCreated  map[string]int
	GamesPlayed   map[string]int
	ActiveGames   map[string]int
	ActivePlayers map[string]int
}
