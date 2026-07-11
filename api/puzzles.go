package api

import (
	"net/http"
	"strings"

	"example/hello/db"

	"github.com/gin-gonic/gin"
)

func GetNextPuzzle(c *gin.Context) {
	userID := c.GetString("userID")
	u, err := db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
		return
	}
	p, err := db.GetPuzzleForUser(c.Request.Context(), userID, u.PuzzleRating)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no puzzles available"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type puzzleAttemptReq struct {
	Moves []string `json:"moves"` // UCI moves the user played, in order
}

func SubmitPuzzleAttempt(c *gin.Context) {
	userID := c.GetString("userID")
	var req puzzleAttemptReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	puzzle, err := db.GetPuzzleByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "puzzle not found"})
		return
	}

	solution := strings.Fields(puzzle.Moves)[1:] // skip opponent's setup move
	solved := len(req.Moves) == len(solution)
	if solved {
		for i, m := range req.Moves {
			if m != solution[i] {
				solved = false
				break
			}
		}
	}

	newRating, err := db.RecordPuzzleAttempt(c.Request.Context(), userID, puzzle, solved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record attempt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"solved": solved, "puzzleRating": newRating})
}
