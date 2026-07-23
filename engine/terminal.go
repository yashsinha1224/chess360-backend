package engine

import (
	"example/hello/rules"
	"example/hello/types"
)

type GameStatus int

const (
	Ongoing GameStatus = iota
	Checkmate
	Stalemate
	DrawFiftyMove
	DrawRepetition
	DrawInsufficientMaterial
)

func GetStatus(gs *GameState, legalMoves []Move, gameHistory []uint64) GameStatus {
	if len(legalMoves) == 0 {
		if rules.IsKingInCheck(gs.SideToMove, gs.Board) {
			return Checkmate
		}
		return Stalemate
	}
	if gs.HalfmoveClock >= 100 {
		return DrawFiftyMove
	}
	if isThreefoldRepetition(gameHistory) {
		return DrawRepetition
	}
	if isInsufficientMaterial(gs) {
		return DrawInsufficientMaterial
	}
	return Ongoing
}

func IsCheckmate(gs *GameState) bool {
	moves := GenerateLegalMoves(gs)
	return len(moves) == 0 && rules.IsKingInCheck(gs.SideToMove, gs.Board)
}

func IsStalemate(gs *GameState) bool {
	moves := GenerateLegalMoves(gs)
	return len(moves) == 0 && !rules.IsKingInCheck(gs.SideToMove, gs.Board)
}

func isThreefoldRepetition(history []uint64) bool {
	if len(history) == 0 {
		return false
	}
	current := history[len(history)-1]
	count := 0
	for _, h := range history {
		if h == current {
			count++
			if count >= 3 {
				return true
			}
		}
	}
	return false
}

func isInsufficientMaterial(gs *GameState) bool {
	type side struct {
		knights, bishops          int
		lightBishops, darkBishops int
		heavy                     bool
	}
	var white, black side

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			p := gs.Board[row][col].Piece
			if p == nil {
				continue
			}
			s := &white
			if p.GetColor() == types.Black {
				s = &black
			}
			switch p.GetType() {
			case types.Pawn, types.Rook, types.Queen:
				s.heavy = true
			case types.Knight:
				s.knights++
			case types.Bishop:
				s.bishops++
				if (row+col)%2 == 0 {
					s.lightBishops++
				} else {
					s.darkBishops++
				}
			}
		}
	}

	if white.heavy || black.heavy {
		return false
	}

	whiteMinors := white.knights + white.bishops
	blackMinors := black.knights + black.bishops

	if whiteMinors == 0 && blackMinors == 0 {
		return true
	}
	if (whiteMinors == 1 && blackMinors == 0) || (whiteMinors == 0 && blackMinors == 1) {
		return true
	}
	if whiteMinors == 1 && blackMinors == 1 && white.knights == 0 && black.knights == 0 {
		if (white.lightBishops == 1 && black.lightBishops == 1) ||
			(white.darkBishops == 1 && black.darkBishops == 1) {
			return true // K+B vs K+B, same-colored bishops
		}
	}
	return false
}
