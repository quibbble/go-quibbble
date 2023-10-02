package go_boardgame_networking

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-quibbble/internal/datastore"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

// gameHub is a hub for a unique game type i.e. only for connect4 or only for tsuro
type gameHub struct {
	gameStore  datastore.GameStore
	builder    bg.BoardGameWithBGNBuilder
	games      map[string]*gameServer // mapping from game ID to game server
	cleanup    map[string]chan bool   // mapping from game ID to game's done channel that should be closed on game cleanup
	create     chan CreateGameOptions
	join       chan JoinGameOptions
	close      chan string
	done       chan error
	gameExpiry time.Duration
	adapters   []NetworkAdapter
}

func newGameHub(builder bg.BoardGameWithBGNBuilder, gameExpiry time.Duration, adapters []NetworkAdapter, gameStore datastore.GameStore) *gameHub {
	return &gameHub{
		gameStore:  gameStore,
		builder:    builder,
		games:      make(map[string]*gameServer),
		cleanup:    make(map[string]chan bool),
		create:     make(chan CreateGameOptions),
		join:       make(chan JoinGameOptions),
		close:      make(chan string),
		done:       make(chan error),
		gameExpiry: gameExpiry,
		adapters:   adapters,
	}
}

func (h *gameHub) Start() {
	// catch any panics and close the game out gracefully
	// prevents the server from crashing due to bugs in a game
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v from game key '%s' with stack trace %s", r, h.builder.Key(), string(debug.Stack()))
			logger.Log.Error().Caller().Msg(msg)
		}
	}()
	go h.clean()
	for {
		select {
		case create := <-h.create:
			_, ok := h.games[create.NetworkOptions.GameID]
			if ok {
				h.done <- fmt.Errorf("game id '%s' already exists", create.NetworkOptions.GameID)
				continue
			}
			server, err := newServer(h.builder, &create, h.adapters)
			if err != nil {
				logger.Log.Error().Err(err).Msgf("failed to create game '%s' with id '%s'", h.builder.Key(), create.NetworkOptions.GameID)
				h.done <- err
				continue
			}
			cleanup := make(chan bool)
			go server.Start(cleanup)
			h.games[create.NetworkOptions.GameID] = server
			h.cleanup[create.NetworkOptions.GameID] = cleanup
			h.done <- nil
		case join := <-h.join:
			server, ok := h.games[join.GameID]
			if !ok {
				h.done <- fmt.Errorf("game id '%s' does not exist", join.GameID)
				continue
			}
			h.done <- server.Join(join)
		case gameID := <-h.close:
			close(h.cleanup[gameID])
			delete(h.cleanup, gameID)
			delete(h.games, gameID)
		}
	}
}

func (h *gameHub) Create(options CreateGameOptions) error {
	h.create <- options
	err := <-h.done
	if err != nil {
		return err
	}
	for _, adapter := range h.adapters {
		adapter.OnGameStart(&options)
	}
	return nil
}

func (h *gameHub) Join(options JoinGameOptions) error {
	h.join <- options
	return <-h.done
}

func (h *gameHub) clean() {
	// every hour check if game is passed gameExpiry in which case it is removed
	for range time.Tick(time.Minute) {
		for gameID, server := range h.games {
			deleteTime := server.updatedAt.Add(h.gameExpiry)
			if time.Now().After(deleteTime) {
				logger.Log.Debug().Msgf("cleaning '%s' with id '%s'", h.builder.Key(), gameID)

				// only store games that were actually used
				if len(server.game.GetBGN().Actions) > 0 || server.playCount > 0 {
					if err := h.gameStore.Store(&datastore.Game{
						GameKey:   h.builder.Key(),
						GameID:    gameID,
						BGN:       server.game.GetBGN(),
						CreatedAt: server.createdAt,
						UpdatedAt: server.updatedAt,
						PlayCount: server.playCount,
					}); err != nil {
						logger.Log.Error().Caller().Err(err).Msgf("failed to store '%s' with id '%s'", h.builder.Key(), gameID)
					}
				}

				h.close <- gameID
			}
		}
	}
}

func (h *gameHub) Store(ctx context.Context) error {
	for gameID, server := range h.games {
		// don't store games that were never used
		if len(server.game.GetBGN().Actions) <= 0 && server.playCount <= 0 {
			continue
		}
		if err := h.gameStore.Store(&datastore.Game{
			GameKey:   h.builder.Key(),
			GameID:    gameID,
			BGN:       server.game.GetBGN(),
			CreatedAt: server.createdAt,
			UpdatedAt: server.updatedAt,
			PlayCount: server.playCount,
		}); err != nil {
			logger.Log.Error().Caller().Err(err).Msgf("failed to store '%s' with id '%s'", h.builder.Key(), gameID)
			return err
		}
	}
	return nil
}

func (h *gameHub) Close() error {
	for gameID := range h.games {
		h.close <- gameID
	}
	return nil
}
