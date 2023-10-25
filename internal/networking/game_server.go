package go_boardgame_networking

import (
	"encoding/json"
	"math/rand"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/pkg/logger"
	"github.com/quibbble/go-quibbble/pkg/timer"
)

// Actions that if sent are performed in the server and not sent down to the game level
const (
	ServerActionSetTeam     = "SetTeam"
	ServerActionSetOpenTeam = "SetOpenTeam"
	ServerActionReset       = "Reset"
	ServerActionUndo        = "Undo"
	ServerActionResign      = "Resign"
	ServerActionChat        = "Chat"
)

// gameServer handles all the processing of messages from players for a single game instance
type gameServer struct {
	options       *NetworkingCreateGameOptions
	create        *CreateGameOptions
	initializedAt time.Time
	createdAt     time.Time
	updatedAt     time.Time
	builder       bg.BoardGameBuilder
	game          bg.BoardGame
	playCount     int // number of time a game has been completed on this game server
	timer         *timer.Timer
	alarm         chan bool
	players       map[*player]string
	chat          []*ChatMessage
	join          chan *player
	leave         chan *player
	process       chan *message
	errCh         chan error
	stop          chan interface{}
	adapters      []NetworkAdapter
}

func newServer(builder bg.BoardGameBuilder, options *CreateGameOptions, adapters []NetworkAdapter) (*gameServer, error) {
	gameKey, gameID := builder.Key(), options.NetworkOptions.GameID

	var clock *timer.Timer
	alarm := make(chan bool)
	if options.NetworkOptions.TurnLength != nil {
		clock = timer.NewTimer(time.Duration(*options.NetworkOptions.TurnLength), alarm)
	}
	// playing online against unknown people with timer enabled so start timer right away
	if clock != nil && len(options.NetworkOptions.Players) > 0 {
		defer clock.Start()
	}
	server := &gameServer{
		options:       options.NetworkOptions,
		builder:       builder,
		timer:         clock,
		alarm:         alarm,
		initializedAt: time.Now().UTC(),
		createdAt:     time.Now().UTC(),
		updatedAt:     time.Now().UTC(),
		players:       make(map[*player]string),
		chat:          make([]*ChatMessage, 0),
		join:          make(chan *player),
		leave:         make(chan *player),
		process:       make(chan *message),
		errCh:         make(chan error),
		stop:          make(chan interface{}),
		adapters:      adapters,
	}
	if options.GameOptions != nil {
		game, err := builder.Create(options.GameOptions)
		if err != nil {
			return nil, err
		}
		server.game = game
		server.create = options
	} else {
		bgnBuilder, ok := builder.(bg.BoardGameWithBGNBuilder)
		if !ok {
			return nil, ErrBGNUnsupported(gameKey)
		}
		if options.BGN != nil {
			game, err := bgnBuilder.Load(options.BGN)
			if err != nil {
				return nil, err
			}
			server.game = game
			server.create = options
		} else if options.GameData != nil {
			game, err := bgnBuilder.Load(options.GameData.BGN)
			if err != nil {
				return nil, err
			}
			server.game = game
			server.create = options
			server.createdAt = options.GameData.CreatedAt
			server.updatedAt = options.GameData.UpdatedAt
			server.playCount = options.GameData.PlayCount
		} else {
			return nil, ErrCreateGameOptions(gameKey, gameID)
		}
	}
	return server, nil
}

func (s *gameServer) Start(cleanup chan string) {
	gameID := s.create.NetworkOptions.GameID
	// catch any panics and close the game out gracefully
	// prevents the server from crashing due to bugs in a game
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error().Caller().Msgf("%v from game key '%s' and id '%s' and stack trace %s", r, s.options.GameKey, s.options.GameID, string(debug.Stack()))
			cleanup <- gameID
			s.loop(true) // restart the server loop to allow for graceful game closure
		}
	}()

	s.loop(false)
}

