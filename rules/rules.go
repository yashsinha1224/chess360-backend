package rules

import (
	"example/hello/types"
)

func GetMoveForPiece(Square types.BoardSquare, board [][]types.BoardSquare) []types.Position {
	var legal []types.Position
	p := Square.Piece.GetType()

	if p == "p" {
		pawn, ok := Square.Piece.(*types.PawnPiece)
		if ok {
			direction := 1
			if Square.Piece.GetColor() == types.White {
				direction = -1
			}

			// Double move on first move
			if !pawn.HasMoved {
				intermediateRow := Square.Position.Row + direction
				newRow := Square.Position.Row + direction*2
				col := Square.Position.Col
				if newRow >= 0 && newRow < 8 &&
					board[intermediateRow][col].Piece == nil &&
					board[newRow][col].Piece == nil {
					legal = append(legal, types.Position{Row: newRow, Col: col})
				}
			}

			// Single move forward
			newRow := Square.Position.Row + direction
			col := Square.Position.Col
			if newRow >= 0 && newRow < 8 && board[newRow][col].Piece == nil {
				legal = append(legal, types.Position{Row: newRow, Col: col})
			}

			// Captures
			for _, captureCol := range []int{col - 1, col + 1} {
				if newRow >= 0 && newRow < 8 && captureCol >= 0 && captureCol < 8 {
					target := board[newRow][captureCol].Piece
					if target != nil && target.GetColor() != Square.Piece.GetColor() {
						legal = append(legal, types.Position{Row: newRow, Col: captureCol})
					}
				}
			}
		}
	}

	if p == "k" {
		king, ok := Square.Piece.(*types.KingPiece)
		if ok {
			row := Square.Position.Row
			col := Square.Position.Col

			// Regular king moves (1 square in any direction)
			offsets := [][2]int{
				{1, 0}, {-1, 0}, {0, 1}, {0, -1},
				{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
			}
			for _, offset := range offsets {
				newRow := row + offset[0]
				newCol := col + offset[1]
				if newRow < 0 || newRow >= 8 || newCol < 0 || newCol >= 8 {
					continue
				}
				target := board[newRow][newCol].Piece
				if target == nil || target.GetColor() != king.GetColor() {
					legal = append(legal, types.Position{Row: newRow, Col: newCol})
				}
			}

			// CASTLING - Only if king hasn't moved
			if !king.HasMoved && !king.HasCastled {
				// Kingside castling (short castling) - to col+2
				if col+3 < 8 {
					if rook, ok := board[row][col+3].Piece.(*types.RookPiece); ok && !rook.HasMoved {
						// Check if squares between are empty
						if board[row][col+1].Piece == nil && board[row][col+2].Piece == nil {
							legal = append(legal, types.Position{Row: row, Col: col + 2})
						}
					}
				}

				// Queenside castling (long castling) - to col-2
				if col-4 >= 0 {
					if rook, ok := board[row][col-4].Piece.(*types.RookPiece); ok && !rook.HasMoved {
						// Check if squares between are empty
						if board[row][col-1].Piece == nil && board[row][col-2].Piece == nil && board[row][col-3].Piece == nil {
							legal = append(legal, types.Position{Row: row, Col: col - 2})
						}
					}
				}
			}
		}
	}

	if p == "q" {
		queen, ok := Square.Piece.(*types.QueenPiece)
		if ok {
			row := Square.Position.Row
			col := Square.Position.Col

			// Diagonal moves (top-right)
			for i, j := row-1, col+1; i >= 0 && j < 8; i, j = i-1, j+1 {
				target := board[i][j].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: j})
				} else if target.GetColor() != queen.GetColor() {
					legal = append(legal, types.Position{Row: i, Col: j})
					break
				} else {
					break
				}
			}
			// Diagonal moves (top-left)
			for i, j := row-1, col-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
				target := board[i][j].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: j})
				} else if target.GetColor() != queen.GetColor() {
					legal = append(legal, types.Position{Row: i, Col: j})
					break
				} else {
					break
				}
			}
			// Diagonal moves (bottom-right)
			for i, j := row+1, col+1; i < 8 && j < 8; i, j = i+1, j+1 {
				target := board[i][j].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: j})
				} else if target.GetColor() != queen.GetColor() {
					legal = append(legal, types.Position{Row: i, Col: j})
					break
				} else {
					break
				}
			}
			// Diagonal moves (bottom-left)
			for i, j := row+1, col-1; i < 8 && j >= 0; i, j = i+1, j-1 {
				target := board[i][j].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: j})
				} else if target.GetColor() != queen.GetColor() {
					legal = append(legal, types.Position{Row: i, Col: j})
					break
				} else {
					break
				}
			}
			// Vertical down
			for i := row + 1; i < 8; i++ {
				target := board[i][col].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: col})
				} else {
					if target.GetColor() != queen.GetColor() {
						legal = append(legal, types.Position{Row: i, Col: col})
					}
					break
				}
			}
			// Vertical up
			for i := row - 1; i >= 0; i-- {
				target := board[i][col].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: i, Col: col})
				} else {
					if target.GetColor() != queen.GetColor() {
						legal = append(legal, types.Position{Row: i, Col: col})
					}
					break
				}
			}
			// Horizontal right
			for i := col + 1; i < 8; i++ {
				target := board[row][i].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: row, Col: i})
				} else {
					if target.GetColor() != queen.GetColor() {
						legal = append(legal, types.Position{Row: row, Col: i})
					}
					break
				}
			}
			// Horizontal left
			for i := col - 1; i >= 0; i-- {
				target := board[row][i].Piece
				if target == nil {
					legal = append(legal, types.Position{Row: row, Col: i})
				} else {
					if target.GetColor() != queen.GetColor() {
						legal = append(legal, types.Position{Row: row, Col: i})
					}
					break
				}
			}
		}
	}

	if p == "r" {
		color := Square.Piece.GetColor()
		row := Square.Position.Row
		col := Square.Position.Col

		// Down
		for i := row + 1; i < 8; i++ {
			target := board[i][col].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: col})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: col})
				}
				break
			}
		}
		// Up
		for i := row - 1; i >= 0; i-- {
			target := board[i][col].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: col})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: col})
				}
				break
			}
		}
		// Right
		for i := col + 1; i < 8; i++ {
			target := board[row][i].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: row, Col: i})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: row, Col: i})
				}
				break
			}
		}
		// Left
		for i := col - 1; i >= 0; i-- {
			target := board[row][i].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: row, Col: i})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: row, Col: i})
				}
				break
			}
		}
	}

	if p == "b" {
		color := Square.Piece.GetColor()
		row := Square.Position.Row
		col := Square.Position.Col

		// Top-right
		for i, j := row-1, col+1; i >= 0 && j < 8; i, j = i-1, j+1 {
			target := board[i][j].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: j})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: j})
				}
				break
			}
		}
		// Top-left
		for i, j := row-1, col-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
			target := board[i][j].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: j})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: j})
				}
				break
			}
		}
		// Bottom-right
		for i, j := row+1, col+1; i < 8 && j < 8; i, j = i+1, j+1 {
			target := board[i][j].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: j})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: j})
				}
				break
			}
		}
		// Bottom-left
		for i, j := row+1, col-1; i < 8 && j >= 0; i, j = i+1, j-1 {
			target := board[i][j].Piece
			if target == nil {
				legal = append(legal, types.Position{Row: i, Col: j})
			} else {
				if target.GetColor() != color {
					legal = append(legal, types.Position{Row: i, Col: j})
				}
				break
			}
		}
	}

	if p == "n" {
		color := Square.Piece.GetColor()
		row := Square.Position.Row
		col := Square.Position.Col

		offsets := [][2]int{
			{-2, -1}, {-2, 1},
			{-1, -2}, {-1, 2},
			{1, -2}, {1, 2},
			{2, -1}, {2, 1},
		}
		for _, offset := range offsets {
			newRow := row + offset[0]
			newCol := col + offset[1]
			if newRow < 0 || newRow >= 8 || newCol < 0 || newCol >= 8 {
				continue
			}
			target := board[newRow][newCol].Piece
			if target == nil || target.GetColor() != color {
				legal = append(legal, types.Position{Row: newRow, Col: newCol})
			}
		}
	}

	return legal
}

