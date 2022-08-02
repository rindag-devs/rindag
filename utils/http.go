package utils

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// SetHeaderNoCache sets the no cache header.
func SetHeaderNoCache(c *gin.Context) {
	c.Header("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	c.Header("Pragma", "no-cache")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
}

// SetHeaderCacheForever sets the cache forever header.
func SetHeaderCacheForever(c *gin.Context) {
	now := time.Now().Unix()
	expires := now + 31536000
	c.Header("Date", fmt.Sprintf("%d", expires))
	c.Header("Expires", fmt.Sprintf("%d", expires))
	c.Header("Cache-Control", "public, max-age=31536000")
}

// SendLocalFile sends the file to the client.
func SendLocalFile(c *gin.Context, path string, contentType string) {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", fi.Size()))
	c.Header("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	c.File(path)
}
