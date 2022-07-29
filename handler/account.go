package handler

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"
	"rindag/utils"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Account  string `json:"user"`
	Password string `json:"password"`
}

// @summary Login
// @description returns a JWT token if the credentials are correct
// @tags account
// @accept json
// @produce json
// @param login body loginReq true "Login request"
// @success 200 {object} json "{"token": "..."}"
// @failure 400 {object} json "{"error": "..."}"
// @failure 500 {object} json "{"error": "..."}"
// @router /login [post]
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

	if user.Password != login.Password {
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

// @summary Logout
// @description logs out the user and deletes the JWT token
// @accept json
// @produce json
// @success 200 {object} json "{"message": "logged out"}"
// @failure 401 {object} json "{"error": "..."}"
// @router /logout [delete]
func HandleLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
