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
		log.Debug(c.GetHeader("X-API-KEY"))
		token, err := utils.ParseToken(c.GetHeader("X-API-KEY"))
		if err != nil {
			log.WithError(err).Error("Error parsing token")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("_user", token.ID)
		c.Next()
	}
}
