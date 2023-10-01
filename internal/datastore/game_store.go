package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/quibbble/go-boardgame/pkg/bgn"
)

var (
	ErrGameStoreNotEnabled = fmt.Errorf("game store is not enabled")
	ErrGameStoreNotFound   = fmt.Errorf("no game found in game store")
	ErrGameStoreConnection = fmt.Errorf("failed to connect to game store")
	ErrGameStoreSelect     = fmt.Errorf("failed to select from game store")
	ErrGameStoreInsert     = fmt.Errorf("failed to insert into game store")
)

type Game struct {
	GameKey   string    `json:"game_key"`
	GameID    string    `json:"game_id"`
	BGN       *bgn.Game `json:"bgn"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PlayCount int       `json:"play_count"` // multiple games could have been played under the same game id
}

type Stats struct {
	GamesPlayed    map[string]int `json:"games_played"`
	GamesCompleted map[string]int `json:"games_completed"`
}

// GameStore stores games into long term storage
type GameStore interface {
	GetGame(gameKey, gameID string) (*Game, error)
	GetStats(games []string) (*Stats, error)
	Store(game *Game) error
	Close(ctx context.Context) error
}
