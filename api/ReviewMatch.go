package api

import (
	"net/http"

	"example/hello/db"
	"example/hello/eval"

	"github.com/gin-gonic/gin"
)

func ReviewMatch(c *gin.Context) {
	userID := c.GetString("userID")
	match, err := db.GetMatchByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "match not found"})
		return
	}
	if match.WhiteID != userID && match.BlackID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your match"})
		return
	}

	reviews, err := eval.ReviewMoves(match.Moves)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matchId": match.ID,
		"moves":   reviews,
	})
}
