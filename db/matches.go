package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
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

type MatchDetail struct {
	ID        string     `json:"id"`
	WhiteID   string     `json:"whiteId"`
	WhiteName string     `json:"whiteName"`
	BlackID   string     `json:"blackId"`
	BlackName string     `json:"blackName"`
	Winner    *string    `json:"winner"`
	Status    string     `json:"status"`
	Moves     []string   `json:"moves"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt"`
}

func scanMatchDetail(row pgx.Row) (*MatchDetail, error) {
	var m MatchDetail
	var movesRaw []byte
	if err := row.Scan(&m.ID, &m.WhiteID, &m.WhiteName, &m.BlackID, &m.BlackName,
		&m.Winner, &m.Status, &movesRaw, &m.StartedAt, &m.EndedAt); err != nil {
		return nil, err
	}
	if len(movesRaw) == 0 {
		m.Moves = []string{}
	} else if err := json.Unmarshal(movesRaw, &m.Moves); err != nil {
		return nil, err
	}
	return &m, nil
}

const matchDetailSelect = `
	SELECT m.id, m.white_id, wu.name, m.black_id, bu.name,
	       m.winner, m.status, m.moves, m.started_at, m.ended_at
	FROM matches m
	JOIN users wu ON wu.id = m.white_id
	JOIN users bu ON bu.id = m.black_id
`

func GetMatchByID(ctx context.Context, matchID string) (*MatchDetail, error) {
	row := Pool.QueryRow(ctx, matchDetailSelect+` WHERE m.id = $1`, matchID)
	return scanMatchDetail(row)
}

func GetUserMatches(ctx context.Context, userID string) ([]MatchDetail, error) {
	rows, err := Pool.Query(ctx, matchDetailSelect+`
		WHERE m.white_id = $1 OR m.black_id = $1
		ORDER BY m.started_at DESC LIMIT 20
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []MatchDetail{}
	for rows.Next() {
		m, err := scanMatchDetail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
