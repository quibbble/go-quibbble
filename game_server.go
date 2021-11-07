package go_boardgame_networking

import (
	"encoding/json"
	"fmt"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame-networking/pkg/timer"
	"log"
	"math/rand"
	"sync"
	"time"
)

// gameServer handles all the processing of messages from players for a single game instance
type gameServer struct {
	options   *NetworkingCreateGameOptions
	createdAt time.Time
	game      bg.BoardGameWithBGN
	timer     *timer.Timer
	alarm     chan bool
	players   map[*player]string
	join      chan *player
	leave     chan *player
	process   chan *message
	done      chan error
	adapters  []NetworkAdapter
}

func newServer(game bg.BoardGameWithBGN, options *NetworkingCreateGameOptions, adapters []NetworkAdapter) *gameServer {
	var clock *timer.Timer
	alarm := make(chan bool)
	if options.TurnLength != nil {
		clock = timer.NewTimer(time.Duration(*options.TurnLength), alarm)
	}
	// playing online against unknown people with timer enabled so start timer right away
	if clock != nil && len(options.Players) > 0 {
		defer clock.Start()
	}
	return &gameServer{
		options:   options,
		createdAt: time.Now(),
		game:      game,
		timer:     clock,
		alarm:     alarm,
		players:   make(map[*player]string),
		join:      make(chan *player),
		leave:     make(chan *player),
		process:   make(chan *message),
		done:      make(chan error),
		adapters:  adapters,
	}
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
			var snapshot *bg.BoardGameSnapshot
			var err error
			if s.players[player] != "" {
				snapshot, err = s.game.GetSnapshot(s.players[player])
			} else {
				snapshot, err = s.game.GetSnapshot()
			}
			if err != nil {
				s.done <- err
				continue
			}
			var timeLeft string
			if s.timer != nil {
				timeLeft = s.timer.Remaining().String()
			}
			s.sendGameMessage(player, &NetworkingGameMessage{s.options, timeLeft}, snapshot)
			s.done <- nil
		case player := <-s.leave:
			delete(s.players, player)
			if !byteChanIsClosed(player.send) {
				close(player.send)
			}
		case message := <-s.process:
			oldSnapshot, _ := s.game.GetSnapshot()
			var action bg.BoardGameAction
			if err := json.Unmarshal(message.payload, &action); err != nil {
				s.sendErrorMessage(message.player, &NetworkingGameMessage{s.options, ""}, err)
				continue
			}
			if err := s.game.Do(&action); err != nil {
				s.sendErrorMessage(message.player, &NetworkingGameMessage{s.options, ""}, err)
				continue
			}
			snapshot, _ := s.game.GetSnapshot()
			if len(snapshot.Winners) > 0 {
				for _, adapter := range s.adapters {
					adapter.OnGameEnd(&GameMessage{
						Network:  &NetworkingGameMessage{s.options, ""},
						Snapshot: snapshot,
					})
				}
			}
			var timeLeft string
			if s.timer != nil {
				if len(snapshot.Winners) > 0 {
					s.timer.Stop()
				} else if oldSnapshot.Turn != snapshot.Turn {
					s.timer.Start()
					timeLeft = s.timer.Remaining().String()
				}
			}
			for player, team := range s.players {
				if team != "" {
					snapshot, err := s.game.GetSnapshot(team)
					if err != nil {
						s.sendErrorMessage(player, &NetworkingGameMessage{s.options, timeLeft}, err)
						continue
					}
					s.sendGameMessage(player, &NetworkingGameMessage{s.options, timeLeft}, snapshot)
				} else {
					s.sendGameMessage(player, &NetworkingGameMessage{s.options, timeLeft}, snapshot)
				}
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
			var timeLeft string
			if s.timer != nil {
				if len(snapshot.Winners) > 0 {
					s.timer.Stop()
				} else {
					s.timer.Start()
					timeLeft = s.timer.Remaining().String()
				}
			}
			for player, team := range s.players {
				if team != "" {
					snapshot, err := s.game.GetSnapshot(team)
					if err != nil {
						s.sendErrorMessage(player, &NetworkingGameMessage{s.options, timeLeft}, err)
						continue
					}
					s.sendGameMessage(player, &NetworkingGameMessage{s.options, timeLeft}, snapshot)
				} else {
					s.sendGameMessage(player, &NetworkingGameMessage{s.options, timeLeft}, snapshot)
				}
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

func (s *gameServer) sendGameMessage(player *player, network *NetworkingGameMessage, snapshot *bg.BoardGameSnapshot) {
	payload, _ := json.Marshal(GameMessage{
		Network:  network,
		Snapshot: snapshot,
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

func (s *gameServer) sendErrorMessage(player *player, network *NetworkingGameMessage, err error) {
	payload, _ := json.Marshal(GameErrorMessage{
		Network: network,
		Error:   err.Error(),
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
