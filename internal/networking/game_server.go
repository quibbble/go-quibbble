package go_boardgame_networking

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/pkg/timer"
	"github.com/rs/zerolog"
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
	log       zerolog.Logger
}

func newServer(builder bg.BoardGameWithBGNBuilder, options *CreateGameOptions, adapters []NetworkAdapter, log zerolog.Logger) (*gameServer, error) {
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
		log:       log,
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

func (s *gameServer) Start(done <-chan bool) {
	// catch any panics and close the game out gracefully
	// prevents the server from crashing due to bugs in a game
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v from game key '%s' and id '%s' and stack trace %s", r, s.options.GameKey, s.options.GameID, string(debug.Stack()))
			s.log.Error().Caller().Msg(msg)
			s.done <- fmt.Errorf(msg)
		}
	}()

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
			s.sendNetworkMessage(player)
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
				s.sendGameMessage(message.player)
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
				s.game = game
				for player := range s.players {
					s.sendGameMessage(player)
				}
				continue
			case ServerActionResign:
				if len(s.options.Players) == 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				// todo add resign field to server and do random action for resigned player if it is their turn
				continue
			case ServerActionReset:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, fmt.Errorf("%s action not allowed", action.ActionType))
					continue
				}
				var details struct {
					MoreOptions interface{}
				}
				if err := mapstructure.Decode(action.MoreDetails, &details); err != nil {
					s.sendErrorMessage(message.player, err)
					continue
				}
				var game bg.BoardGameWithBGN
				var err error
				if s.create != nil {
					game, err = s.builder.CreateWithBGN(&bg.BoardGameOptions{
						Teams:       s.create.Teams,
						MoreOptions: details.MoreOptions,
					})
					if err != nil {
						s.sendErrorMessage(message.player, err)
						continue
					}
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
			if s.players[message.player] != action.Team {
				s.sendErrorMessage(message.player, fmt.Errorf("cannot perform game action for another team"))
				continue
			}
			if err := s.game.Do(&action); err != nil {
				s.sendErrorMessage(message.player, err)
				continue
			}
			snapshot, _ := s.game.GetSnapshot()
			if len(snapshot.Winners) > 0 {
				for _, adapter := range s.adapters {
					adapter.OnGameEnd(snapshot, s.options)
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
		case <-s.alarm:
			// do random action(s) for player if time runs out
			snapshot, _ := s.game.GetSnapshot()
			targets, ok := snapshot.Targets.([]*bg.BoardGameAction)
			if !ok {
				s.log.Debug().Msg("cannot do random action as targets are not of type []*bg.BoardGameAction")
				continue
			}
			if len(targets) == 0 {
				s.log.Debug().Msg("cannot do random action as no valid targets exist")
				continue
			}
			turn := snapshot.Turn
			for len(snapshot.Winners) == 0 && turn == snapshot.Turn {
				action := targets[rand.Intn(len(targets))]
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
		case <-done:
			return
		}
	}
}

func (s *gameServer) Join(options JoinGameOptions) error {
	player := newPlayer(options, s, s.log)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go player.ReadPump(wg)
	go player.WritePump(wg)
	wg.Wait()
	s.join <- player
	return <-s.done
}

func (s *gameServer) sendGameMessage(player *player) {
	var snapshot *bg.BoardGameSnapshot
	if s.players[player] == "" {
		snapshot, _ = s.game.GetSnapshot()
	} else {
		snapshot, _ = s.game.GetSnapshot(s.players[player])
	}
	payload, _ := json.Marshal(OutboundMessage{
		Type:    "Game",
		Payload: snapshot,
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

func (s *gameServer) sendNetworkMessage(player *player) {
	var timeLeft string
	if s.timer != nil {
		timeLeft = s.timer.Remaining().String()
	}
	payload, _ := json.Marshal(OutboundMessage{
		Type: "Network",
		Payload: &outboundNetworkMessage{
			NetworkingCreateGameOptions: s.options,
			Name:                        player.playerName,
			TurnTimeLeft:                timeLeft,
		},
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
	payload, _ := json.Marshal(OutboundMessage{
		Type:    "Chat",
		Payload: s.chat[len(s.chat)-1],
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
	payload, _ := json.Marshal(OutboundMessage{
		Type:    "Connected",
		Payload: connected,
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
	payload, _ := json.Marshal(OutboundMessage{
		Type:    "Error",
		Payload: err.Error(),
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
