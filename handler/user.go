package handler

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"
	"rindag/utils"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// @Summary Login
// @Description returns a JWT token if the credentials are correct
// @Accept json
// @Produce json
// @Param login body loginReq true "Login request"
// @Success 200 {object} json "{"token": "..."}"
// @Failure 400 {object} json "{"error": "..."}"
// @Failure 500 {object} json "{"error": "..."}"
// @Router /login [post]
func HandleLogin(c *gin.Context) {
	var login loginReq
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := model.GetUser(db.PDB, login.User)
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

// @Summary Logout
// @Description logs out the user and deletes the JWT token
// @Accept json
// @Produce json
// @Success 200 {object} json "{"message": "logged out"}"
// @Failure 401 {object} json "{"error": "..."}"
// @Router /logout [delete]
func HandleLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
