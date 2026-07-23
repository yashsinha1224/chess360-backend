// engine/movegen.go
package engine

import (
	"example/hello/rules"
	"example/hello/types"
)

func GenerateLegalMoves(gs *GameState) []Move {
	var moves []Move
	board := gs.Board
	color := gs.SideToMove

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := board[row][col]
			if sq.Piece == nil || sq.Piece.GetColor() != color {
				continue
			}

			pseudo := rules.GetMoveForPiece(sq, board)
			for _, dest := range pseudo {
				m := Move{
					From:  sq.Position,
					To:    dest,
					Piece: sq.Piece.GetType(),
					Color: color,
				}
				if target := board[dest.Row][dest.Col].Piece; target != nil {
					m.Capture = target.GetType()
				}
				if sq.Piece.GetType() == types.King && abs(dest.Col-sq.Position.Col) == 2 {
					if dest.Col > sq.Position.Col {
						m.IsCastleKing = true
					} else {
						m.IsCastleQueen = true
					}
					if !castleIsSafe(gs, sq.Position, dest) {
						continue // skip: castling through/out of/into check
					}
				}

				isPromotion := sq.Piece.GetType() == types.Pawn && (dest.Row == 0 || dest.Row == 7)
				if isPromotion {
					for _, promo := range []types.Piece{types.Queen, types.Rook, types.Bishop, types.Knight} {
						pm := m
						pm.Promotion = promo
						if legalAfter(gs, pm) {
							moves = append(moves, pm)
						}
					}
					continue
				}

				if legalAfter(gs, m) {
					moves = append(moves, m)
				}
			}

			// En passant: not in GetMoveForPiece, add separately.
			if sq.Piece.GetType() == types.Pawn && gs.EnPassantTarget != nil {
				ep := *gs.EnPassantTarget
				direction := -1
				if color == types.Black {
					direction = 1
				}
				if ep.Row == sq.Position.Row+direction && abs(ep.Col-sq.Position.Col) == 1 {
					m := Move{From: sq.Position, To: ep, Piece: types.Pawn, Color: color, IsEnPassant: true, Capture: types.Pawn}
					if legalAfter(gs, m) {
						moves = append(moves, m)
					}
				}
			}
		}
	}
	return moves
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func castleIsSafe(gs *GameState, from, to types.Position) bool {
	if rules.IsKingInCheck(gs.SideToMove, gs.Board) {
		return false
	}
	step := 1
	if to.Col < from.Col {
		step = -1
	}
	for col := from.Col; col != to.Col+step; col += step {
		if squareAttacked(gs.Board, types.Position{Row: from.Row, Col: col}, gs.SideToMove) {
			return false
		}
	}
	return true
}

func squareAttacked(board [][]types.BoardSquare, pos types.Position, defendingColor types.Color) bool {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			sq := board[i][j]
			if sq.Piece == nil || sq.Piece.GetColor() == defendingColor {
				continue
			}
			for _, m := range rules.GetMoveForPiece(sq, board) {
				if m.Row == pos.Row && m.Col == pos.Col {
					return true
				}
			}
		}
	}
	return false
}

func legalAfter(gs *GameState, m Move) bool {
	undo := ApplyMove(gs, m)
	inCheck := rules.IsKingInCheck(m.Color, gs.Board)
	UndoMove(gs, m, undo)
	return !inCheck
}
