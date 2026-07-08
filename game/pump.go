package game

import (
	"time"

	"example/hello/types"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // must be < pongWait
	maxMessageSize = 4096
)

// startPumps wires up a player's connection the instant it's upgraded —
// before matchmaking, before any game exists. Both pumps run for the
// entire lifetime of the socket.
func startPumps(p *types.Player) {
	p.Send = make(chan []byte, 16)
	p.Incoming = make(chan []byte, 4)
	p.GameFound = make(chan struct{})

	go writePump(p)
	go readPump(p)
}

// writePump is the ONLY goroutine allowed to call p.Conn.WriteMessage.
// Game writes (via Send) and pings share it, so they can never race
// on the connection.
func writePump(p *types.Player) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-p.Send:
			p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := p.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			p.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := p.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump is the ONLY goroutine allowed to call p.Conn.ReadMessage.
// It runs from connection-open to connection-death, including while
// the player is sitting in the matchmaking queue — this is what makes
// pong replies get processed (and read deadlines reset) even before
// a game exists.
func readPump(p *types.Player) {
	defer close(p.Incoming) // signals "this player's socket is gone"

	p.Conn.SetReadLimit(maxMessageSize)
	p.Conn.SetReadDeadline(time.Now().Add(pongWait))
	p.Conn.SetPongHandler(func(string) error {
		p.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := p.Conn.ReadMessage()
		if err != nil {
			return
		}
		select {
		case p.Incoming <- message:
		default:
			// Nobody's consuming yet (still queued) or the game loop is
			// momentarily behind — drop rather than block the read loop,
			// since a blocked read loop stops pong processing too.
		}
	}
}

// send is the single safe way to write to a player from anywhere else
// in the codebase. Never call p.Conn.WriteMessage directly outside this file.
func send(p *types.Player, data []byte) {
	if p == nil {
		return
	}
	select {
	case p.Send <- data:
	default:
		// Send buffer full -> connection is backed up or dead.
		// Don't block the caller, which may be holding game state.
	}
}

func closePlayerConn(p *types.Player) {
	if p == nil {
		return
	}

	p.CloseOnce.Do(func() {
		close(p.Send)
	})
}
