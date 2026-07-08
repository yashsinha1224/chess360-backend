package types

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID       string
	Name     string
	Gmail    string
	ElO      int
	Conn     *websocket.Conn
	GameId   string
	IsInGame bool
}

type GameStatus string

var (
	StatusActive    GameStatus = "active"
	StatusCheckmate GameStatus = "checkmate"
	StatusStalemate GameStatus = "stalemate"
	StatusEneded    GameStatus = "ended"
	StatusCheck     GameStatus = "check"
)

type Game struct {
	ID        string
	White     *Player
	Black     *Player
	Board     [][]BoardSquare
	Turn      Color
	Status    GameStatus
	Winner    Color
	Moves     []string
	StartTime time.Time
}

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type MessageType string

const (
	MsgMove     MessageType = "move"
	MsgJoin     MessageType = "join"
	MsgResign   MessageType = "resign"
	MsgRematch  MessageType = "rematch"
	MsgGameOver MessageType = "game_over"
	MsgError    MessageType = "error"
)

type Hub struct {
	Games   map[string]*Game
	Players map[string]*Player
	Buckets map[int][]*Player
	Mu      sync.RWMutex
}
type gameold struct {
	board          [][]BoardSquare
	Turn           Color
	WhiteConnected bool
	BlackConnected bool
	WhiteConn      *websocket.Conn
	BlackConn      *websocket.Conn
}
