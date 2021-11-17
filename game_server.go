package go_boardgame_networking

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame-networking/pkg/timer"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"log"
	"math/rand"
	"sync"
	"time"
)

// Actions that if sent are performed in the server and not sent down to the game level
const (
	ServerActionSetTeam = "SetTeam"
	ServerActionReset   = "Reset"
	ServerActionUndo    = "Undo"
	ServerActionResign  = "Resign"
	ServerActionChat    = "Chat"
)

// gameServer handles all the processing of messages from players for a single game instance
type gameServer struct {
	options   *NetworkingCreateGameOptions
	create    *bg.BoardGameOptions
	load      *bgn.Game
	createdAt time.Time
	builder   bg.BoardGameWithBGNBuilder
	game      bg.BoardGameWithBGN
	timer     *timer.Timer
	alarm     chan bool
	players   map[*player]string
	chat      []*ChatMessage
	join      chan *player
	leave     chan *player
	process   chan *message
	done      chan error
	adapters  []NetworkAdapter
}

func newServer(builder bg.BoardGameWithBGNBuilder, options *CreateGameOptions, adapters []NetworkAdapter) (*gameServer, error) {
	game, err := builder.CreateWithBGN(options.GameOptions)
	if err != nil {
		return nil, err
	}
	var clock *timer.Timer
	alarm := make(chan bool)
	if options.NetworkOptions.TurnLength != nil {
		clock = timer.NewTimer(time.Duration(*options.NetworkOptions.TurnLength), alarm)
	}
	// playing online against unknown people with timer enabled so start timer right away
	if clock != nil && len(options.NetworkOptions.Players) > 0 {
		defer clock.Start()
	}
	return &gameServer{
		options:   options.NetworkOptions,
		create:    options.GameOptions,
		createdAt: time.Now(),
		builder:   builder,
		game:      game,
		timer:     clock,
		alarm:     alarm,
		players:   make(map[*player]string),
		chat:      make([]*ChatMessage, 0),
		join:      make(chan *player),
		leave:     make(chan *player),
		process:   make(chan *message),
		done:      make(chan error),
		adapters:  adapters,
	}, nil
}

func newServerWithBGN(builder bg.BoardGameWithBGNBuilder, options *LoadGameOptions, adapters []NetworkAdapter) (*gameServer, error) {
	game, err := builder.Load(options.BGN)
	if err != nil {
		return nil, err
	}
	var clock *timer.Timer
	alarm := make(chan bool)
	if options.NetworkOptions.TurnLength != nil {
		clock = timer.NewTimer(time.Duration(*options.NetworkOptions.TurnLength), alarm)
	}
	// playing online against unknown people with timer enabled so start timer right away
	if clock != nil && len(options.NetworkOptions.Players) > 0 {
		defer clock.Start()
	}
	return &gameServer{
		options:   options.NetworkOptions,
		load:      options.BGN,
		createdAt: time.Now(),
		builder:   builder,
		game:      game,
		timer:     clock,
		alarm:     alarm,
		players:   make(map[*player]string),
		chat:      make([]*ChatMessage, 0),
		join:      make(chan *player),
		leave:     make(chan *player),
		process:   make(chan *message),
		done:      make(chan error),
		adapters:  adapters,
	}, nil
}

