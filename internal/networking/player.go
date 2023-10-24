package go_boardgame_networking

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

const (
	writeWait      = 10 * time.Second    // time allowed to write a message to the peer
	pongWait       = 60 * time.Second    // time allowed to read the next pong message from the peer
	pingPeriod     = (pongWait * 9) / 10 // send pings to peer with this period. Must be less than pongWait
	maxMessageSize = 512                 // maximum message size allowed from peer
)

type message struct {
	player  *player
	payload []byte
}

// player is the player connecting to a specific game instance
type player struct {
	playerID   string
	playerName string
	server     *gameServer
	conn       *websocket.Conn
	send       chan []byte

	mu     sync.Mutex
	closed bool
}

func newPlayer(join JoinGameOptions, server *gameServer) *player {
	return &player{
		playerID:   join.PlayerID,
		playerName: join.PlayerName,
		server:     server,
		conn:       join.Conn,
		send:       make(chan []byte, 2),
	}
}

func (p *player) ReadPump(wg *sync.WaitGroup) {
	// read message from client
	defer p.Close()
	p.conn.SetReadLimit(maxMessageSize)
	_ = p.conn.SetReadDeadline(time.Now().Add(pongWait))
	p.conn.SetPongHandler(func(string) error { _ = p.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	// tell outside resource pump started
	wg.Done()
	for {
		_, msg, err := p.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Log.Debug().Err(err).Msg("websocket unexpected close error")
			}
			break
		}
		p.server.process <- &message{
			p,
			msg,
		}
	}
}

func (p *player) WritePump(wg *sync.WaitGroup) {
	// write back to to client
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.Close()
	}()
	// tell outside resource pump started
	wg.Done()
	for {
		select {
		case message, ok := <-p.send:
			if !ok {
				_ = p.writeMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := p.writeMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := p.writeMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (p *player) writeMessage(msgType int, payload []byte) error {
	_ = p.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return p.conn.WriteMessage(msgType, payload)
}

func (p *player) Close() error {
	gameKey, gameID := p.server.builder.Key(), p.server.create.NetworkOptions.GameID
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	logger.Log.Debug().Caller().Msgf("closing player with key %s and id %s", gameKey, gameID)
	p.closed = true
	p.server.leave <- p
	close(p.send)
	return p.conn.Close()
}
