package go_boardgame_networking

import (
	"github.com/gorilla/websocket"
	"sync"
	"time"
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
	defer func() {
		p.server.leave <- p
		_ = p.conn.Close()
	}()
	p.conn.SetReadLimit(maxMessageSize)
	_ = p.conn.SetReadDeadline(time.Now().Add(pongWait))
	p.conn.SetPongHandler(func(string) error { _ = p.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	// tell outside resource pump started
	wg.Done()
	for {
		_, msg, err := p.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
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
		_ = p.conn.Close()
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
