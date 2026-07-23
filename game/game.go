package game

import (
	"example/hello/types"

	"example/hello/rules"
	"strings"
)

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

func ValidMove(message []byte, gamestate *types.Game, p *types.Player) bool {
	msg := string(message)
	parts := strings.Split(msg, ",")

	if len(parts) != 2 {
		send(p, []byte(`{"type":"error","payload":{"message":"malformed move"}}`))
		return false
	}

	fromRow, fromCol, ok := parsePosition(parts[0])
	if !ok {
		send(p, []byte(`{"type":"error","payload":{"message":"invalid from-square"}}`))
		return false
	}
	toRow, toCol, ok := parsePosition(parts[1])
	if !ok {
		send(p, []byte(`{"type":"error","payload":{"message":"invalid to-square"}}`))
		return false
	}

	fromSquare := gamestate.Board[fromRow][fromCol]

	if fromSquare.Piece == nil {
		send(p, []byte(`{"type":"error","payload":{"message":"no piece there"}}`))
		return false
	}

	if fromSquare.Piece.GetColor() != gamestate.Turn {
		send(p, []byte(`{"type":"error","payload":{"message":"not your piece"}}`))
		return false
	}

	legalMoves := rules.GetLegalMovesForSquare(fromSquare, gamestate.Board)

	for _, move := range legalMoves {
		if move.Row == toRow && move.Col == toCol {
			return true
		}
	}
	if fromSquare.Piece.GetType() == types.Pawn {
		if epPos, ok := rules.GetEnPassantMove(fromSquare.Position, gamestate.Board, gamestate.EnPassantTarget); ok {
			if epPos.Row == toRow && epPos.Col == toCol {
				return true
			}
		}
	}

	send(p, []byte(`{"type":"error","payload":{"message":"illegal move"}}`))
	return false
}

func ExecuteMove(message []byte, g *types.Game, p *types.Player) {
	if !ValidMove(message, g, p) {
		return
	}

	msg := string(message)
	parts := strings.Split(msg, ",")
	if len(parts) != 2 {
		return
	}

	fromRow, fromCol, ok1 := parsePosition(parts[0])
	toRow, toCol, ok2 := parsePosition(parts[1])
	if !ok1 || !ok2 {
		return
	}

	movingPiece := g.Board[fromRow][fromCol].Piece
	capturedPiece := g.Board[toRow][toCol].Piece

	isCastle := false
	if _, ok := movingPiece.(*types.KingPiece); ok {
		if toCol-fromCol == 2 || toCol-fromCol == -2 {
			isCastle = true
		}
	}

	isEnPassant := false
	if _, ok := movingPiece.(*types.PawnPiece); ok {
		if fromCol != toCol && capturedPiece == nil {
			isEnPassant = true
		}
	}

	g.Board[toRow][toCol].Piece = g.Board[fromRow][fromCol].Piece
	g.Board[fromRow][fromCol].Piece = nil

	if isCastle {
		fromSq := types.BoardSquare{Position: types.Position{Row: fromRow, Col: fromCol}}
		toSq := types.BoardSquare{Position: types.Position{Row: toRow, Col: toCol}}
		rules.ExecuteCastling(fromSq, toSq, g.Board)
	}

	if isEnPassant {
		capRow, capCol := fromRow, toCol
		capturedPiece = g.Board[capRow][capCol].Piece
		g.Board[capRow][capCol].Piece = nil
	}

	if capturedPiece != nil && movingPiece != nil {
		if movingPiece.GetColor() == types.White {
			g.CapturedByWhite = append(g.CapturedByWhite, capturedPiece)
		} else {
			g.CapturedByBlack = append(g.CapturedByBlack, capturedPiece)
		}
	}

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

	// Update en passant target: only set immediately after a pawn's
	// double-step, cleared on every other move (it only ever lasts one
	// ply — you can't en passant a pawn that skipped two moves ago).
	g.EnPassantTarget = nil
	if _, ok := movingPiece.(*types.PawnPiece); ok {
		if toRow-fromRow == 2 || toRow-fromRow == -2 {
			midRow := (fromRow + toRow) / 2
			g.EnPassantTarget = &types.Position{Row: midRow, Col: fromCol}
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
		g.Winner = "Draw"
	}
}

func convertCapturedToJSON(pieces []types.ChessPiece) []interface{} {
	out := make([]interface{}, len(pieces))
	for i, p := range pieces {
		out[i] = map[string]interface{}{
			"type":  string(p.GetType()),
			"color": string(p.GetColor()),
		}
	}
	return out
}