// IsKingInCheck checks if the king of given color is in check
func IsKingInCheck(color types.Color, board [][]types.BoardSquare) bool {
	var kingPos types.Position
	found := false

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			square := board[i][j]
			if square.Piece != nil && square.Piece.GetType() == types.King && square.Piece.GetColor() == color {
				kingPos = types.Position{Row: i, Col: j}
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return false
	}

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			square := board[i][j]
			if square.Piece != nil && square.Piece.GetColor() != color {
				moves := GetMoveForPiece(square, board)
				for _, move := range moves {
					if move.Row == kingPos.Row && move.Col == kingPos.Col {
						return true
					}
				}
			}
		}
	}

	return false
}

func GetEnPassantMove(from types.Position, board [][]types.BoardSquare, target *types.Position) (types.Position, bool) {
	if target == nil {
		return types.Position{}, false
	}
	sq := board[from.Row][from.Col]
	pawn, ok := sq.Piece.(*types.PawnPiece)
	if !ok {
		return types.Position{}, false
	}

	direction := -1
	if pawn.GetColor() == types.Black {
		direction = 1
	}

	if target.Row == from.Row+direction && (target.Col == from.Col-1 || target.Col == from.Col+1) {
		return *target, true
	}
	return types.Position{}, false
}

// IsMoveSafe checks if making a move would leave own king in check
func IsMoveSafe(from types.BoardSquare, to types.BoardSquare, board [][]types.BoardSquare) bool {
	simulatedBoard := copyBoard(board)

	fromRow, fromCol := from.Position.Row, from.Position.Col
	toRow, toCol := to.Position.Row, to.Position.Col

	// Make the move
	simulatedBoard[toRow][toCol].Piece = simulatedBoard[fromRow][fromCol].Piece
	simulatedBoard[fromRow][fromCol].Piece = nil

	if simulatedBoard[toRow][toCol].Piece == nil {
		return false
	}

	myColor := simulatedBoard[toRow][toCol].Piece.GetColor()

	return !IsKingInCheck(myColor, simulatedBoard)
}

