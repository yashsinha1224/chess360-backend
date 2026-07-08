package game

import (
	"example/hello/types"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func NewHub() *types.Hub {
	return &types.Hub{
		Games:   make(map[string]*types.Game),
		Players: make(map[string]*types.Player),
		Buckets: make(map[int][]*types.Player),
		Mu:      sync.RWMutex{},
	}
}

func addWaiting(hub *types.Hub, p *types.Player) {
	hub.Mu.Lock()
	defer hub.Mu.Unlock()

	if p.IsInGame {
		fmt.Printf("[REJECT] Player %s is already in a game\n", p.Name)
		return
	}

	elo_bucket := p.ElO / 100
	hub.Buckets[elo_bucket] = append(hub.Buckets[elo_bucket], p)
	fmt.Printf("[QUEUE] %s added to bucket %d\n", p.Name, elo_bucket)
}

func removeFromBucket(hub *types.Hub, p *types.Player) {
	bucket := p.ElO / 100
	for i, player := range hub.Buckets[bucket] {
		if player == p {
			hub.Buckets[bucket] = append(hub.Buckets[bucket][:i], hub.Buckets[bucket][i+1:]...)
			fmt.Printf("[QUEUE] %s removed from bucket %d\n", p.Name, bucket)
			break
		}
	}

	if len(hub.Buckets[bucket]) == 0 {
		delete(hub.Buckets, bucket)
	}
}

func matchmake(hub *types.Hub, p *types.Player) *types.Player {
	hub.Mu.Lock()
	defer hub.Mu.Unlock()

	bucket := p.ElO / 100
	bucketsToCheck := []int{bucket, bucket - 1, bucket + 1, bucket - 2, bucket + 2}

	for _, b := range bucketsToCheck {
		if len(hub.Buckets[b]) >= 1 {
			for i, opponent := range hub.Buckets[b] {
				if opponent != p && !opponent.IsInGame {
					// Remove opponent from bucket
					hub.Buckets[b] = append(hub.Buckets[b][:i], hub.Buckets[b][i+1:]...)
					fmt.Printf("[MATCH] %s matched with %s from bucket %d\n", p.Name, opponent.Name, b)
					return opponent
				}
			}
		}
	}
	return nil
}

func addplayer(hub *types.Hub, name string, conn *websocket.Conn) *types.Player {
	ID := uuid.New().String()
	elo := 1200
	player := &types.Player{
		ID:       ID,
		Name:     name,
		ElO:      elo,
		Gmail:    "",
		Conn:     conn,
		GameId:   "",
		IsInGame: false,
	}
	hub.Players[player.ID] = player
	fmt.Printf("[PLAYER] Created player: %s (ID: %s, ELO: %d)\n", name, ID, elo)
	return player
}

func CleanupPlayer(hub *types.Hub, player *types.Player) {
	hub.Mu.Lock()
	defer hub.Mu.Unlock()

	bucket := player.ElO / 100
	for i, p := range hub.Buckets[bucket] {
		if p == player {
			hub.Buckets[bucket] = append(hub.Buckets[bucket][:i], hub.Buckets[bucket][i+1:]...)
			break
		}
	}

	delete(hub.Players, player.ID)

	if len(hub.Buckets[bucket]) == 0 {
		delete(hub.Buckets, bucket)
	}

	fmt.Printf("[CLEANUP] Player %s completely removed from hub\n", player.Name)
}
