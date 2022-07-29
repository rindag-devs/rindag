package middleware

import (
	"net/http"

	"rindag/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// JWTMiddleware is a middleware that validates a JWT token.
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			log.WithError(err).Error("Error getting token from cookie")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if _, err := utils.ParseToken(token); err != nil {
			log.WithError(err).Error("Error parsing token")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}
