package handler

import (
	"io"
	"net/http"

	"rindag/service/judge"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// @summary     IdleJudge
// @description Get an idle judge and return its ID.
// @tags        judge
// @produce     json
// @success     200 {object} any{judge=string}
// @success     500 {object} any{error=string}
// @security    ApiKeyAuth
// @router      /judge/idle [get]
func HandleIdleJudge(c *gin.Context) {
	judgeID, _, err := judge.GetIdleJudge()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"judge": judgeID})
}

// @summary     JudgeFileList
// @description List all cached files of the judge.
// @tags        judge
// @accept      json
// @produce     json
// @param       judge_id path     string true "Judge ID"
// @success     200      {object} any{files=[]map[string]string}
// @failure     400      {object} any{error=string}
// @failure     500      {object} any{error=string}
// @security    ApiKeyAuth
// @router      /judge/file/{judge_id} [get]
func HandleJudgeFileList(c *gin.Context) {
	judgeID := c.Param("judge_id")

	j, err := judge.GetJudge(judgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	files, err := j.FileList(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// @summary     JudgeFileGet
// @description Get cached file from the judge.
// @tags        judge
// @accept      json
// @produce     octet-stream
// @param       judge_id path     string true "Judge ID"
// @param       file_id  path     string true "File ID"
// @success     200      {object} []byte
// @failure     400      {object} any{error=string}
// @failure     404      {object} any{error=string}
// @failure     500      {object} any{error=string}
// @security    ApiKeyAuth
// @router      /judge/file/{judge_id}/{file_id} [get]
func HandleJudgeFileGet(c *gin.Context) {
	judgeID := c.Param("judge_id")
	fileID := c.Param("file_id")

	j, err := judge.GetJudge(judgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	file, err := j.FileGet(ctx, fileID)
	if err != nil {
		status, ok := status.FromError(err)
		if !ok || status.Code() != codes.NotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if _, err := c.Writer.Write(file.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// @summary     JudgeFileAdd
// @description Add a file to the judge and returns its id.
// @tags        judge
// @accept      json
// @produce     json
// @param       judge_id path     string true "Judge ID"
// @param       content  formData file   true "Content"
// @success     200      {object} any{file=string}
// @failure     400      {object} any{error=string}
// @failure     500      {object} any{error=string}
// @security    ApiKeyAuth
// @router      /judge/file/{judge_id}/ [post]
func HandleJudgeFileAdd(c *gin.Context) {
	judgeID := c.Param("judge_id")

	fileHander, err := c.FormFile("content")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	j, err := judge.GetJudge(judgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	file, err := fileHander.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileID, err := j.FileAdd(ctx, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": fileID})
}

// @summary     JudgeFileDelete
// @description Delete a file from the judge.
// @tags        judge
// @accept      json
// @produce     json
// @param       judge_id path     string true "Judge ID"
// @param       file_id  path     string true "File ID"
// @success     200      {object} any{message=string}
// @failure     400      {object} any{error=string}
// @failure     500      {object} any{error=string}
// @security    ApiKeyAuth
// @router      /judge/file/{judge_id}/{file_id} [delete]
func HandleJudgeFileDelete(c *gin.Context) {
	judgeID := c.Param("judge_id")
	fileID := c.Param("file_id")

	j, err := judge.GetJudge(judgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err = j.FileDelete(ctx, fileID)
	if err != nil {
		status, ok := status.FromError(err)
		if !ok || status.Code() != codes.NotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		// Can not find such file
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}
