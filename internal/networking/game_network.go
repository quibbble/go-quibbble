package go_boardgame_networking

import (
	"fmt"
	"strings"

	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/rs/zerolog"
)

type GameNetwork struct {
	hubs map[string]*gameHub // mapping from game key to game hub
	log  zerolog.Logger
}

type GameStats struct {
	CurrentGameCount   map[string]int
	CurrentPlayerCount map[string]int
}

func NewGameNetwork(options GameNetworkOptions, log zerolog.Logger) *GameNetwork {
	hubs := make(map[string]*gameHub)
	for _, builder := range options.Games {
		hub := newGameHub(builder, options.GameExpiry, options.Adapters, log)
		go hub.Start()
		hubs[strings.ToLower(builder.Key())] = hub
	}
	return &GameNetwork{
		hubs: hubs,
		log:  log,
	}
}

func (n *GameNetwork) CreateGame(options CreateGameOptions) error {
	hub, ok := n.hubs[strings.ToLower(options.NetworkOptions.GameKey)]
	if !ok {
		return fmt.Errorf("game key '%s' does not exist", options.NetworkOptions.GameKey)
	}
	if len(options.NetworkOptions.Players) > 0 && len(options.NetworkOptions.Players) != len(options.GameOptions.Teams) {
		return fmt.Errorf("number of teams are inconsistent")
	}
	return hub.Create(options)
}

func (n *GameNetwork) LoadGame(options LoadGameOptions) error {
	hub, ok := n.hubs[strings.ToLower(options.NetworkOptions.GameKey)]
	if !ok {
		return fmt.Errorf("game key '%s' does not exist", options.NetworkOptions.GameKey)
	}
	if len(options.NetworkOptions.Players) > 0 && len(options.NetworkOptions.Players) != len(strings.Split(options.BGN.Tags["Teams"], ", ")) {
		return fmt.Errorf("number of teams are inconsistent")
	}
	return hub.Load(options)
}

func (n *GameNetwork) JoinGame(options JoinGameOptions) error {
	hub, ok := n.hubs[strings.ToLower(options.GameKey)]
	if !ok {
		return fmt.Errorf("game key '%s' does not exist", options.GameKey)
	}
	return hub.Join(options)
}

func (n *GameNetwork) GetStats() *GameStats {
	stats := &GameStats{
		CurrentGameCount:   make(map[string]int),
		CurrentPlayerCount: make(map[string]int),
	}
	for _, hub := range n.hubs {
		stats.CurrentGameCount[hub.builder.Key()] = len(hub.games)
		stats.CurrentPlayerCount[hub.builder.Key()] = 0
		for _, game := range hub.games {
			stats.CurrentPlayerCount[hub.builder.Key()] += len(game.players)
		}
	}
	return stats
}

func (n *GameNetwork) GetBGN(gameKey, gameID string) (*bgn.Game, error) {
	hub, ok := n.hubs[strings.ToLower(gameKey)]
	if !ok {
		return nil, fmt.Errorf("game key '%s' does not exist", gameKey)
	}
	server, ok := hub.games[gameID]
	if !ok {
		return nil, fmt.Errorf("game id '%s' does not exists", gameID)
	}
	return server.game.GetBGN(), nil
}

func (n *GameNetwork) GetGames() []string {
	games := make([]string, 0)
	for _, hub := range n.hubs {
		games = append(games, hub.builder.Key())
	}
	return games
}
