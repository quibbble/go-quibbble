package go_boardgame_networking

import (
	"context"
	"runtime/debug"
	"time"

	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-quibbble/internal/datastore"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

// TODO add timeout context to prevent gamHub from hanging

// gameHub is a hub for a unique game type i.e. only for connect4 or only for tsuro
type gameHub struct {
	gameStore  datastore.GameStore
	builder    bg.BoardGameBuilder
	games      map[string]*gameServer // mapping from game ID to game server
	create     chan CreateGameOptions
	join       chan JoinGameOptions
	cleanup    chan string
	errCh      chan error
	gameExpiry time.Duration
	adapters   []NetworkAdapter
}

func newGameHub(builder bg.BoardGameBuilder, gameExpiry time.Duration, adapters []NetworkAdapter, gameStore datastore.GameStore) *gameHub {
	return &gameHub{
		gameStore:  gameStore,
		builder:    builder,
		games:      make(map[string]*gameServer),
		create:     make(chan CreateGameOptions),
		join:       make(chan JoinGameOptions),
		cleanup:    make(chan string),
		errCh:      make(chan error),
		gameExpiry: gameExpiry,
		adapters:   adapters,
	}
}

func (h *gameHub) Start() {
	gameKey := h.builder.Key()

	// catch any panics and close the game out gracefully
	// prevents the server from crashing due to bugs in a game
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error().Caller().Msgf("%v from game key '%s' with stack trace %s", r, h.builder.Key(), string(debug.Stack()))
		}
	}()

	go h.clean()
	for {
		select {
		case create := <-h.create:
			gameID := create.NetworkOptions.GameID
			_, ok := h.games[gameID]
			if ok {
				h.errCh <- ErrExistingGameID(gameKey, gameID)
				continue
			}
			server, err := newServer(h.builder, &create, h.adapters)
			if err != nil {
				logger.Log.Error().Err(err).Msgf(ErrCreateGame(gameKey, gameID).Error())
				h.errCh <- err
				continue
			}
			go server.Start(h.cleanup)
			h.games[gameID] = server
			h.errCh <- nil
		case join := <-h.join:
			server, ok := h.games[join.GameID]
			if !ok {
				h.errCh <- ErrNoExistingGameID(gameKey, join.GameID)
				continue
			}
			h.errCh <- server.Join(join)
		case gameID := <-h.cleanup:
			logger.Log.Debug().Caller().Msgf("cleaning up game with key %s and id %s", gameKey, gameID)
			h.games[gameID].Close()
			delete(h.games, gameID)
		}
	}
}

func (h *gameHub) Create(options CreateGameOptions) error {
	h.create <- options
	if err := <-h.errCh; err != nil {
		return err
	}
	for _, adapter := range h.adapters {
		adapter.OnGameStart(&options)
	}
	return nil
}

func (h *gameHub) Join(options JoinGameOptions) error {
	h.join <- options
	return <-h.errCh
}

func (h *gameHub) clean() {
	gameKey := h.builder.Key()

	// every hour check if game is passed gameExpiry in which case it is removed
	for range time.Tick(time.Minute) {
		for gameID, server := range h.games {
			deleteUpdatedAt := server.updatedAt.Add(h.gameExpiry)
			deleteIntializedAt := server.initializedAt.Add(h.gameExpiry)
			now := time.Now().UTC()
			if now.After(deleteUpdatedAt) && now.After(deleteIntializedAt) {
				bgnGame, ok := server.game.(bg.BoardGameWithBGN)
				if !ok {
					logger.Log.Error().Caller().Err(ErrBGNUnsupported(gameKey))
				} else if len(bgnGame.GetBGN().Actions) > 0 || server.playCount > 0 {
					if err := h.gameStore.Store(&datastore.Game{
						GameKey:   gameKey,
						GameID:    gameID,
						BGN:       bgnGame.GetBGN(),
						CreatedAt: server.createdAt,
						UpdatedAt: server.updatedAt,
						PlayCount: server.playCount,
					}); err != nil {
						logger.Log.Error().Caller().Err(err).Msgf(ErrStoreGame(gameKey, gameID).Error())
					}
				}
				logger.Log.Debug().Msgf("cleaning '%s' with id '%s'", h.builder.Key(), gameID)
				h.cleanup <- gameID
			}
		}
	}
}

func (h *gameHub) Store(ctx context.Context) error {
	gameKey := h.builder.Key()

	if _, ok := h.builder.(bg.BoardGameWithBGNBuilder); !ok {
		return ErrBGNUnsupported(gameKey)
	}
	for gameID, server := range h.games {
		bgnGame := server.game.(bg.BoardGameWithBGN)
		if len(bgnGame.GetBGN().Actions) <= 0 && server.playCount <= 0 {
			continue
		}
		if err := h.gameStore.Store(&datastore.Game{
			GameKey:   gameKey,
			GameID:    gameID,
			BGN:       bgnGame.GetBGN(),
			CreatedAt: server.createdAt,
			UpdatedAt: server.updatedAt,
			PlayCount: server.playCount,
		}); err != nil {
			logger.Log.Error().Caller().Err(err).Msgf(ErrStoreGame(gameKey, gameID).Error())
			return err
		}
	}
	return nil
}

func (h *gameHub) Close() error {
	for gameID := range h.games {
		h.cleanup <- gameID
	}
	return nil
}
