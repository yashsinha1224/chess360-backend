// Package eval provides a static position evaluator for the chess engine.
//
// Convention: Evaluate always returns centipawns from White's fixed
// perspective — positive means White is better, negative means Black is
// better, regardless of whose turn it is. This is deliberate: it keeps
// the eval trivial to reason about and to log/graph over a game (a
// "review" curve just reads left to right), at the cost of needing a
// small sign-flip if this is ever wired into a negamax-style search
// (which expects the score relative to the side to move instead).
package eval

import (
	"example/hello/rules"
	"example/hello/types"
)

// ---------------------------------------------------------------------
// Material
// ---------------------------------------------------------------------

// Two material tables (midgame / endgame) so a piece's raw value can
// shift slightly as the game goes on — e.g. a queen is marginally less
// dominant once most other material is off the board, while pawns get
// relatively more valuable.
var materialMG = map[types.Piece]int{
	types.Pawn:   100,
	types.Knight: 320,
	types.Bishop: 330,
	types.Rook:   500,
	types.Queen:  900,
	types.King:   0,
}

var materialEG = map[types.Piece]int{
	types.Pawn:   120,
	types.Knight: 300,
	types.Bishop: 320,
	types.Rook:   520,
	types.Queen:  880,
	types.King:   0,
}

// phaseWeight/totalPhase drive gamePhase(): how much non-pawn material
// is still on the board, normalized to [0,1] (1 = full opening material,
// 0 = king-and-pawn-ish endgame).
var phaseWeight = map[types.Piece]int{
	types.Knight: 1,
	types.Bishop: 1,
	types.Rook:   2,
	types.Queen:  4,
}

const totalPhase = 24 // 4*1 + 4*1 + 4*2 + 2*4

// ---------------------------------------------------------------------
// Piece-square tables
//
// Indexed [row][col] exactly like types.BoardSquare's board layout:
// row 0 = rank 8 (Black's home rank), row 7 = rank 1 (White's home
// rank) — matching intiaton.InitializeBoard(). Tables are written from
// White's point of view; for Black pieces we mirror the row (7-row)
// and negate the contribution.
// ---------------------------------------------------------------------

var pawnPST = [8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{50, 50, 50, 50, 50, 50, 50, 50},
	{10, 10, 20, 30, 30, 20, 10, 10},
	{5, 5, 10, 25, 25, 10, 5, 5},
	{0, 0, 0, 20, 20, 0, 0, 0},
	{5, -5, -10, 0, 0, -10, -5, 5},
	{5, 10, 10, -20, -20, 10, 10, 5},
	{0, 0, 0, 0, 0, 0, 0, 0},
}

var knightPST = [8][8]int{
	{-50, -40, -30, -30, -30, -30, -40, -50},
	{-40, -20, 0, 0, 0, 0, -20, -40},
	{-30, 0, 10, 15, 15, 10, 0, -30},
	{-30, 5, 15, 20, 20, 15, 5, -30},
	{-30, 0, 15, 20, 20, 15, 0, -30},
	{-30, 5, 10, 15, 15, 10, 5, -30},
	{-40, -20, 0, 5, 5, 0, -20, -40},
	{-50, -40, -30, -30, -30, -30, -40, -50},
}

var bishopPST = [8][8]int{
	{-20, -10, -10, -10, -10, -10, -10, -20},
	{-10, 0, 0, 0, 0, 0, 0, -10},
	{-10, 0, 5, 10, 10, 5, 0, -10},
	{-10, 5, 5, 10, 10, 5, 5, -10},
	{-10, 0, 10, 10, 10, 10, 0, -10},
	{-10, 10, 10, 10, 10, 10, 10, -10},
	{-10, 5, 0, 0, 0, 0, 5, -10},
	{-20, -10, -10, -10, -10, -10, -10, -20},
}

var rookPST = [8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{5, 10, 10, 10, 10, 10, 10, 5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{0, 0, 0, 5, 5, 0, 0, 0},
}

var queenPST = [8][8]int{
	{-20, -10, -10, -5, -5, -10, -10, -20},
	{-10, 0, 0, 0, 0, 0, 0, -10},
	{-10, 0, 5, 5, 5, 5, 0, -10},
	{-5, 0, 5, 5, 5, 5, 0, -5},
	{0, 0, 5, 5, 5, 5, 0, -5},
	{-10, 5, 5, 5, 5, 5, 0, -10},
	{-10, 0, 5, 0, 0, 0, 0, -10},
	{-20, -10, -10, -5, -5, -10, -10, -20},
}

// King gets separate mg/eg tables — it wants safety on the back rank
// early and activity in the center once the board empties out.
var kingMGPST = [8][8]int{
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-20, -30, -30, -40, -40, -30, -30, -20},
	{-10, -20, -20, -20, -20, -20, -20, -10},
	{20, 20, 0, 0, 0, 0, 20, 20},
	{20, 30, 10, 0, 0, 10, 30, 20},
}