func (s *gameServer) loop(errored bool) {
	gameKey, gameID := s.builder.Key(), s.create.NetworkOptions.GameID
	for {
		select {
		case player := <-s.join:
			if errored {
				continue
			}
			if _, ok := s.players[player]; ok {
				s.errCh <- ErrPlayerAlreadyConnected(gameKey, gameID)
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
					s.errCh <- ErrPlayerUnauthorized(gameKey, gameID)
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
			s.errCh <- nil
		case player := <-s.leave:
			delete(s.players, player)
			player.Close()
			for player := range s.players {
				s.sendConnectedMessage(player)
			}
		case message := <-s.process:
			if errored {
				continue
			}
			s.updatedAt = time.Now().UTC()
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
					s.sendErrorMessage(message.player, ErrActionNotAllowed(action.ActionType))
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
					s.sendErrorMessage(message.player, ErrInvalidTeam)
					continue
				}
				s.players[message.player] = details.Team
				s.sendGameMessage(message.player)
				for player := range s.players {
					s.sendConnectedMessage(player)
				}
				continue
			case ServerActionSetOpenTeam:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, ErrActionNotAllowed(action.ActionType))
					continue
				}
				if s.players[message.player] != "" {
					s.sendErrorMessage(message.player, ErrAlreadyInTeam)
					continue
				}
				openTeams := append([]string{}, oldSnapshot.Teams...)
				for _, team := range s.players {
					for i, other := range openTeams {
						if other == team {
							openTeams = append(openTeams[:i], openTeams[i+1:]...)
							break
						}
					}
				}
				if len(openTeams) <= 0 {
					s.sendErrorMessage(message.player, ErrNoOpenTeam)
					continue
				}
				s.players[message.player] = openTeams[0]
				s.sendGameMessage(message.player)
				for player := range s.players {
					s.sendConnectedMessage(player)
				}
				continue
			case ServerActionChat:
				if len(s.chat) >= 250 {
					s.sendErrorMessage(message.player, ErrMaxChat)
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
					s.sendErrorMessage(message.player, ErrActionNotAllowed(action.ActionType))
					continue
				}
				if len(oldSnapshot.Actions) == 0 {
					s.sendErrorMessage(message.player, ErrNoActionToUndo)
					continue
				}
				if s.timer != nil {
					s.timer.Stop()
				}
				var game bg.BoardGame
				var err error
				if s.create.GameOptions != nil {
					game, err = s.builder.Create(s.create.GameOptions)
				} else {
					bgnBuilder, ok := s.builder.(bg.BoardGameWithBGNBuilder)
					if !ok {
						logger.Log.Error().Caller().Err(ErrBGNUnsupported(gameKey))
						continue
					}
					if s.create.BGN != nil {
						game, err = bgnBuilder.Load(&bgn.Game{
							Tags: s.create.BGN.Tags,
						})
					} else if s.create.GameData != nil {
						game, err = bgnBuilder.Load(&bgn.Game{
							Tags: s.create.GameData.BGN.Tags,
						})
					} else {
						logger.Log.Error().Msg("missing create options in undo")
						continue
					}
				}
				if err != nil {
					logger.Log.Error().Err(err).Msg("undo action error")
					continue
				}
				var failed bool
				for _, action := range oldSnapshot.Actions[:len(oldSnapshot.Actions)-1] {
					if err = game.Do(action); err != nil {
						logger.Log.Error().Err(err).Msg("undo action error")
						failed = true
						break
					}
				}
				if failed {
					continue
				}
				s.game = game
				for player := range s.players {
					s.sendGameMessage(player)
				}
				continue
			case ServerActionResign:
				if len(s.options.Players) == 0 {
					s.sendErrorMessage(message.player, ErrActionNotAllowed(action.ActionType))
					continue
				}
				// todo add resign field to server and do random action for resigned player if it is their turn
				continue
			case ServerActionReset:
				if len(s.options.Players) > 0 {
					s.sendErrorMessage(message.player, ErrActionNotAllowed(action.ActionType))
					continue
				}
				var details struct {
					MoreOptions struct {
						Seed    int
						Variant string
					}
				}
				if err := mapstructure.Decode(action.MoreDetails, &details); err != nil {
					s.sendErrorMessage(message.player, err)
					continue
				}
				if details.MoreOptions.Seed == 0 {
					details.MoreOptions.Seed = int(time.Now().Unix())
				}
				var game bg.BoardGame
				var err error
				if s.create.GameOptions != nil {
					game, err = s.builder.Create(&bg.BoardGameOptions{
						Teams:       s.create.GameOptions.Teams,
						MoreOptions: details.MoreOptions,
					})
					s.create.GameOptions.MoreOptions = details.MoreOptions
				} else {
					bgnBuilder, ok := s.builder.(bg.BoardGameWithBGNBuilder)
					if !ok {
						logger.Log.Error().Caller().Err(ErrBGNUnsupported(gameKey))
						continue
					}
					if s.create.BGN != nil {
						tags := s.create.BGN.Tags
						tags[bgn.SeedTag] = strconv.Itoa(details.MoreOptions.Seed)
						tags[bgn.VariantTag] = details.MoreOptions.Variant
						game, err = bgnBuilder.Load(&bgn.Game{Tags: tags})
						s.create.BGN.Tags = tags
					} else if s.create.GameData != nil {
						tags := s.create.GameData.BGN.Tags
						tags[bgn.SeedTag] = strconv.Itoa(details.MoreOptions.Seed)
						tags[bgn.VariantTag] = details.MoreOptions.Variant
						game, err = bgnBuilder.Load(&bgn.Game{Tags: tags})
						s.create.GameData.BGN.Tags = tags
					} else {
						logger.Log.Error().Msg("missing create options in undo")
						continue
					}
				}
				if err != nil {
					logger.Log.Error().Err(err).Msg("game reset error")
					continue
				}
				s.game = game
				for player := range s.players {
					s.sendGameMessage(player)
				}
				continue
			default:
				// board game action
				if s.players[message.player] != action.Team {
					s.sendErrorMessage(message.player, ErrWrongTeamAction)
					continue
				}
				if err := s.game.Do(&action); err != nil {
					s.sendErrorMessage(message.player, err)
					continue
				}
				snapshot, _ := s.game.GetSnapshot()
				if len(snapshot.Winners) > 0 {
					s.playCount++
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
			}
		case <-s.alarm:
			if errored {
				continue
			}
			// do random action(s) for player if time runs out
			snapshot, _ := s.game.GetSnapshot()
			targets, ok := snapshot.Targets.([]*bg.BoardGameAction)
			if !ok {
				logger.Log.Debug().Msg("cannot do random action as targets are not of type []*bg.BoardGameAction")
				continue
			}
			if len(targets) == 0 {
				logger.Log.Debug().Msg("cannot do random action as no valid targets exist")
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
		case <-s.stop:
			return
		}
	}
}

func (s *gameServer) Close() {
	gameKey, gameID := s.builder.Key(), s.create.NetworkOptions.GameID
	logger.Log.Debug().Caller().Msgf("closing game server with key %s and id %s", gameKey, gameID)
	for player := range s.players {
		if err := player.Close(); err != nil {
			logger.Log.Error().Caller().Err(err)
		}
	}
	s.stop <- true
}

func (s *gameServer) Join(options JoinGameOptions) error {
	player := newPlayer(options, s)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go player.ReadPump(wg)
	go player.WritePump(wg)
	wg.Wait()
	s.join <- player
	return <-s.errCh
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
		player.Close()
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
		player.Close()
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
		player.Close()
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
		player.Close()
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
		player.Close()
	}
}
