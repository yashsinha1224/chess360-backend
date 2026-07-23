package engine

import (
	"sort"

	"example/hello/types"
)

var mvvLvaValue = map[types.Piece]int{
	types.Pawn:   1,
	types.Knight: 3,
	types.Bishop: 3,
	types.Rook:   5,
	types.Queen:  9,
	types.King:   20,
}

func isCapture(m Move) bool {
	return m.Capture != ""
}

func mvvLvaScore(m Move) int {
	return mvvLvaValue[m.Capture]*10 - mvvLvaValue[m.Piece]
}

func squareIndex(p types.Position) int {
	return p.Row*8 + p.Col
}

func orderMoves(moves []Move) {
	sort.SliceStable(moves, func(i, j int) bool {
		iCap, jCap := isCapture(moves[i]), isCapture(moves[j])
		if iCap != jCap {
			return iCap
		}
		if iCap {
			return mvvLvaScore(moves[i]) > mvvLvaScore(moves[j])
		}
		return false
	})
}

func loudMoves(moves []Move) []Move {
	out := make([]Move, 0, len(moves))
	for _, m := range moves {
		if isCapture(m) || m.Promotion != "" {
			out = append(out, m)
		}
	}
	return out
}

const (
	ttMoveOrderScore  = 2_000_000_000
	captureOrderBase  = 1_000_000_000
	killer0OrderScore = 900_000_000
	killer1OrderScore = 899_000_000
)

func orderMovesForSearch(moves []Move, ttMove Move, ply int, ctx *searchContext) {
	hasTTMove := ttMove != (Move{})

	var killer0, killer1 Move
	if ply < len(ctx.killers) {
		killer0, killer1 = ctx.killers[ply][0], ctx.killers[ply][1]
	}

	scoreOf := func(m Move) int {
		switch {
		case hasTTMove && m == ttMove:
			return ttMoveOrderScore
		case isCapture(m):
			return captureOrderBase + mvvLvaScore(m)
		case killer0 != (Move{}) && m == killer0:
			return killer0OrderScore
		case killer1 != (Move{}) && m == killer1:
			return killer1OrderScore
		default:
			return ctx.historyTable[colorIdx(m.Color)][squareIndex(m.From)][squareIndex(m.To)]
		}
	}

	sort.SliceStable(moves, func(i, j int) bool {
		return scoreOf(moves[i]) > scoreOf(moves[j])
	})
}

func recordKiller(ctx *searchContext, ply int, m Move) {
	if ply >= len(ctx.killers) {
		return
	}
	if ctx.killers[ply][0] == m {
		return
	}
	ctx.killers[ply][1] = ctx.killers[ply][0]
	ctx.killers[ply][0] = m
}

func recordHistory(ctx *searchContext, m Move, depth int) {
	ctx.historyTable[colorIdx(m.Color)][squareIndex(m.From)][squareIndex(m.To)] += depth * depth
}