var kingEGPST = [8][8]int{
	{-50, -40, -30, -20, -20, -30, -40, -50},
	{-30, -20, -10, 0, 0, -10, -20, -30},
	{-30, -10, 20, 30, 30, 20, -10, -30},
	{-30, -10, 30, 40, 40, 30, -10, -30},
	{-30, -10, 30, 40, 40, 30, -10, -30},
	{-30, -10, 20, 30, 30, 20, -10, -30},
	{-30, -30, 0, 0, 0, 0, -30, -30},
	{-50, -30, -30, -30, -30, -30, -30, -50},
}

// ---------------------------------------------------------------------
// Tunable bonuses/penalties for the non-PST heuristics
// ---------------------------------------------------------------------

const (
	bishopPairBonus     = 30
	doubledPawnPenalty  = 15
	isolatedPawnPenalty = 12
	openFileBonus       = 15
	semiOpenFileBonus   = 8
	mobilityWeight      = 2
	kingShieldBonus     = 10
	inCheckPenalty      = 50
)

// passedPawnBonus is indexed by row (0=rank8 ... 7=rank1), i.e. how far
// a WHITE pawn has advanced. Black pawns mirror via 7-row.
var passedPawnBonus = [8]int{0, 120, 80, 50, 30, 15, 5, 0}

// ---------------------------------------------------------------------
// Evaluate
// ---------------------------------------------------------------------

// Evaluate returns a centipawn score for the given position, positive
// favoring White, from White's fixed perspective (see package doc).
func Evaluate(board [][]types.BoardSquare) int {
	phase := gamePhase(board)
	score := 0.0

	whitePawnFiles := map[int]int{}
	blackPawnFiles := map[int]int{}
	var whiteBishops, blackBishops int
	var whiteKing, blackKing [2]int

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := board[row][col]
			if sq.Piece == nil {
				continue
			}
			piece := sq.Piece.GetType()
			color := sq.Piece.GetColor()

			matVal := taper(materialMG[piece], materialEG[piece], phase)
			pstVal := float64(pstValue(piece, color, row, col, phase))

			sign := 1.0
			if color == types.Black {
				sign = -1.0
			}
			score += sign * (matVal + pstVal)

			switch piece {
			case types.Pawn:
				if color == types.White {
					whitePawnFiles[col]++
				} else {
					blackPawnFiles[col]++
				}
			case types.Bishop:
				if color == types.White {
					whiteBishops++
				} else {
					blackBishops++
				}
			case types.King:
				if color == types.White {
					whiteKing = [2]int{row, col}
				} else {
					blackKing = [2]int{row, col}
				}
			}
		}
	}

	if whiteBishops >= 2 {
		score += bishopPairBonus
	}
	if blackBishops >= 2 {
		score -= bishopPairBonus
	}

	score += pawnStructureScore(board, whitePawnFiles, blackPawnFiles, phase)
	score += rookFileScore(board, whitePawnFiles, blackPawnFiles)
	score += mobilityScore(board)
	score += kingSafetyScore(board, whiteKing, blackKing, phase)

	return int(score)
}

// ---------------------------------------------------------------------
// Phase / PST helpers
// ---------------------------------------------------------------------

func gamePhase(board [][]types.BoardSquare) float64 {
	total := 0
	for _, row := range board {
		for _, sq := range row {
			if sq.Piece == nil {
				continue
			}
			total += phaseWeight[sq.Piece.GetType()]
		}
	}
	if total > totalPhase {
		total = totalPhase
	}
	return float64(total) / float64(totalPhase)
}

func taper(mg, eg int, phase float64) float64 {
	return float64(mg)*phase + float64(eg)*(1-phase)
}

func pstValue(piece types.Piece, color types.Color, row, col int, phase float64) int {
	r := row
	if color == types.Black {
		r = 7 - row
	}
	switch piece {
	case types.Pawn:
		return pawnPST[r][col]
	case types.Knight:
		return knightPST[r][col]
	case types.Bishop:
		return bishopPST[r][col]
	case types.Rook:
		return rookPST[r][col]
	case types.Queen:
		return queenPST[r][col]
	case types.King:
		return int(taper(kingMGPST[r][col], kingEGPST[r][col], phase))
	}
	return 0
}

// ---------------------------------------------------------------------
// Pawn structure: doubled, isolated, passed
// ---------------------------------------------------------------------

