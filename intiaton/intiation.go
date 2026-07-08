package intiaton

import (
	"example/hello/types"
)

func InitializeBoard() [][]types.BoardSquare {
	board := make([][]types.BoardSquare, 8)

	backRankOrder := []types.Piece{
		types.Rook,
		types.Knight,
		types.Bishop,
		types.Queen,
		types.King,
		types.Bishop,
		types.Knight,
		types.Rook,
	}

	for row := 0; row < 8; row++ {
		board[row] = make([]types.BoardSquare, 8)
		for col := 0; col < 8; col++ {
			var piece types.ChessPiece = nil

			if row == 0 {
				pieceType := backRankOrder[col]
				switch pieceType {
				case types.Rook:
					piece = &types.RookPiece{BasePiece: types.BasePiece{Type: types.Rook, Color: types.Black, HasMoved: false}}
				case types.Knight:
					piece = &types.KnightPiece{BasePiece: types.BasePiece{Type: types.Knight, Color: types.Black}}
				case types.Bishop:
					piece = &types.BishopPiece{BasePiece: types.BasePiece{Type: types.Bishop, Color: types.Black}}
				case types.Queen:
					piece = &types.QueenPiece{BasePiece: types.BasePiece{Type: types.Queen, Color: types.Black}}
				case types.King:
					piece = &types.KingPiece{BasePiece: types.BasePiece{Type: types.King, Color: types.Black, HasMoved: false}, HasCastled: false, HasCheck: false}
				}
			}

			if row == 1 {
				piece = &types.PawnPiece{BasePiece: types.BasePiece{Type: types.Pawn, Color: types.Black, HasMoved: false}}
			}

			if row == 6 {
				piece = &types.PawnPiece{BasePiece: types.BasePiece{Type: types.Pawn, Color: types.White, HasMoved: false}}
			}

			if row == 7 {
				pieceType := backRankOrder[col]
				switch pieceType {
				case types.Rook:
					piece = &types.RookPiece{BasePiece: types.BasePiece{Type: types.Rook, Color: types.White, HasMoved: false}}
				case types.Knight:
					piece = &types.KnightPiece{BasePiece: types.BasePiece{Type: types.Knight, Color: types.White}}
				case types.Bishop:
					piece = &types.BishopPiece{BasePiece: types.BasePiece{Type: types.Bishop, Color: types.White}}
				case types.Queen:
					piece = &types.QueenPiece{BasePiece: types.BasePiece{Type: types.Queen, Color: types.White}}
				case types.King:
					piece = &types.KingPiece{BasePiece: types.BasePiece{Type: types.King, Color: types.White, HasMoved: false}, HasCastled: false, HasCheck: false}
				}
			}

			board[row][col] = types.BoardSquare{
				Piece:    piece,
				Position: types.Position{Row: row, Col: col},
			}
		}
	}

	return board
}
