package engine

import (
	"time"

	"example/hello/eval"
	"example/hello/rules"
	"example/hello/types"
)

const (
	MateScore = 1_000_000
	DrawScore = 0

	maxIterativeDepth = 64
	nodesPerTimeCheck = 1024
)

type SearchResult struct {
	BestMove Move
	Score    int
	Nodes    int
	Depth    int
}

type searchContext struct {
	nodes int

	hashHistory []uint64

	tt *TranspositionTable

	killers      [maxSearchPly][2]Move
	historyTable [2][64][64]int

	startTime time.Time
	timeLimit time.Duration
	stop      bool
}

func (ctx *searchContext) timeUp() bool {
	if ctx.stop {
		return true
	}
	if ctx.timeLimit <= 0 {
		return false
	}
	if ctx.nodes%nodesPerTimeCheck != 0 {
		return false
	}
	if time.Since(ctx.startTime) >= ctx.timeLimit {
		ctx.stop = true
	}
	return ctx.stop
}

func Search(gs *GameState, depth int, gameHistory []uint64) SearchResult {
	moves := GenerateLegalMoves(gs)
	if len(moves) == 0 {
		if rules.IsKingInCheck(gs.SideToMove, gs.Board) {
			return SearchResult{Score: -MateScore, Nodes: 1}
		}
		return SearchResult{Score: DrawScore, Nodes: 1}
	}

	rootHistory := rootHashHistory(gs, gameHistory)
	if gs.HalfmoveClock >= 100 || isThreefoldRepetition(rootHistory) || isInsufficientMaterial(gs) {
		return SearchResult{Score: DrawScore, Nodes: 1}
	}

	ctx := &searchContext{hashHistory: rootHistory}
	return searchRootMoves(gs, depth, moves, ctx)
}

func SearchIterative(gs *GameState, timeLimit time.Duration, gameHistory []uint64, tt *TranspositionTable) SearchResult {
	moves := GenerateLegalMoves(gs)
	if len(moves) == 0 {
		if rules.IsKingInCheck(gs.SideToMove, gs.Board) {
			return SearchResult{Score: -MateScore, Nodes: 1}
		}
		return SearchResult{Score: DrawScore, Nodes: 1}
	}

	rootHistory := rootHashHistory(gs, gameHistory)
	if gs.HalfmoveClock >= 100 || isThreefoldRepetition(rootHistory) || isInsufficientMaterial(gs) {
		return SearchResult{Score: DrawScore, Nodes: 1}
	}

	ctx := &searchContext{
		hashHistory: rootHistory,
		tt:          tt,
		startTime:   time.Now(),
		timeLimit:   timeLimit,
	}

	var best SearchResult
	for depth := 1; depth <= maxIterativeDepth; depth++ {
		ctx.stop = false

		result := searchRootMoves(gs, depth, moves, ctx)

		if ctx.stop && depth > 1 {
			break
		}

		best = result
		best.Depth = depth

		if ctx.stop || time.Since(ctx.startTime) >= timeLimit || isForcedMateScore(best.Score) {
			break
		}
	}
	return best
}

func rootHashHistory(gs *GameState, gameHistory []uint64) []uint64 {
	h := append([]uint64{}, gameHistory...)
	if len(h) == 0 || h[len(h)-1] != gs.Hash {
		h = append(h, gs.Hash)
	}
	return h
}

func searchRootMoves(gs *GameState, depth int, moves []Move, ctx *searchContext) SearchResult {
	var ttMove Move
	if ctx.tt != nil {
		if entry, ok := ctx.tt.Probe(gs.Hash); ok {
			ttMove = entry.BestMove
		}
	}
	orderMovesForSearch(moves, ttMove, 0, ctx)

	alpha, beta := -MateScore-1, MateScore+1
	best := SearchResult{Score: -MateScore - 1}

	for _, m := range moves {
		undo := ApplyMove(gs, m)
		ctx.hashHistory = append(ctx.hashHistory, gs.Hash)

		score := -negamax(gs, depth-1, 1, -beta, -alpha, ctx)

		ctx.hashHistory = ctx.hashHistory[:len(ctx.hashHistory)-1]
		UndoMove(gs, m, undo)

		if ctx.stop {
			break
		}

		if score > best.Score {
			best.Score = score
			best.BestMove = m
		}
		if score > alpha {
			alpha = score
		}
	}

	best.Nodes = ctx.nodes
	if ctx.tt != nil && !ctx.stop {
		ctx.tt.Store(gs.Hash, depth, scoreToTT(best.Score, 0), TTExact, best.BestMove)
	}
	return best
}

func negamax(gs *GameState, depth, ply, alpha, beta int, ctx *searchContext) int {
	ctx.nodes++

	if ctx.timeUp() {
		return 0
	}

	if gs.HalfmoveClock >= 100 || isThreefoldRepetition(ctx.hashHistory) {
		return DrawScore
	}

	moves := GenerateLegalMoves(gs)
	if len(moves) == 0 {
		if rules.IsKingInCheck(gs.SideToMove, gs.Board) {

			return -(MateScore - ply)
		}
		return DrawScore
	}

	if isInsufficientMaterial(gs) {
		return DrawScore
	}

	if depth == 0 {
		return quiescence(gs, alpha, beta, ply, ctx)
	}

	origAlpha := alpha
	var ttMove Move

	if ctx.tt != nil {
		if entry, ok := ctx.tt.Probe(gs.Hash); ok {
			ttMove = entry.BestMove
			if entry.Depth >= depth {
				score := scoreFromTT(entry.Score, ply)
				switch entry.Flag {
				case TTExact:
					return score
				case TTLowerBound:
					if score > alpha {
						alpha = score
					}
				case TTUpperBound:
					if score < beta {
						beta = score
					}
				}
				if alpha >= beta {
					return score
				}
			}
		}
	}

	orderMovesForSearch(moves, ttMove, ply, ctx)

	best := -MateScore - 1
	var bestMove Move

	for _, m := range moves {
		undo := ApplyMove(gs, m)
		ctx.hashHistory = append(ctx.hashHistory, gs.Hash)

		score := -negamax(gs, depth-1, ply+1, -beta, -alpha, ctx)

		ctx.hashHistory = ctx.hashHistory[:len(ctx.hashHistory)-1]
		UndoMove(gs, m, undo)

		if ctx.stop {
			return best
		}

		if score > best {
			best = score
			bestMove = m
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			if !isCapture(m) {
				recordKiller(ctx, ply, m)
				recordHistory(ctx, m, depth)
			}
			break
		}
	}

	if ctx.tt != nil {
		var flag TTFlag
		switch {
		case best <= origAlpha:
			flag = TTUpperBound
		case best >= beta:
			flag = TTLowerBound
		default:
			flag = TTExact
		}
		ctx.tt.Store(gs.Hash, depth, scoreToTT(best, ply), flag, bestMove)
	}

	return best
}

func evaluateForSideToMove(gs *GameState) int {
	score := eval.Evaluate(gs.Board)
	if gs.SideToMove == types.Black {
		return -score
	}
	return score
}
