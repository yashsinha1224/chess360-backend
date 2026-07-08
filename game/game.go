package game

import (
	"example/hello/types"

	"example/hello/rules"
	"strings"

	"github.com/gorilla/websocket"
)

// parsePosition parses a square like "e2" into (row, col).
// Returns ok=false for malformed or out-of-range input instead of panicking.
func parsePosition(pos string) (row int, col int, ok bool) {
	if len(pos) != 2 {
		return 0, 0, false
	}
	col = int(pos[0] - 'a')
	row = 8 - int(pos[1]-'0')
	if col < 0 || col > 7 || row < 0 || row > 7 {
		return 0, 0, false
	}
	return row, col, true
}

func ValidMove(message []byte, gamestate *types.Game, conn *websocket.Conn) bool {
	msg := string(message)
	parts := strings.Split(msg, ",")

	if len(parts) != 2 {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"malformed move"}}`))
		return false
	}

	fromRow, fromCol, ok := parsePosition(parts[0])
	if !ok {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"invalid from-square"}}`))
		return false
	}
	toRow, toCol, ok := parsePosition(parts[1])
	if !ok {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"invalid to-square"}}`))
		return false
	}

	fromSquare := gamestate.Board[fromRow][fromCol]

	if fromSquare.Piece == nil {
		conn.WriteMessage(websocket.TextMessage, []byte("no piece there"))
		return false
	}

	if fromSquare.Piece.GetColor() != gamestate.Turn {
		conn.WriteMessage(websocket.TextMessage, []byte("not your piece"))
		return false
	}

	legalMoves := rules.GetLegalMovesForSquare(fromSquare, gamestate.Board)

	for _, move := range legalMoves {
		if move.Row == toRow && move.Col == toCol {
			return true
		}
	}

	conn.WriteMessage(websocket.TextMessage, []byte("illegal move"))
	return false
}

// ExecuteMove validates the move itself before mutating any state. This
// makes ExecuteMove safe to call directly — callers no longer need to call
// ValidMove first, though doing so beforehand (e.g. to avoid extra logging
// on a rejected move) is still fine, since this check is idempotent.
func ExecuteMove(message []byte, g *types.Game, p *types.Player) {
	if !ValidMove(message, g, p.Conn) {
		return
	}

	msg := string(message)
	parts := strings.Split(msg, ",")
	if len(parts) != 2 {
		// Unreachable given ValidMove passed, kept as defense in depth.
		return
	}

	fromRow, fromCol, ok1 := parsePosition(parts[0])
	toRow, toCol, ok2 := parsePosition(parts[1])
	if !ok1 || !ok2 {
		return
	}

	g.Board[toRow][toCol].Piece = g.Board[fromRow][fromCol].Piece
	g.Board[fromRow][fromCol].Piece = nil

	if piece, ok := g.Board[toRow][toCol].Piece.(*types.PawnPiece); ok {
		piece.HasMoved = true
	}
	if piece, ok := g.Board[toRow][toCol].Piece.(*types.RookPiece); ok {
		piece.HasMoved = true
	}
	if piece, ok := g.Board[toRow][toCol].Piece.(*types.KingPiece); ok {
		piece.HasMoved = true
	}

	if piece, ok := g.Board[toRow][toCol].Piece.(*types.PawnPiece); ok {
		if (toRow == 0 && piece.GetColor() == types.White) ||
			(toRow == 7 && piece.GetColor() == types.Black) {
			g.Board[toRow][toCol].Piece = types.MakePiece(piece.GetColor(), types.Queen)
		}
	}

	if g.Turn == types.White {
		g.Turn = types.Black
	} else {
		g.Turn = types.White
	}

	g.Moves = append(g.Moves, msg)

	if rules.IsCheckmate(g.Turn, g.Board) {
		g.Status = types.StatusCheckmate
		if g.Turn == types.White {
			g.Winner = types.Black
		} else {
			g.Winner = types.White
		}
	}
	if rules.IsStalemate(g.Turn, g.Board) {
		g.Status = types.StatusStalemate
		if g.Turn == types.White {
			g.Winner = "Draw"
		} else {
			g.Winner = "Draw"
		}
	}
}
