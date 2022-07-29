package handler

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"
	"rindag/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @param account  body string true "Account credentials"
// @param password body string true "Account credentials"
type loginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// @summary     Login
// @description Returns a JWT token if the credentials are correct.
// @tags        account
// @accept      json
// @produce     json
// @param       loginReq body     loginReq true "Account credentials"
// @success     200      {object} any{token=string}
// @failure     400      {object} any{error=string}
// @failure     500      {object} any{error=string} "Generate Token Failed"
// @router      /login [post]
func HandleLogin(c *gin.Context) {
	var login loginReq
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := model.GetUser(db.PDB, login.Account)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !user.ValidatePassword(login.Password) {
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
// @accept      json
// @produce     json
// @success     200 {object} any{message=string}
// @failure     401 {object} any{error=string}
// @security    ApiKeyAuth
// @router      /logout [delete]
func HandleLogout(c *gin.Context) {
	userID, _ := c.MustGet("_user").(uuid.UUID)
	user, _ := model.GetUserById(db.PDB, userID)
	user.ExpireToken(db.PDB)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
