package types

type Color string
type Piece string
type Position struct {
	Row int
	Col int
}

const (
	White Color = "white"
	Black Color = "black"
)

const (
	King   Piece = "k"
	Rook   Piece = "r"
	Queen  Piece = "q"
	Bishop Piece = "b"
	Knight Piece = "n"
	Pawn   Piece = "p"
)

type ChessPiece interface {
	GetType() Piece
	GetColor() Color
}

type BasePiece struct {
	Type     Piece
	Color    Color
	HasMoved bool
}
type BoardSquare struct {
	Piece    ChessPiece
	Position Position
}

func (b *BasePiece) GetType() Piece  { return b.Type }
func (b *BasePiece) GetColor() Color { return b.Color }

type KingPiece struct {
	BasePiece
	HasCastled bool
	HasCheck   bool
}

type PawnPiece struct{ BasePiece }
type QueenPiece struct{ BasePiece }
type BishopPiece struct{ BasePiece }
type KnightPiece struct{ BasePiece }
type RookPiece struct{ BasePiece }

func MakePiece(c Color, p Piece) ChessPiece {
	base := BasePiece{Type: p, Color: c}
	switch p {
	case King:
		return &KingPiece{BasePiece: base}
	case Queen:
		return &QueenPiece{BasePiece: base}
	case Rook:
		return &RookPiece{BasePiece: base}
	case Bishop:
		return &BishopPiece{BasePiece: base}
	case Knight:
		return &KnightPiece{BasePiece: base}
	case Pawn:
		return &PawnPiece{BasePiece: base}
	default:
		return nil
	}
}
func MakeBoard(c Color, p Piece, pos Position) BoardSquare {
	return BoardSquare{Piece: MakePiece(c, p), Position: pos}
}
func SquareColor(pos Position) Color {
	if (pos.Row+pos.Col)%2 == 0 {
		return White
	}
	return Black
}
