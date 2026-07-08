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
		fmt.Println("[WS] Connected:", dbUser.Name)

		player := addplayer(hub, dbUser, conn)
		addWaiting(hub, player)

		done := make(chan struct{})
		registerDone(player.ID, done)
		defer unregisterDone(player.ID)

		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"waiting","payload":{"message":"Looking for opponent..."}}`))

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

	if err := db.CreateMatch(context.Background(), game.ID, white.ID, black.ID, string(types.StatusActive)); err != nil {
		fmt.Println("[ERROR] failed to persist match:", err)
	}

	fmt.Printf("[GAME] Game %s created, turn: %s\n", game.ID, game.Turn)

	notifyGameStart(game)
	return game
}

func notifyGameStart(game *types.Game) {
	fmt.Printf("[NOTIFY] game_start → %s (White) and %s (Black)\n", game.White.Name, game.Black.Name)

	boardMsg := map[string]interface{}{
		"type":            "board_update",
		"board":           convertBoardToJSON(game.Board),
		"turn":            string(game.Turn),
		"capturedByWhite": convertCapturedToJSON(game.CapturedByWhite),
		"capturedByBlack": convertCapturedToJSON(game.CapturedByBlack),
	}
	boardBytes, _ := json.Marshal(boardMsg)

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
		if err := pair.player.Conn.WriteMessage(websocket.TextMessage, pair.start); err != nil {
			fmt.Printf("[ERROR] game_start to %s: %v\n", pair.player.Name, err)
		}
		if err := pair.player.Conn.WriteMessage(websocket.TextMessage, boardBytes); err != nil {
			fmt.Printf("[ERROR] board_update to %s: %v\n", pair.player.Name, err)
		}
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

	for {
		_, message, err := player.Conn.ReadMessage()
		if err != nil {
			fmt.Printf("[DISCONNECT] %s read error: %v\n", player.Name, err)
			handleDisconnect(hub, player, game)
			return
		}

		fmt.Printf("[MOVE] %s sent: %s\n", player.Name, string(message))
		fmt.Printf("[STATE] Game %s turn: %s, %s is: %s\n", game.ID, game.Turn, player.Name, myColor)

		if myColor != game.Turn {
			fmt.Printf("[REJECT] Not %s's turn\n", player.Name)
			player.Conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"not your turn"}}`))
			continue
		}

		fmt.Printf("[ACCEPT] Move from %s: %s\n", player.Name, string(message))
		ExecuteMove(message, game, player)
		fmt.Printf("[STATE] New turn: %s\n", game.Turn)

		if rules.IsKingInCheck(game.Turn, game.Board) {
			opponent := game.Black
			if player == game.Black {
				opponent = game.White
			}

			checkMsg := map[string]interface{}{
				"type": "check",
				"payload": map[string]interface{}{
					"kingColor": string(game.Turn),
					"message":   "CHECK! " + string(game.Turn) + " king is in check!",
				},
			}
			checkBytes, _ := json.Marshal(checkMsg)

			player.Conn.WriteMessage(websocket.TextMessage, checkBytes)
			opponent.Conn.WriteMessage(websocket.TextMessage, checkBytes)
			fmt.Printf("[CHECK] %s king is in check!\n", game.Turn)
		}

		opponent := game.Black
		if player == game.Black {
			opponent = game.White
		}

		boardMsg := map[string]interface{}{
			"type":            "board_update",
			"board":           convertBoardToJSON(game.Board),
			"turn":            string(game.Turn),
			"capturedByWhite": convertCapturedToJSON(game.CapturedByWhite),
			"capturedByBlack": convertCapturedToJSON(game.CapturedByBlack),
		}
		boardBytes, _ := json.Marshal(boardMsg)

		if err := player.Conn.WriteMessage(websocket.TextMessage, boardBytes); err != nil {
			fmt.Printf("[ERROR] board to mover %s: %v\n", player.Name, err)
		}
		if err := opponent.Conn.WriteMessage(websocket.TextMessage, boardBytes); err != nil {
			fmt.Printf("[ERROR] board to opponent %s: %v\n", opponent.Name, err)
		}

		if game.Status != types.StatusActive {
			fmt.Printf("[GAME_OVER] Game %s ended: %s\n", game.ID, game.Status)
			notifyGameOver(hub, game)
			return
		}
	}
}

var gameEndedMu sync.Mutex

func handleDisconnect(hub *types.Hub, player *types.Player, game *types.Game) {
	fmt.Printf("[DISCONNECT] %s in game %s\n", player.Name, game.ID)

	gameEndedMu.Lock()
	defer gameEndedMu.Unlock()

	if game.Status != types.StatusActive && game.Status != types.StatusCheck {
		fmt.Printf("[DISCONNECT] Game already ended (%s)\n", game.Status)
		return
	}
	game.Status = types.StatusEneded

	// Whoever disconnected loses; the other color wins.
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

	opponent := game.Black
	if player == game.Black {
		opponent = game.White
	}
	if opponent != nil {
		opponent.Conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"Opponent disconnected"}}`))
		opponent.Conn.Close()
	}
	player.Conn.Close()

	CleanupPlayer(hub, player)
	if opponent != nil {
		CleanupPlayer(hub, opponent)
	}

	// Remove game from hub
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
			p.Conn.WriteMessage(websocket.TextMessage, msgBytes)
			p.Conn.Close()
		}
	}

	// Clean up from hub
	if game.White != nil {
		CleanupPlayer(hub, game.White)
	}
	if game.Black != nil {
		CleanupPlayer(hub, game.Black)
	}

	// Remove game from hub
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
