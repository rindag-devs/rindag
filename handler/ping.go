package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandlePing handles ping requests.
func HandlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