func pawnStructureScore(board [][]types.BoardSquare, whiteFiles, blackFiles map[int]int, phase float64) float64 {
	score := 0.0

	for file, count := range whiteFiles {
		if count > 1 {
			score -= float64(doubledPawnPenalty * (count - 1))
		}
		if !hasNeighborPawns(whiteFiles, file) {
			score -= isolatedPawnPenalty
		}
	}
	for file, count := range blackFiles {
		if count > 1 {
			score += float64(doubledPawnPenalty * (count - 1))
		}
		if !hasNeighborPawns(blackFiles, file) {
			score += isolatedPawnPenalty
		}
	}

	score += passedPawnScore(board, phase)
	return score
}

func hasNeighborPawns(files map[int]int, file int) bool {
	return files[file-1] > 0 || files[file+1] > 0
}

func passedPawnScore(board [][]types.BoardSquare, phase float64) float64 {
	score := 0.0
	// Passed pawns matter more the fewer pieces are left to stop them.
	endgameFactor := 1.0 + (1.0 - phase) // ranges 1..2

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := board[row][col]
			if sq.Piece == nil || sq.Piece.GetType() != types.Pawn {
				continue
			}
			color := sq.Piece.GetColor()
			if !isPassed(board, row, col, color) {
				continue
			}
			var bonus float64
			if color == types.White {
				bonus = float64(passedPawnBonus[row])
			} else {
				bonus = float64(passedPawnBonus[7-row])
			}
			bonus *= endgameFactor
			if color == types.White {
				score += bonus
			} else {
				score -= bonus
			}
		}
	}
	return score
}

// isPassed reports whether the pawn at (row,col) has no enemy pawns on
// its own or adjacent files anywhere ahead of it.
func isPassed(board [][]types.BoardSquare, row, col int, color types.Color) bool {
	var start, end, step int
	if color == types.White {
		start, end, step = row-1, -1, -1
	} else {
		start, end, step = row+1, 8, 1
	}

	for c := col - 1; c <= col+1; c++ {
		if c < 0 || c > 7 {
			continue
		}
		for r := start; r != end; r += step {
			p := board[r][c].Piece
			if p != nil && p.GetType() == types.Pawn && p.GetColor() != color {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------
// Rooks on open/semi-open files
// ---------------------------------------------------------------------

func rookFileScore(board [][]types.BoardSquare, whiteFiles, blackFiles map[int]int) float64 {
	score := 0.0
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := board[row][col]
			if sq.Piece == nil || sq.Piece.GetType() != types.Rook {
				continue
			}
			color := sq.Piece.GetColor()
			ownPawns, oppPawns := whiteFiles[col], blackFiles[col]
			if color == types.Black {
				ownPawns, oppPawns = blackFiles[col], whiteFiles[col]
			}
			var bonus float64
			switch {
			case ownPawns == 0 && oppPawns == 0:
				bonus = openFileBonus
			case ownPawns == 0:
				bonus = semiOpenFileBonus
			}
			if color == types.White {
				score += bonus
			} else {
				score -= bonus
			}
		}
	}
	return score
}

// ---------------------------------------------------------------------
// Mobility
// ---------------------------------------------------------------------

func mobilityScore(board [][]types.BoardSquare) float64 {
	score := 0.0
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := board[row][col]
			if sq.Piece == nil || sq.Piece.GetType() == types.King {
				// King "mobility" via GetMoveForPiece isn't a safety signal
				// (it doesn't know about check), so it's skipped here.
				continue
			}
			moves := rules.GetMoveForPiece(sq, board)
			bonus := float64(len(moves)) * mobilityWeight
			if sq.Piece.GetColor() == types.White {
				score += bonus
			} else {
				score -= bonus
			}
		}
	}
	return score
}

// ---------------------------------------------------------------------
// King safety
// ---------------------------------------------------------------------

func kingSafetyScore(board [][]types.BoardSquare, whiteKing, blackKing [2]int, phase float64) float64 {
	score := 0.0

	// Pawn shield only matters while there's enough material left on the
	// board to actually mount an attack on the king.
	if phase >= 0.3 {
		score += float64(pawnShieldCount(board, whiteKing, types.White)) * kingShieldBonus
		score -= float64(pawnShieldCount(board, blackKing, types.Black)) * kingShieldBonus
	}

	if rules.IsKingInCheck(types.White, board) {
		score -= inCheckPenalty
	}
	if rules.IsKingInCheck(types.Black, board) {
		score += inCheckPenalty
	}

	return score
}

func pawnShieldCount(board [][]types.BoardSquare, kingPos [2]int, color types.Color) int {
	row, col := kingPos[0], kingPos[1]
	shieldRow := row - 1
	if color == types.Black {
		shieldRow = row + 1
	}
	if shieldRow < 0 || shieldRow > 7 {
		return 0
	}
	count := 0
	for c := col - 1; c <= col+1; c++ {
		if c < 0 || c > 7 {
			continue
		}
		p := board[shieldRow][c].Piece
		if p != nil && p.GetType() == types.Pawn && p.GetColor() == color {
			count++
		}
	}
	return count
}