func (s *gameServer) Start() {
	for {
		select {
		case player := <-s.join:
			if _, ok := s.players[player]; ok {
				s.done <- fmt.Errorf("player already connected")
				continue
			}
			if len(s.options.Players) > 0 {
				found := false
				for team, players := range s.options.Players {
					if contains(players, player.playerID) {
						s.players[player] = team
						found = true
						break
					}
				}
				if !found {
					s.done <- fmt.Errorf("player may not join")
					continue
				}
			} else {
				s.players[player] = ""
			}
			s.sendGameMessage(player)
			for player := range s.players {
				s.sendConnectedMessage(player)
			}
			s.done <- nil
		case player := <-s.leave:
			delete(s.players, player)
			if !byteChanIsClosed(player.send) {
				close(player.send)
			}
			for player := range s.players {
				s.sendConnectedMessage(player)
			}
		case message := <-s.process:
			oldSnapshot, _ := s.game.GetSnapshot()
			var action bg.BoardGameAction
			if err := json.Unmarshal(message.payload, &action); err != nil {
				s.sendErrorMessage(message.player, err)
				continue
			}
			// try server action
			switch action.ActionType {
			case ServerActionSetTeam:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				var details struct {
					Team string
				}
				if err := mapstructure.Decode(action.MoreDetails, &details); err != nil {
					s.sendErrorMessage(message.player, err)
					continue
				}
				if !contains(oldSnapshot.Teams, details.Team) {
					s.sendErrorMessage(message.player, fmt.Errorf("invalid team"))
					continue
				}
				s.players[message.player] = details.Team
				for player := range s.players {
					s.sendConnectedMessage(player)
				}
				continue
			case ServerActionChat:
				if len(s.chat) >= 250 {
					s.sendErrorMessage(message.player, fmt.Errorf("max chat limit reached"))
					continue
				}
				var details struct {
					Msg string
				}
				if err := mapstructure.Decode(action.MoreDetails, &details); err != nil {
					s.sendErrorMessage(message.player, err)
					continue
				}
				s.chat = append(s.chat, &ChatMessage{
					Name: message.player.playerName,
					Msg:  details.Msg,
				})
				for player := range s.players {
					s.sendChatMessage(player)
				}
				continue
			case ServerActionUndo:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				if len(oldSnapshot.Actions) == 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("no action to undo"))
					continue
				}
				if s.timer != nil {
					s.timer.Stop()
				}
				var game bg.BoardGameWithBGN
				if s.create != nil {
					game, _ = s.builder.CreateWithBGN(s.create)
				} else {
					game, _ = s.builder.Load(&bgn.Game{
						Tags:    s.load.Tags,
						Actions: make([]bgn.Action, 0),
					})
				}
				for _, action := range oldSnapshot.Actions[:len(oldSnapshot.Actions)-1] {
					_ = game.Do(action)
				}
				for player := range s.players {
					s.sendGameMessage(player)
				}
				continue
			case ServerActionResign:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				// todo add resign field to server and do random action for resigned player if it is their turn
				continue
			case ServerActionReset:
				if len(s.options.Players) == 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				var game bg.BoardGameWithBGN
				if s.create != nil {
					game, _ = s.builder.CreateWithBGN(&bg.BoardGameOptions{
						Teams:       s.create.Teams,
						MoreOptions: s.create.MoreOptions,
					})
				} else {
					game, _ = s.builder.Load(&bgn.Game{
						Tags:    s.load.Tags,
						Actions: make([]bgn.Action, 0),
					})
				}
				s.game = game
				for player := range s.players {
					s.sendGameMessage(player)
				}
				continue
			}
			// try board game action
			if err := s.game.Do(&action); err != nil {
				s.sendErrorMessage(message.player, err)
				continue
			}
			snapshot, _ := s.game.GetSnapshot()
			if len(snapshot.Winners) > 0 {
				for _, adapter := range s.adapters {
					adapter.OnGameEnd(&OutboundGameMessage{
						Type:                        "Game",
						NetworkingCreateGameOptions: s.options,
						Snapshot:                    snapshot,
					})
				}
			}
			if s.timer != nil {
				if len(snapshot.Winners) > 0 {
					s.timer.Stop()
				} else if oldSnapshot.Turn != snapshot.Turn {
					s.timer.Start()
				}
			}
			for player := range s.players {
				s.sendGameMessage(player)
			}
		case _ = <-s.alarm:
			// do random action(s) for player if time runs out
			snapshot, _ := s.game.GetSnapshot()
			if len(snapshot.Targets) == 0 {
				log.Println("cannot do random action as no valid targets exist")
				continue
			}
			turn := snapshot.Turn
			for len(snapshot.Winners) == 0 && turn == snapshot.Turn {
				action := snapshot.Targets[rand.Intn(len(snapshot.Targets))]
				_ = s.game.Do(action)
				snapshot, _ = s.game.GetSnapshot()
			}
			if s.timer != nil {
				if len(snapshot.Winners) > 0 {
					s.timer.Stop()
				} else {
					s.timer.Start()
				}
			}
			for player := range s.players {
				s.sendGameMessage(player)
			}
		}
	}
}

func (s *gameServer) Join(options JoinGameOptions) error {
	player := newPlayer(options, s)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go player.ReadPump(wg)
	go player.WritePump(wg)
	wg.Wait()
	s.join <- player
	return <-s.done
}

func (s *gameServer) sendGameMessage(player *player) {
	var timeLeft string
	if s.timer != nil {
		timeLeft = s.timer.Remaining().String()
	}
	var snapshot *bg.BoardGameSnapshot
	if s.players[player] == "" {
		snapshot, _ = s.game.GetSnapshot()
	} else {
		snapshot, _ = s.game.GetSnapshot(s.players[player])
	}
	payload, _ := json.Marshal(OutboundGameMessage{
		Type:                        "Game",
		NetworkingCreateGameOptions: s.options,
		Snapshot:                    snapshot,
		TurnTimeLeft:                timeLeft,
	})
	select {
	case player.send <- payload:
	default:
		delete(s.players, player)
		if !byteChanIsClosed(player.send) {
			close(player.send)
		}
	}
}

func (s *gameServer) sendChatMessage(player *player) {
	payload, _ := json.Marshal(OutboundChatMessage{
		Type: "Chat",
		Chat: s.chat,
	})
	select {
	case player.send <- payload:
	default:
		delete(s.players, player)
		if !byteChanIsClosed(player.send) {
			close(player.send)
		}
	}
}

func (s *gameServer) sendConnectedMessage(player *player) {
	connected := make(map[string]string)
	for player, team := range s.players {
		connected[player.playerName] = team
	}
	payload, _ := json.Marshal(OutboundConnectedMessage{
		Type:      "Connected",
		Connected: connected,
	})
	select {
	case player.send <- payload:
	default:
		delete(s.players, player)
		if !byteChanIsClosed(player.send) {
			close(player.send)
		}
	}
}

func (s *gameServer) sendErrorMessage(player *player, err error) {
	payload, _ := json.Marshal(OutboundErrorMessage{
		Type:  "Error",
		Error: err.Error(),
	})
	select {
	case player.send <- payload:
	default:
		delete(s.players, player)
		if !byteChanIsClosed(player.send) {
			close(player.send)
		}
	}
}
