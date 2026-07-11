package db

import (
	"context"
	"example/hello/types"
)

func UpsertUserByGoogleID(ctx context.Context, googleID, email, name string) (*types.User, error) {
	row := Pool.QueryRow(ctx, `
		INSERT INTO users (google_id, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (google_id) DO UPDATE
			SET email = EXCLUDED.email, name = EXCLUDED.name
		RETURNING id, google_id, email, name, elo, created_at
	`, googleID, email, name)

	var u types.User
	if err := row.Scan(&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.Elo, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(ctx context.Context, id string) (*types.User, error) {
	row := Pool.QueryRow(ctx, `
		SELECT id, google_id, email, name, elo, puzzle_rating, created_at
		FROM users WHERE id = $1
	`, id)

	var u types.User
	if err := row.Scan(&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.Elo, &u.PuzzleRating, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
