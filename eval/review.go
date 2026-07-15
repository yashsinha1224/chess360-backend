package eval

import (
	"fmt"
	"strings"

	"example/hello/intiaton"
	"example/hello/types"
)

type Classification string

const (
	Blunder    Classification = "blunder"
	Mistake    Classification = "mistake"
	Inaccuracy Classification = "inaccuracy"
	Good       Classification = "good"
	Excellent  Classification = "excellent"
)

type MoveReview struct {
	Number         int            `json:"number"`
	Move           string         `json:"move"`
	Color          types.Color    `json:"color"`
	EvalBefore     int            `json:"evalBefore"`
	EvalAfter      int            `json:"evalAfter"`
	DeltaForMover  int            `json:"deltaForMover"`
	Classification Classification `json:"classification"`
}

func classify(deltaForMover int) Classification {
	switch {
	case deltaForMover <= -200:
		return Blunder
	case deltaForMover <= -90:
		return Mistake
	case deltaForMover <= -40:
		return Inaccuracy
	case deltaForMover >= 60:
		return Excellent
	default:
		return Good
	}
}
func ReviewMoves(moves []string) ([]MoveReview, error) {
	board := intiaton.InitializeBoard()
	reviews := make([]MoveReview, 0, len(moves))

	for i, mv := range moves {
		fr, fc, tr, tc, ok := parseMove(mv)
		if !ok {
			return nil, fmt.Errorf("move %d (%q): malformed", i+1, mv)
		}
		if fr < 0 || fr > 7 || fc < 0 || fc > 7 || tr < 0 || tr > 7 || tc < 0 || tc > 7 {
			return nil, fmt.Errorf("move %d (%q): out of range", i+1, mv)
		}
		if board[fr][fc].Piece == nil {
			return nil, fmt.Errorf("move %d (%q): no piece on source square", i+1, mv)
		}

		evalBefore := Evaluate(board)
		movingColor := board[fr][fc].Piece.GetColor()

		applyMove(board, fr, fc, tr, tc)

		evalAfter := Evaluate(board)

		delta := evalAfter - evalBefore
		if movingColor == types.Black {
			delta = -delta
		}

		reviews = append(reviews, MoveReview{
			Number:         i + 1,
			Move:           mv,
			Color:          movingColor,
			EvalBefore:     evalBefore,
			EvalAfter:      evalAfter,
			DeltaForMover:  delta,
			Classification: classify(delta),
		})
	}

	return reviews, nil
}

func applyMove(board [][]types.BoardSquare, fr, fc, tr, tc int) {
	piece := board[fr][fc].Piece
	board[tr][tc].Piece = piece
	board[fr][fc].Piece = nil

	switch p := board[tr][tc].Piece.(type) {
	case *types.PawnPiece:
		p.HasMoved = true
		if (tr == 0 && p.GetColor() == types.White) || (tr == 7 && p.GetColor() == types.Black) {
			board[tr][tc].Piece = types.MakePiece(p.GetColor(), types.Queen)
		}
	case *types.RookPiece:
		p.HasMoved = true
	case *types.KingPiece:
		p.HasMoved = true
	}
}

func parseMove(mv string) (fr, fc, tr, tc int, ok bool) {
	parts := strings.Split(mv, ",")
	if len(parts) != 2 {
		return 0, 0, 0, 0, false
	}
	fr, fc, ok1 := parseSquare(parts[0])
	tr, tc, ok2 := parseSquare(parts[1])
	if !ok1 || !ok2 {
		return 0, 0, 0, 0, false
	}
	return fr, fc, tr, tc, true
}

func parseSquare(pos string) (row, col int, ok bool) {
	if len(pos) != 2 {
		return 0, 0, false
	}
	col = int(pos[0] - 'a')
	row = 8 - int(pos[1]-'0')
	if col < 0 || col > 7 || row < 0 || row > 7 {
		return 0, 0, false
	}
	return row, col, true
}
