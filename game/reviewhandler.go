package game

import (
	"net/http"

	"example/hello/eval"
	"example/hello/types"

	"github.com/gin-gonic/gin"
)

func ReviewGameHandler(hub *types.Hub) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		gameID := ctx.Param("id")
		if gameID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing game id"})
			return
		}

		hub.Mu.RLock()
		g, ok := hub.Games[gameID]
		hub.Mu.RUnlock()
		if !ok {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}

		reviews, err := eval.ReviewMoves(g.Moves)
		if err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"gameId": gameID,
			"moves":  reviews,
		})
	}
}
