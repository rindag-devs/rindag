package middleware

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"

	"github.com/gin-gonic/gin"
)

// GitAuthMiddleware is a middleware that validates the user and password for git.
func GitAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", "Basic realm=\".\"")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		account, password, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		user, err := model.GetUser(db.PDB, account)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !user.Authenticate(password) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("_user", user.ID)
		c.Next()
	}
}
