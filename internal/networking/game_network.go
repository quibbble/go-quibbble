package go_boardgame_networking

import (
	"context"
	"fmt"
	"strings"

	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/internal/datastore"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

type GameNetwork struct {
	hubs      map[string]*gameHub // mapping from game key to game hub
	gameStore datastore.GameStore
}

type GameStats struct {
	ActiveGames   map[string]int
	ActivePlayers map[string]int
}

func NewGameNetwork(options GameNetworkOptions, gameStore datastore.GameStore) *GameNetwork {
	hubs := make(map[string]*gameHub)
	for _, builder := range options.Games {
		hub := newGameHub(builder, options.GameExpiry, options.Adapters, gameStore)
		go hub.Start()
		hubs[builder.Key()] = hub
	}
	return &GameNetwork{
		hubs:      hubs,
		gameStore: gameStore,
	}
}

func (n *GameNetwork) CreateGame(options CreateGameOptions) error {
	gameKey, gameID := options.NetworkOptions.GameKey, options.NetworkOptions.GameID
	hub, ok := n.hubs[gameKey]
	if !ok {
		return fmt.Errorf("game key '%s' does not exist", gameKey)
	}
	if len(options.NetworkOptions.Players) > 0 && len(options.NetworkOptions.Players) != len(options.GameOptions.Teams) {
		return fmt.Errorf("number of teams are inconsistent")
	}
	if _, ok := hub.games[gameID]; !ok {
		if gameData, err := n.gameStore.GetGame(gameKey, gameID); err == nil {
			options.GameOptions = nil
			options.BGN = nil
			options.GameData = gameData
		}
	}
	return hub.Create(options)
}

func (n *GameNetwork) JoinGame(options JoinGameOptions) error {
	gameKey, gameID := options.GameKey, options.GameID
	hub, ok := n.hubs[gameKey]
	if !ok {
		return fmt.Errorf("game key '%s' does not exist", gameKey)
	}
	if _, ok := hub.games[gameID]; !ok {
		gameData, err := n.gameStore.GetGame(gameKey, gameID)
		if err != nil {
			return fmt.Errorf("game id '%s' does not exist", gameID)
		}
		if err := hub.Create(CreateGameOptions{
			NetworkOptions: &NetworkingCreateGameOptions{
				GameKey: gameKey,
				GameID:  gameID,
			},
			GameData: gameData,
		}); err != nil {
			return err
		}
	}
	return hub.Join(options)
}

func (n *GameNetwork) GetStats() *GameStats {
	stats := &GameStats{
		ActiveGames:   make(map[string]int),
		ActivePlayers: make(map[string]int),
	}
	for _, hub := range n.hubs {
		stats.ActiveGames[hub.builder.Key()] = len(hub.games)
		stats.ActivePlayers[hub.builder.Key()] = 0
		for _, game := range hub.games {
			stats.ActivePlayers[hub.builder.Key()] += len(game.players)
		}
	}
	return stats
}

func (n *GameNetwork) GetBGN(gameKey, gameID string) (*bgn.Game, error) {
	hub, ok := n.hubs[gameKey]
	if !ok {
		return nil, fmt.Errorf("game key '%s' does not exist", gameKey)
	}
	if _, ok := hub.games[gameID]; !ok {
		gameData, err := n.gameStore.GetGame(gameKey, gameID)
		if err != nil {
			return nil, fmt.Errorf("game id '%s' does not exist", gameID)
		}
		if err := hub.Create(CreateGameOptions{
			NetworkOptions: &NetworkingCreateGameOptions{
				GameKey: gameKey,
				GameID:  gameID,
			},
			GameData: gameData,
		}); err != nil {
			return nil, err
		}
	}
	return hub.games[gameID].game.GetBGN(), nil
}

func (n *GameNetwork) GetGames() []string {
	games := make([]string, 0)
	for _, hub := range n.hubs {
		games = append(games, hub.builder.Key())
	}
	return games
}

func (n *GameNetwork) GetGame(gameKey, gameID string, team ...string) (interface{}, error) {
	hub, ok := n.hubs[gameKey]
	if !ok {
		return nil, fmt.Errorf("game key '%s' does not exist", gameKey)
	}
	if _, ok := hub.games[gameID]; !ok {
		gameData, err := n.gameStore.GetGame(gameKey, gameID)
		if err != nil {
			return nil, fmt.Errorf("game id '%s' does not exist", gameID)
		}
		if err := hub.Create(CreateGameOptions{
			NetworkOptions: &NetworkingCreateGameOptions{
				GameKey: gameKey,
				GameID:  gameID,
			},
			GameData: gameData,
		}); err != nil {
			return nil, err
		}
	}
	return hub.games[gameID].game.GetSnapshot(team...)
}

func (n *GameNetwork) Close(ctx context.Context) error {
	games := make([]string, 0)
	for gameKey, hub := range n.hubs {
		errored := false
		if err := hub.Store(ctx); err != nil {
			logger.Log.Error().Caller().Err(err).Msgf("failed to store '%s' hub", gameKey)
			errored = true
		}
		if err := hub.Close(); err != nil {
			logger.Log.Error().Caller().Err(err).Msgf("failed to close '%s' hub", gameKey)
			errored = true
		}
		if errored {
			games = append(games, gameKey)
		}
	}
	if len(games) > 0 {
		return fmt.Errorf("game hubs %s failed to close gracefully", strings.Join(games, ", "))
	}
	return nil
}
