package api

import (
	"net/http"

	"example/hello/db"

	"github.com/gin-gonic/gin"
)

func GetMatch(c *gin.Context) {
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
	c.JSON(http.StatusOK, match)
}

func ListMyMatches(c *gin.Context) {
	matches, err := db.GetUserMatches(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch matches"})
		return
	}
	c.JSON(http.StatusOK, matches)
}
