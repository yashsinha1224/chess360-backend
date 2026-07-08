package game

import (
	"context"
	"encoding/json"
	"example/hello/auth"
	"example/hello/db"
	"example/hello/intiaton"
	"example/hello/rules"
	"example/hello/types"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const disconnectGraceWindow = 60 * time.Second

var (
	playerDoneMu sync.Mutex
	playerDone   = make(map[string]chan struct{})
)

func registerDone(playerID string, done chan struct{}) {
	playerDoneMu.Lock()
	playerDone[playerID] = done
	playerDoneMu.Unlock()
}

func unregisterDone(playerID string) {
	playerDoneMu.Lock()
	delete(playerDone, playerID)
	playerDoneMu.Unlock()
}

func signalPlayerDone(playerID string) {
	playerDoneMu.Lock()
	ch, ok := playerDone[playerID]
	playerDoneMu.Unlock()
	if !ok {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func signalGameEnd(game *types.Game) {
	if game.White != nil {
		signalPlayerDone(game.White.ID)
	}
	if game.Black != nil {
		signalPlayerDone(game.Black.ID)
	}
}

// ---- Reconnect grace period ----
//
// When a player's socket dies mid-game, we don't end the game immediately.
// We record a pendingDisconnect keyed by player.ID and give them 60s to
// open a new WebSocket with a valid token for the same user. If they do,
// handleReconnect rebinds the SAME *types.Player (and therefore the SAME
// *types.Game.White/Black pointer) to the new connection. If the timer
// fires first, finalizeDisconnect runs the old "opponent wins" logic.

type pendingDisconnect struct {
	timer  *time.Timer
	game   *types.Game
	player *types.Player
}

var (
	graceMu sync.Mutex
	grace   = make(map[string]*pendingDisconnect)
)

func opponentOf(game *types.Game, player *types.Player) *types.Player {
	if player == game.White {
		return game.Black
	}
	return game.White
}

func boardUpdateBytes(game *types.Game) []byte {
	msg := map[string]interface{}{
		"type":            "board_update",
		"board":           convertBoardToJSON(game.Board),
		"turn":            string(game.Turn),
		"capturedByWhite": convertCapturedToJSON(game.CapturedByWhite),
		"capturedByBlack": convertCapturedToJSON(game.CapturedByBlack),
	}
	b, _ := json.Marshal(msg)
	return b
}

func HandleWebSocket(hub *types.Hub) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenStr := ctx.Query("token")
		if tokenStr == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		userID, err := auth.ParseJWT(tokenStr)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		dbUser, err := db.GetUserByID(ctx.Request.Context(), userID)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			fmt.Println("[ERROR] Failed to upgrade connection:", err)
			return
		}

		// ---- Reconnection path: is this user mid-grace-period? ----
		graceMu.Lock()
		pd, isReconnect := grace[dbUser.ID]
		if isReconnect {
			delete(grace, dbUser.ID)
		}
		graceMu.Unlock()

		if isReconnect {
			pd.timer.Stop()
			fmt.Printf("[RECONNECT] %s reconnected to game %s\n", dbUser.Name, pd.game.ID)

			player := pd.player
			player.Conn = conn
			startPumps(player) // fresh Send/Incoming/GameFound bound to the new conn

			done := make(chan struct{})
			registerDone(player.ID, done)
			defer unregisterDone(player.ID)

			// Re-sync this player: who they are + current board state.
			var color string
			if player == pd.game.White {
				color = "white"
			} else {
				color = "black"
			}
			resyncBytes, _ := json.Marshal(map[string]interface{}{
				"type": "game_resync",
				"payload": map[string]interface{}{
					"gameId":   pd.game.ID,
					"color":    color,
					"opponent": opponentOf(pd.game, player).Name,
				},
			})
			send(player, resyncBytes)
			send(player, boardUpdateBytes(pd.game))

			// Let the opponent know play can continue.
			reconnectedBytes, _ := json.Marshal(map[string]interface{}{
				"type":    "opponent_reconnected",
				"payload": map[string]interface{}{"message": fmt.Sprintf("%s reconnected", player.Name)},
			})
			send(opponentOf(pd.game, player), reconnectedBytes)

			go listenForMoves(hub, player, pd.game)

			<-done
			return
		}

		// ---- Normal path: brand-new player, goes into matchmaking ----
		player := addplayer(hub, dbUser, conn)
		startPumps(player) // reads/writes + ping/pong live from here on, even while queued

		addWaiting(hub, player)

		done := make(chan struct{})
		registerDone(player.ID, done)
		defer unregisterDone(player.ID)

		send(player, []byte(`{"type":"waiting","payload":{"message":"Looking for opponent..."}}`))

		go func() {
			opponent := matchmake(hub, player)
			if opponent == nil {
				return
			}
			removeFromBucket(hub, player)

			game := CreateGameAndMatch(hub, player, opponent)
			if game == nil {
				addWaiting(hub, player)
				addWaiting(hub, opponent)
				signalPlayerDone(player.ID)
				signalPlayerDone(opponent.ID)
				return
			}
			play(hub, player, opponent, game)
		}()

		// Detect a socket dying while the player is still queued. Once a
		// game is found, GameFound closes and this goroutine steps aside
		// immediately WITHOUT consuming from Incoming — listenForMoves
		// becomes the sole reader from that point on.
		go func() {
			for {
				select {
				case _, ok := <-player.Incoming:
					if !ok {
						if !player.IsInGame {
							CleanupPlayer(hub, player)
						}
						signalPlayerDone(player.ID)
						return
					}
				case <-player.GameFound:
					return
				}
			}
		}()

		<-done
	}
}

