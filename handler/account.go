package handler

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"
	"rindag/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// @summary     Login
// @description Returns a JWT token if the credentials are correct.
// @tags        account
// @accept      json
// @produce     json
// @param       account body     string true "User account"
// @param       account body     string true "User password (unencrypted plaintext)"
// @success     200     {object} any{token=string}
// @failure     400     {object} any{error=string}
// @failure     500     {object} any{error=string} "Generate Token Failed"
// @router      /login [post]
func HandleLogin(c *gin.Context) {
	account := c.GetString("account")
	password := c.GetString("password")

	user, err := model.GetUser(db.PDB, account)
	if err != nil {
		// We don't want to leak the fact that the user doesn't exist.
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.ValidatePassword(password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// @summary     Logout
// @description Logs out the user and deletes the JWT token.
// @tags        account
// @produce     json
// @success     200 {object} any{message=string}
// @security    ApiKeyAuth
// @router      /logout [delete]
func HandleLogout(c *gin.Context) {
	// _user was already set in the middleware and it must be valid.
	userID, _ := c.MustGet("_user").(uuid.UUID)
	user, err := model.GetUserById(db.PDB, userID)
	if err != nil {
		// It is almost impossible to get here.
		log.WithError(err).Panic("Failed to get user")
	}
	user.ExpireToken(db.PDB)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
