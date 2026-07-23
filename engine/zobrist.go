package engine

import (
	"math/rand"

	"example/hello/types"
)

var zobristPiece [2][6][8][8]uint64
var zobristSideToMove uint64
var zobristCastle [4]uint64
var zobristEnPassantFile [8]uint64

func InitZobrist(seed int64) {
	r := rand.New(rand.NewSource(seed))
	for c := 0; c < 2; c++ {
		for p := 0; p < 6; p++ {
			for row := 0; row < 8; row++ {
				for col := 0; col < 8; col++ {
					zobristPiece[c][p][row][col] = r.Uint64()
				}
			}
		}
	}
	zobristSideToMove = r.Uint64()
	for i := range zobristCastle {
		zobristCastle[i] = r.Uint64()
	}
	for i := range zobristEnPassantFile {
		zobristEnPassantFile[i] = r.Uint64()
	}
}

func pieceIdx(p types.Piece) int {
	switch p {
	case types.Pawn:
		return 0
	case types.Knight:
		return 1
	case types.Bishop:
		return 2
	case types.Rook:
		return 3
	case types.Queen:
		return 4
	case types.King:
		return 5
	}
	return -1
}

func colorIdx(c types.Color) int {
	if c == types.White {
		return 0
	}
	return 1
}

func pieceKey(p types.Piece, c types.Color, row, col int) uint64 {
	idx := pieceIdx(p)
	if idx < 0 {
		return 0
	}
	return zobristPiece[colorIdx(c)][idx][row][col]
}

func castleRightsHash(cr CastleRights) uint64 {
	var h uint64
	if cr.WK {
		h ^= zobristCastle[0]
	}
	if cr.WQ {
		h ^= zobristCastle[1]
	}
	if cr.BK {
		h ^= zobristCastle[2]
	}
	if cr.BQ {
		h ^= zobristCastle[3]
	}
	return h
}

func ComputeHash(gs *GameState) uint64 {
	var h uint64
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			sq := gs.Board[row][col]
			if sq.Piece == nil {
				continue
			}
			h ^= pieceKey(sq.Piece.GetType(), sq.Piece.GetColor(), row, col)
		}
	}
	if gs.SideToMove == types.Black {
		h ^= zobristSideToMove
	}
	h ^= castleRightsHash(gs.CastleRights)
	if gs.EnPassantTarget != nil {
		h ^= zobristEnPassantFile[gs.EnPassantTarget.Col]
	}
	return h
}
