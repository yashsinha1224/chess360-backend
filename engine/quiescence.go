package engine

import "example/hello/rules"

const maxQuiescencePly = 32

func quiescence(gs *GameState, alpha, beta, ply int, ctx *searchContext) int {
	ctx.nodes++

	if ctx.timeUp() {
		return alpha
	}

	inCheck := rules.IsKingInCheck(gs.SideToMove, gs.Board)

	var candidates []Move
	if inCheck {

		candidates = GenerateLegalMoves(gs)
		if len(candidates) == 0 {
			return -(MateScore - ply)
		}
	} else {
		standPat := evaluateForSideToMove(gs)
		if standPat >= beta {
			return standPat
		}
		if standPat > alpha {
			alpha = standPat
		}
		if ply >= maxQuiescencePly {
			return standPat
		}
		candidates = loudMoves(GenerateLegalMoves(gs))
	}

	orderMoves(candidates)

	for _, m := range candidates {
		undo := ApplyMove(gs, m)
		score := -quiescence(gs, -beta, -alpha, ply+1, ctx)
		UndoMove(gs, m, undo)

		if ctx.stop {
			return alpha
		}

		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			return alpha
		}
	}
	return alpha
}