// HasLegalMoves checks if a player has any legal moves available
func HasLegalMoves(color types.Color, board [][]types.BoardSquare) bool {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			square := board[i][j]
			if square.Piece != nil && square.Piece.GetColor() == color {
				moves := GetMoveForPiece(square, board)

				for _, movePos := range moves {
					destSquare := board[movePos.Row][movePos.Col]
					if IsMoveSafe(square, destSquare, board) {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetLegalMovesForSquare returns all legal moves for a piece (considering check)
func GetLegalMovesForSquare(square types.BoardSquare, board [][]types.BoardSquare) []types.Position {
	allMoves := GetMoveForPiece(square, board)
	legalMoves := []types.Position{}

	for _, move := range allMoves {
		destSquare := board[move.Row][move.Col]
		if IsMoveSafe(square, destSquare, board) {
			legalMoves = append(legalMoves, move)
		}
	}

	return legalMoves
}

func IsCheckmate(color types.Color, board [][]types.BoardSquare) bool {
	if !IsKingInCheck(color, board) {
		return false
	}
	return !HasLegalMoves(color, board)
}

func IsStalemate(color types.Color, board [][]types.BoardSquare) bool {
	if IsKingInCheck(color, board) {
		return false
	}
	return !HasLegalMoves(color, board)
}

func ExecuteCastling(from types.BoardSquare, to types.BoardSquare, board [][]types.BoardSquare) [][]types.BoardSquare {
	fromCol := from.Position.Col
	toCol := to.Position.Col
	row := from.Position.Row

	if toCol == fromCol+2 {
		// Move rook from col+3 to col+1
		rook := board[row][fromCol+3].Piece
		board[row][fromCol+1].Piece = rook
		board[row][fromCol+3].Piece = nil

		if r, ok := rook.(*types.RookPiece); ok {
			r.HasMoved = true
		}
	}

	if toCol == fromCol-2 {
		rook := board[row][fromCol-4].Piece
		board[row][fromCol-1].Piece = rook
		board[row][fromCol-4].Piece = nil

		if r, ok := rook.(*types.RookPiece); ok {
			r.HasMoved = true
		}
	}

	return board
}

func HandlePawnPromotion(targetPiece types.ChessPiece, toRow int, toCol int, board [][]types.BoardSquare, promoteTo string) [][]types.BoardSquare {
	if targetPiece.GetType() != types.Pawn {
		return board
	}

	if (targetPiece.GetColor() == types.White && toRow != 0) ||
		(targetPiece.GetColor() == types.Black && toRow != 7) {
		return board
	}

	var newPieceType types.Piece
	switch promoteTo {
	case "q":
		newPieceType = types.Queen
	case "r":
		newPieceType = types.Rook
	case "b":
		newPieceType = types.Bishop
	case "n":
		newPieceType = types.Knight
	default:
		newPieceType = types.Queen
	}

	color := targetPiece.GetColor()
	promotedPiece := types.MakePiece(color, newPieceType)

	board[toRow][toCol].Piece = promotedPiece
	return board
}

func copyBoard(board [][]types.BoardSquare) [][]types.BoardSquare {
	newBoard := make([][]types.BoardSquare, 8)
	for i := 0; i < 8; i++ {
		newBoard[i] = make([]types.BoardSquare, 8)
		for j := 0; j < 8; j++ {
			newBoard[i][j] = types.BoardSquare{
				Position: board[i][j].Position,
			}

			if board[i][j].Piece != nil {
				originalPiece := board[i][j].Piece
				color := originalPiece.GetColor()
				pieceType := originalPiece.GetType()

				newPiece := types.MakePiece(color, pieceType)

				switch p := newPiece.(type) {
				case *types.PawnPiece:
					if pawn, ok := originalPiece.(*types.PawnPiece); ok {
						p.HasMoved = pawn.HasMoved
					}
				case *types.RookPiece:
					if rook, ok := originalPiece.(*types.RookPiece); ok {
						p.HasMoved = rook.HasMoved
					}
				case *types.KnightPiece:
				case *types.BishopPiece:
				case *types.QueenPiece:
				case *types.KingPiece:
					if king, ok := originalPiece.(*types.KingPiece); ok {
						p.HasMoved = king.HasMoved
						p.HasCastled = king.HasCastled
						p.HasCheck = king.HasCheck
					}
				}

				newBoard[i][j].Piece = newPiece
			}
		}
	}
	return newBoard
}