func CreateGameAndMatch(hub *types.Hub, p1, p2 *types.Player) *types.Game {
	white := p1
	black := p2
	p1.IsInGame = true
	p2.IsInGame = true
	if randomInt(2) == 0 {
		white = p2
		black = p1
	}

	fmt.Printf("[GAME] Creating game: White=%s, Black=%s\n", white.Name, black.Name)

	game := &types.Game{
		ID:        generateGameID(),
		White:     white,
		Black:     black,
		Board:     initializeBoard(),
		Turn:      types.White,
		Status:    types.StatusActive,
		Winner:    "",
		Moves:     []string{},
		StartTime: time.Now(),
	}

	hub.Mu.Lock()
	hub.Games[game.ID] = game
	white.GameId = game.ID
	black.GameId = game.ID
	hub.Mu.Unlock()

	close(white.GameFound)
	close(black.GameFound)

	if err := db.CreateMatch(context.Background(), game.ID, white.ID, black.ID, string(types.StatusActive)); err != nil {
		fmt.Println("[ERROR] failed to persist match:", err)
	}

	fmt.Printf("[GAME] Game %s created, turn: %s\n", game.ID, game.Turn)

	notifyGameStart(game)
	return game
}

func notifyGameStart(game *types.Game) {
	fmt.Printf("[NOTIFY] game_start → %s (White) and %s (Black)\n", game.White.Name, game.Black.Name)

	boardBytes := boardUpdateBytes(game)

	whiteBytes, _ := json.Marshal(map[string]interface{}{
		"type": "game_start",
		"payload": map[string]interface{}{
			"gameId":   game.ID,
			"color":    "white",
			"opponent": game.Black.Name,
		},
	})
	blackBytes, _ := json.Marshal(map[string]interface{}{
		"type": "game_start",
		"payload": map[string]interface{}{
			"gameId":   game.ID,
			"color":    "black",
			"opponent": game.White.Name,
		},
	})

	for _, pair := range []struct {
		player *types.Player
		start  []byte
	}{
		{game.White, whiteBytes},
		{game.Black, blackBytes},
	} {
		send(pair.player, pair.start)
		send(pair.player, boardBytes)
		fmt.Printf("[NOTIFY] Sent to %s\n", pair.player.Name)
	}
}

