package engine

import "example/hello/types"

type CastleRights struct {
	WK, WQ, BK, BQ bool
}

type GameState struct {
	Board           [][]types.BoardSquare
	SideToMove      types.Color
	EnPassantTarget *types.Position
	HalfmoveClock   int
	CastleRights    CastleRights
	Hash            uint64
}

type Move struct {
	From, To      types.Position
	Piece         types.Piece
	Color         types.Color
	Capture       types.Piece
	IsEnPassant   bool
	IsCastleKing  bool
	IsCastleQueen bool
	Promotion     types.Piece
}

type UndoInfo struct {
	CapturedPiece    types.ChessPiece
	CapturedAt       types.Position
	PrevEnPassant    *types.Position
	PrevHalfmove     int
	MovedPieceMoved  bool
	RookMoved        bool
	PrevCastleRights CastleRights
	PrevSideToMove   types.Color
	PrevHash         uint64
}

func NewGameState(board [][]types.BoardSquare, sideToMove types.Color) *GameState {
	gs := &GameState{
		Board:      board,
		SideToMove: sideToMove,
		CastleRights: CastleRights{
			WK: true, WQ: true, BK: true, BQ: true,
		},
	}
	gs.Hash = ComputeHash(gs)
	return gs
}
