package game

import (
	"context"
	"net/http"

	"example/hello/eval"

	"github.com/gin-gonic/gin"
)

func MatchReviewHandler(getMatchMoves func(ctx context.Context, matchID string) ([]string, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		matchID := ctx.Param("id")
		if matchID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing match id"})
			return
		}

		moves, err := getMatchMoves(ctx.Request.Context(), matchID)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "match not found"})
			return
		}

		reviews, err := eval.ReviewMoves(moves)
		if err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"matchId": matchID,
			"moves":   reviews,
		})
	}
}
