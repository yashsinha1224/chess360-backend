package db

import (
	"context"
	"encoding/json"
)

func CreateMatch(ctx context.Context, matchID, whiteID, blackID, status string) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO matches (id, white_id, black_id, status, moves)
		VALUES ($1, $2, $3, $4, '[]')
	`, matchID, whiteID, blackID, status)
	return err
}

func FinishMatch(ctx context.Context, matchID, winner, status string, moves []string) error {
	movesJSON, err := json.Marshal(moves)
	if err != nil {
		return err
	}
	_, err = Pool.Exec(ctx, `
		UPDATE matches
		SET winner = $2, status = $3, moves = $4, ended_at = now()
		WHERE id = $1
	`, matchID, winner, status, string(movesJSON))
	return err
}