func convertBoardToJSON(board [][]types.BoardSquare) interface{} {
	result := make([][]interface{}, 8)
	for i := 0; i < 8; i++ {
		result[i] = make([]interface{}, 8)
		for j := 0; j < 8; j++ {
			sq := board[i][j]
			if sq.Piece == nil {
				result[i][j] = nil
			} else {
				result[i][j] = map[string]interface{}{
					"type":       string(sq.Piece.GetType()),
					"color":      string(sq.Piece.GetColor()),
					"hasMoved":   getHasMoved(sq.Piece),
					"hasCastled": getHasCastled(sq.Piece),
				}
			}
		}
	}
	return result
}

func getHasMoved(piece types.ChessPiece) bool {
	switch p := piece.(type) {
	case *types.PawnPiece:
		return p.HasMoved
	case *types.RookPiece:
		return p.HasMoved
	case *types.KingPiece:
		return p.HasMoved
	}
	return false
}

func getHasCastled(piece types.ChessPiece) bool {
	if k, ok := piece.(*types.KingPiece); ok {
		return k.HasCastled
	}
	return false
}

func play(hub *types.Hub, p1, p2 *types.Player, game *types.Game) {
	fmt.Printf("[PLAY] Starting listeners for %s and %s\n", p1.Name, p2.Name)
	go listenForMoves(hub, p1, game)
	go listenForMoves(hub, p2, game)
}

func listenForMoves(hub *types.Hub, player *types.Player, game *types.Game) {
	var myColor types.Color
	if player == game.White {
		myColor = types.White
	} else {
		myColor = types.Black
	}

	fmt.Printf("[LISTEN] %s listening as %s\n", player.Name, myColor)

	for message := range player.Incoming {
		fmt.Printf("[MOVE] %s sent: %s\n", player.Name, string(message))

		if myColor != game.Turn {
			fmt.Printf("[REJECT] Not %s's turn\n", player.Name)
			send(player, []byte(`{"type":"error","payload":{"message":"not your turn"}}`))
			continue
		}

		ExecuteMove(message, game, player)

		opponent := opponentOf(game, player)

		if rules.IsKingInCheck(game.Turn, game.Board) {
			checkBytes, _ := json.Marshal(map[string]interface{}{
				"type": "check",
				"payload": map[string]interface{}{
					"kingColor": string(game.Turn),
					"message":   "CHECK! " + string(game.Turn) + " king is in check!",
				},
			})
			send(player, checkBytes)
			send(opponent, checkBytes)
		}

		boardBytes := boardUpdateBytes(game)
		send(player, boardBytes)
		send(opponent, boardBytes)

		if game.Status != types.StatusActive {
			notifyGameOver(hub, game)
			return
		}
	}

	// Incoming closed -> this player's socket died. Unblock their own
	// HandleWebSocket handler immediately (independent of the game's fate),
	// then give them a grace window to reconnect before anyone wins.
	fmt.Printf("[DISCONNECT] %s's read loop ended\n", player.Name)
	signalPlayerDone(player.ID)
	handlePlayerDisconnect(hub, player, game)
}

var gameEndedMu sync.Mutex

// handlePlayerDisconnect starts (or no-ops if the game already ended) a
// 60s grace period instead of immediately forfeiting the game.
func handlePlayerDisconnect(hub *types.Hub, player *types.Player, game *types.Game) {
	gameEndedMu.Lock()
	stillActive := game.Status == types.StatusActive || game.Status == types.StatusCheck
	gameEndedMu.Unlock()
	if !stillActive {
		fmt.Printf("[DISCONNECT] Game already ended (%s)\n", game.Status)
		return
	}

	graceMu.Lock()
	if _, exists := grace[player.ID]; exists {
		graceMu.Unlock()
		return
	}
	timer := time.AfterFunc(disconnectGraceWindow, func() {
		graceMu.Lock()
		delete(grace, player.ID)
		graceMu.Unlock()
		finalizeDisconnect(hub, player, game)
	})
	grace[player.ID] = &pendingDisconnect{timer: timer, game: game, player: player}
	graceMu.Unlock()

	msgBytes, _ := json.Marshal(map[string]interface{}{
		"type": "opponent_disconnected",
		"payload": map[string]interface{}{
			"message":      fmt.Sprintf("%s disconnected. Waiting up to 60s for them to reconnect...", player.Name),
			"graceSeconds": int(disconnectGraceWindow.Seconds()),
		},
	})
	send(opponentOf(game, player), msgBytes)
	fmt.Printf("[GRACE] %s disconnected from game %s — waiting %v\n", player.Name, game.ID, disconnectGraceWindow)
}

