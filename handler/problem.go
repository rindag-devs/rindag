package handler

import (
	"net/http"

	"rindag/model"
	"rindag/service/db"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// @summary     ProblemList
// @description List all problems.
// @tags        problem
// @accept      json
// @produce     json
// @success     200 {object} any{problems=[]model.Problem}
// @failure     500 {object} any{error=string}
// @security    ApiKeyAuth
// @router      /problem [get]
func HandleProblemList(c *gin.Context) {
	problems, err := model.ListProblems(db.PDB)
	if err != nil {
		log.WithError(err).Error("failed to list problems")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list problems"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problems": problems})
}

type problemAddReq struct {
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Badnames bool     `json:"bad_name"`
}

// @summary     ProblemAdd
// @description Add a problem and returns its id.
// @tags        problem
// @accept      json
// @produce     json
// @param       problemAddReq body     problemAddReq true "If badnames is true, it will not check for bad names"
// @success     200           {object} any{problem=uuid.UUID}
// @failure     400           {object} any{error=string}
// @failure     500           {object} any{error=string}
// @security    ApiKeyAuth
// @router      /problem/ [post]
func HandleProblemAdd(c *gin.Context) {
	params := problemAddReq{Badnames: false}

	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem, err := model.CreateProblem(db.PDB, params.Name, params.Tags)
	if err != nil {
		log.WithError(err).Error("failed to create problem")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problem": problem.ID})
}
