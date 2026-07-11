package db

import (
	"context"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"
)

type Puzzle struct {
	ID     string `json:"id"`
	FEN    string `json:"fen"`
	Moves  string `json:"moves"`
	Rating int    `json:"rating"`
	Themes string `json:"themes"`
}

func GetPuzzleForUser(ctx context.Context, userID string, rating int) (*Puzzle, error) {
	for window := 50; window <= 400; window += 100 {
		row := Pool.QueryRow(ctx, `
			SELECT p.id, p.fen, p.moves, p.rating, p.themes
			FROM puzzles p
			WHERE p.rating BETWEEN $1 AND $2
			  AND NOT EXISTS (
			    SELECT 1 FROM puzzle_attempts a
			    WHERE a.puzzle_id = p.id AND a.user_id = $3
			  )
			ORDER BY random() LIMIT 1
		`, rating-window, rating+window, userID)

		var p Puzzle
		err := row.Scan(&p.ID, &p.FEN, &p.Moves, &p.Rating, &p.Themes)
		if err == nil {
			return &p, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	return nil, errors.New("no puzzles available")
}

func GetPuzzleByID(ctx context.Context, id string) (*Puzzle, error) {
	row := Pool.QueryRow(ctx, `SELECT id, fen, moves, rating, themes FROM puzzles WHERE id = $1`, id)
	var p Puzzle
	if err := row.Scan(&p.ID, &p.FEN, &p.Moves, &p.Rating, &p.Themes); err != nil {
		return nil, err
	}
	return &p, nil
}

func RecordPuzzleAttempt(ctx context.Context, userID string, puzzle *Puzzle, solved bool) (int, error) {
	var current int
	if err := Pool.QueryRow(ctx, `SELECT puzzle_rating FROM users WHERE id = $1`, userID).Scan(&current); err != nil {
		return 0, err
	}

	expected := 1.0 / (1.0 + math.Pow(10, float64(puzzle.Rating-current)/400.0))
	actual := 0.0
	if solved {
		actual = 1.0
	}
	newRating := current + int(math.Round(32*(actual-expected)))

	if _, err := Pool.Exec(ctx, `
		INSERT INTO puzzle_attempts (user_id, puzzle_id, solved) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, puzzle_id) DO NOTHING
	`, userID, puzzle.ID, solved); err != nil {
		return 0, err
	}
	if _, err := Pool.Exec(ctx, `UPDATE users SET puzzle_rating = $2 WHERE id = $1`, userID, newRating); err != nil {
		return 0, err
	}
	return newRating, nil
}