// finalizeDisconnect is the old "opponent wins" logic — now only runs
// once the grace window has actually expired without a reconnect.
func finalizeDisconnect(hub *types.Hub, player *types.Player, game *types.Game) {
	fmt.Printf("[DISCONNECT] %s never reconnected to game %s — forfeiting\n", player.Name, game.ID)

	gameEndedMu.Lock()
	defer gameEndedMu.Unlock()

	if game.Status != types.StatusActive && game.Status != types.StatusCheck {
		fmt.Printf("[DISCONNECT] Game already ended (%s)\n", game.Status)
		return
	}
	game.Status = types.StatusEneded

	if player == game.White {
		game.Winner = types.Black
	} else {
		game.Winner = types.White
	}

	winner := "white"
	if game.Winner == types.Black {
		winner = "black"
	} else if game.Winner == "Draw" {
		winner = "draw"
	}

	if err := db.FinishMatch(context.Background(), game.ID, winner, string(game.Status), game.Moves); err != nil {
		fmt.Println("[ERROR] failed to persist match result:", err)
	}

	msgBytes, _ := json.Marshal(map[string]interface{}{
		"type": "game_over",
		"payload": map[string]interface{}{
			"winner": winner,
			"reason": "opponent_disconnected",
		},
	})

	opponent := opponentOf(game, player)
	if opponent != nil {
		send(opponent, msgBytes)
		closePlayerConn(opponent)
	}
	closePlayerConn(player)

	CleanupPlayer(hub, player)
	if opponent != nil {
		CleanupPlayer(hub, opponent)
	}

	hub.Mu.Lock()
	delete(hub.Games, game.ID)
	hub.Mu.Unlock()

	signalGameEnd(game)
	fmt.Printf("[DISCONNECT] Done for %s\n", player.Name)
}

func notifyGameOver(hub *types.Hub, game *types.Game) {
	fmt.Printf("[GAME_OVER] %s\n", game.ID)

	gameEndedMu.Lock()
	defer gameEndedMu.Unlock()

	if game.Status == types.StatusActive {
		game.Status = types.StatusEneded
	}

	winner := "white"
	if game.Winner == types.Black {
		winner = "black"
	} else if game.Winner == "Draw" {
		winner = "draw"
	}

	if err := db.FinishMatch(context.Background(), game.ID, winner, string(game.Status), game.Moves); err != nil {
		fmt.Println("[ERROR] failed to persist match result:", err)
	}

	msgBytes, _ := json.Marshal(map[string]interface{}{
		"type": "game_over",
		"payload": map[string]interface{}{
			"winner": winner,
			"reason": string(game.Status),
		},
	})

	for _, p := range []*types.Player{game.White, game.Black} {
		if p != nil {
			send(p, msgBytes)
			closePlayerConn(p)
		}
	}

	if game.White != nil {
		CleanupPlayer(hub, game.White)
	}
	if game.Black != nil {
		CleanupPlayer(hub, game.Black)
	}

	hub.Mu.Lock()
	delete(hub.Games, game.ID)
	hub.Mu.Unlock()

	signalGameEnd(game)
}

func randomInt(n int) int {
	return int(time.Now().UnixNano() % int64(n))
}

func generateGameID() string {
	return uuid.New().String()
}

func initializeBoard() [][]types.BoardSquare {
	return intiaton.InitializeBoard()
}
