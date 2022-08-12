package go_boardgame_networking

import (
	"fmt"
	"time"

	bg "github.com/quibbble/go-boardgame"
	"github.com/rs/zerolog"
)

// gameHub is a hub for a unique game type i.e. only for connect4 or only for tsuro
type gameHub struct {
	builder    bg.BoardGameWithBGNBuilder
	games      map[string]*gameServer // mapping from game ID to game server
	create     chan CreateGameOptions
	load       chan LoadGameOptions
	join       chan JoinGameOptions
	close      chan string
	done       chan error
	gameExpiry time.Duration
	adapters   []NetworkAdapter
	log        zerolog.Logger
}

func newGameHub(builder bg.BoardGameWithBGNBuilder, gameExpiry time.Duration, adapters []NetworkAdapter, log zerolog.Logger) *gameHub {
	return &gameHub{
		builder:    builder,
		games:      make(map[string]*gameServer),
		create:     make(chan CreateGameOptions),
		load:       make(chan LoadGameOptions),
		join:       make(chan JoinGameOptions),
		close:      make(chan string),
		done:       make(chan error),
		gameExpiry: gameExpiry,
		adapters:   adapters,
		log:        log,
	}
}

func (h *gameHub) Start() {
	go h.clean()
	for {
		select {
		case create := <-h.create:
			_, ok := h.games[create.NetworkOptions.GameID]
			if ok {
				h.done <- fmt.Errorf("game id '%s' already exists", create.NetworkOptions.GameID)
				continue
			}
			server, err := newServer(h.builder, &create, h.adapters, h.log)
			if err != nil {
				h.done <- err
				continue
			}
			go server.Start()
			h.games[create.NetworkOptions.GameID] = server
			h.done <- nil
		case load := <-h.load:
			_, ok := h.games[load.NetworkOptions.GameID]
			if ok {
				h.done <- fmt.Errorf("game id '%s' already exists", load.NetworkOptions.GameID)
				continue
			}
			server, err := newServerWithBGN(h.builder, &load, h.adapters)
			if err != nil {
				h.done <- err
				continue
			}
			go server.Start()
			h.games[load.NetworkOptions.GameID] = server
			h.done <- nil
		case join := <-h.join:
			server, ok := h.games[join.GameID]
			if !ok {
				h.done <- fmt.Errorf("game id '%s' does not exist", join.GameID)
				continue
			}
			h.done <- server.Join(join)
		case gameID := <-h.close:
			delete(h.games, gameID)
		}
	}
}

func (h *gameHub) Create(options CreateGameOptions) error {
	h.create <- options
	err := <-h.done
	if err == nil {
		for _, adapter := range h.adapters {
			adapter.OnGameStart(&options)
		}
	}
	return err
}

func (h *gameHub) Load(options LoadGameOptions) error {
	h.load <- options
	return <-h.done
}

func (h *gameHub) Join(options JoinGameOptions) error {
	h.join <- options
	return <-h.done
}

func (h *gameHub) clean() {
	// every hour check if game is passed gameExpiry in which case it is removed
	for range time.Tick(time.Minute) {
		for gameID, server := range h.games {
			deleteTime := server.createdAt.Add(h.gameExpiry)
			if time.Now().After(deleteTime) {
				h.log.Debug().Msgf("cleaning '%s' with id '%s'", h.builder.Key(), gameID)
				h.close <- gameID
			}
		}
	}
}
