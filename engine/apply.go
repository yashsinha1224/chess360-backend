package engine

import (
	"example/hello/types"
)

func opposite(c types.Color) types.Color {
	if c == types.White {
		return types.Black
	}
	return types.White
}

func ApplyMove(gs *GameState, m Move) UndoInfo {
	board := gs.Board
	from, to := m.From, m.To

	movingPiece := board[from.Row][from.Col].Piece

	undo := UndoInfo{
		PrevEnPassant:    gs.EnPassantTarget,
		PrevHalfmove:     gs.HalfmoveClock,
		PrevCastleRights: gs.CastleRights,
		PrevSideToMove:   gs.SideToMove,
		PrevHash:         gs.Hash,
		CapturedAt:       to,
	}

	undo.MovedPieceMoved = pieceHasMoved(movingPiece)

	gs.Hash ^= pieceKey(m.Piece, m.Color, from.Row, from.Col)

	if m.IsEnPassant {
		capRow := from.Row
		capCol := to.Col
		undo.CapturedPiece = board[capRow][capCol].Piece
		undo.CapturedAt = types.Position{Row: capRow, Col: capCol}
		gs.Hash ^= pieceKey(types.Pawn, opposite(m.Color), capRow, capCol)
		board[capRow][capCol].Piece = nil
	} else if board[to.Row][to.Col].Piece != nil {
		captured := board[to.Row][to.Col].Piece
		undo.CapturedPiece = captured
		gs.Hash ^= pieceKey(captured.GetType(), captured.GetColor(), to.Row, to.Col)
	}

	board[to.Row][to.Col].Piece = movingPiece
	board[from.Row][from.Col].Piece = nil
	setHasMoved(movingPiece, true)

	if m.IsCastleKing || m.IsCastleQueen {
		var rookFromCol, rookToCol int
		if m.IsCastleKing {
			rookFromCol, rookToCol = 7, to.Col-1
		} else {
			rookFromCol, rookToCol = 0, to.Col+1
		}
		rook := board[from.Row][rookFromCol].Piece
		undo.RookMoved = pieceHasMoved(rook)
		gs.Hash ^= pieceKey(types.Rook, m.Color, from.Row, rookFromCol)
		gs.Hash ^= pieceKey(types.Rook, m.Color, from.Row, rookToCol)
		board[from.Row][rookToCol].Piece = rook
		board[from.Row][rookFromCol].Piece = nil
		setHasMoved(rook, true)
	}

	if m.Promotion != "" {
		board[to.Row][to.Col].Piece = types.MakePiece(m.Color, m.Promotion)
		gs.Hash ^= pieceKey(m.Promotion, m.Color, to.Row, to.Col)
	} else {
		gs.Hash ^= pieceKey(m.Piece, m.Color, to.Row, to.Col)
	}

	gs.Hash ^= castleRightsHash(gs.CastleRights)
	updateCastleRights(gs, m, undo.CapturedPiece, undo.CapturedAt)
	gs.Hash ^= castleRightsHash(gs.CastleRights)

	if gs.EnPassantTarget != nil {
		gs.Hash ^= zobristEnPassantFile[gs.EnPassantTarget.Col]
	}
	gs.EnPassantTarget = nil
	if m.Piece == types.Pawn && abs(to.Row-from.Row) == 2 {
		midRow := (from.Row + to.Row) / 2
		gs.EnPassantTarget = &types.Position{Row: midRow, Col: from.Col}
		gs.Hash ^= zobristEnPassantFile[from.Col]
	}

	if m.Piece == types.Pawn || undo.CapturedPiece != nil {
		gs.HalfmoveClock = 0
	} else {
		gs.HalfmoveClock++
	}

	gs.SideToMove = opposite(gs.SideToMove)
	gs.Hash ^= zobristSideToMove

	return undo
}

func UndoMove(gs *GameState, m Move, undo UndoInfo) {
	board := gs.Board
	from, to := m.From, m.To

	var movingPiece types.ChessPiece
	if m.Promotion != "" {
		movingPiece = types.MakePiece(m.Color, types.Pawn)
	} else {
		movingPiece = board[to.Row][to.Col].Piece
	}

	board[from.Row][from.Col].Piece = movingPiece
	setHasMoved(movingPiece, undo.MovedPieceMoved)

	if m.IsEnPassant {
		board[to.Row][to.Col].Piece = nil
		board[undo.CapturedAt.Row][undo.CapturedAt.Col].Piece = undo.CapturedPiece
	} else {
		board[to.Row][to.Col].Piece = undo.CapturedPiece
	}

	if m.IsCastleKing || m.IsCastleQueen {
		var rookFromCol, rookToCol int
		if m.IsCastleKing {
			rookToCol, rookFromCol = 7, to.Col-1
		} else {
			rookToCol, rookFromCol = 0, to.Col+1
		}
		rook := board[from.Row][rookFromCol].Piece
		board[from.Row][rookToCol].Piece = rook
		board[from.Row][rookFromCol].Piece = nil
		setHasMoved(rook, undo.RookMoved)
	}

	gs.EnPassantTarget = undo.PrevEnPassant
	gs.HalfmoveClock = undo.PrevHalfmove
	gs.CastleRights = undo.PrevCastleRights
	gs.SideToMove = undo.PrevSideToMove
	gs.Hash = undo.PrevHash
}

func pieceHasMoved(p types.ChessPiece) bool {
	switch v := p.(type) {
	case *types.PawnPiece:
		return v.HasMoved
	case *types.RookPiece:
		return v.HasMoved
	case *types.KingPiece:
		return v.HasMoved
	}
	return false
}

func setHasMoved(p types.ChessPiece, val bool) {
	switch v := p.(type) {
	case *types.PawnPiece:
		v.HasMoved = val
	case *types.RookPiece:
		v.HasMoved = val
	case *types.KingPiece:
		v.HasMoved = val
	}
}

func updateCastleRights(gs *GameState, m Move, capturedPiece types.ChessPiece, capturedAt types.Position) {
	if m.Piece == types.King {
		if m.Color == types.White {
			gs.CastleRights.WK = false
			gs.CastleRights.WQ = false
		} else {
			gs.CastleRights.BK = false
			gs.CastleRights.BQ = false
		}
	}
	if m.Piece == types.Rook {
		revokeRookRight(gs, m.Color, m.From)
	}
	if capturedPiece != nil && capturedPiece.GetType() == types.Rook {
		revokeRookRight(gs, opposite(m.Color), capturedAt)
	}
}

func revokeRookRight(gs *GameState, color types.Color, pos types.Position) {
	homeRow := 7
	if color == types.Black {
		homeRow = 0
	}
	if pos.Row != homeRow {
		return
	}
	switch pos.Col {
	case 0:
		if color == types.White {
			gs.CastleRights.WQ = false
		} else {
			gs.CastleRights.BQ = false
		}
	case 7:
		if color == types.White {
			gs.CastleRights.WK = false
		} else {
			gs.CastleRights.BK = false
		}
	}
}
