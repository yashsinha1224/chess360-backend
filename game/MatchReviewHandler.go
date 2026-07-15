package game

import (
	"context"
	"net/http"

	"example/hello/eval"

	"github.com/gin-gonic/gin"
)

// MatchReviewHandler powers GET /api/matches/:id/review -- the
// finished-game counterpart to ReviewGameHandler (which reads live
// games out of hub.Games). This one takes a function for fetching a
// completed match's stored move list, so it doesn't need to know your
// db package's exact types -- just wire it to whatever already backs
// your GET /api/matches/:id handler.
//
// Wire it up next to that existing route, e.g.:
//
//	router.GET("/api/matches/:id", matches.GetMatchHandler(db))
//	router.GET("/api/matches/:id/review", game.MatchReviewHandler(
//	    func(ctx context.Context, id string) ([]string, error) {
//	        m, err := db.GetMatch(ctx, id) // <- your real call
//	        if err != nil {
//	            return nil, err
//	        }
//	        return m.Moves, nil
//	    },
//	))
//
// The frontend's useReview hook expects the JSON shape this returns:
// {"matchId": "...", "moves": [{"number", "move", "color", "evalBefore",
// "evalAfter", "deltaForMover", "classification"}, ...]}.
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
