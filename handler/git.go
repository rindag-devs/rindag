package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"rindag/model"
	"rindag/service/db"
	"rindag/service/git"
	"rindag/service/problem"
	"rindag/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var routes = []struct {
	re      *regexp.Regexp
	handler func(*gin.Context, string, string)
}{
	{regexp.MustCompile(`^/info/refs$`), handleGetInfoRefs},
	{regexp.MustCompile(`^/HEAD$`), handleGetTextFile},
	{regexp.MustCompile(`^/objects/info/alternates$`), handleGetTextFile},
	{regexp.MustCompile(`^/objects/info/http-alternates$`), handleGetTextFile},
	{regexp.MustCompile(`^/objects/info/packs$`), handleGetInfoPacks},
	{regexp.MustCompile(`^/objects/info/[^/]*$`), handleGetTextFile},
	{regexp.MustCompile(`^/objects/[0-9a-f]{2}/[0-9a-f]{38}$`), handleGetLooseObject},
	{regexp.MustCompile(`^/objects/pack/pack-[0-9a-f]{40}\.pack$`), handleGetPackFile},
	{regexp.MustCompile(`^/objects/pack/pack-[0-9a-f]{40}\.idx$`), handleGetIdxFile},
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

// getRepo returns the repo path or creates a new repo.
// The first return value is the path of the repo.
// The second return value is true if no error occurs.
func getRepo(c *gin.Context) (string, bool) {
	// Remove redundant suffix ".git".
	repoName := strings.TrimSuffix(c.Param("repo"), ".git")
	if repoName == "" {
		log.Warn("repo name is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo is required"})
		return "", false
	}

	problemID, err := uuid.Parse(repoName)
	if err != nil {
		log.WithError(err).WithField("repoName", repoName).Warn("repo name is not a valid uuid")
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo is not a valid uuid"})
		return "", false
	}

	if _, err := model.GetProblemByID(db.PDB, problemID); err != nil {
		log.WithError(err).Error("failed to get problem")
		c.JSON(http.StatusNotFound, gin.H{"error": "repo is not found"})
		return "", false
	}

	prob := problem.NewProblem(problemID)
	if _, err := prob.GetOrInitRepo(); err != nil {
		log.WithError(err).Error("failed to get or init repo")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return "", false
	}

	return git.GetRepoPath(problemID.String()), true
}

// handleRPC handles the git rpc.
func handleRPC(c *gin.Context, service string) {
	utils.SetHeaderNoCache(c)

	if c.GetHeader("Content-Type") != fmt.Sprintf("application/x-git-%s-request", service) {
		log.Warnf("invalid content-type: %s", c.GetHeader("Content-Type"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content-type"})
		return
	}

	var err error
	reqBody := c.Request.Body

	// Handle GZIP.
	if c.GetHeader("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			log.WithError(err).Error("failed to create gzip reader")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	repoPath, ok := getRepo(c)
	if !ok {
		return
	}

	// Set this for allow pre-receive and post-receive execute.
	env := os.Environ()
	env = append(env, "SSH_ORIGINAL_COMMAND="+service)

	var stderr bytes.Buffer
	cmd, pipe := git.NewCommand(repoPath, service, "--stateless-rpc", repoPath)
	cmd.Stdin = reqBody
	cmd.Env = env
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer utils.CleanUpProcessGroup(cmd)

	data, _ := io.ReadAll(pipe)
	c.Data(http.StatusOK, fmt.Sprintf("application/x-git-%s-result", service), data)

	if err := cmd.Wait(); err != nil {
		log.WithError(err).Error("failed to wait")
		return
	}
}

// handleGetInfoRefs returns the git info refs.
func handleGetInfoRefs(c *gin.Context, repoPath string, _ string) {
	utils.SetHeaderNoCache(c)

	rpc := c.Query("service")
	log.Debugf("rpc: %s", rpc)
	if rpc != "git-upload-pack" && rpc != "git-receive-pack" {
		cmd, _ := git.NewCommand(repoPath, "update-server-info")
		if err := cmd.Start(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := cmd.Wait(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		utils.SendLocalFile(c, path.Join(repoPath, "info/refs"), "text/plain; charset=utf-8")
		return
	}

	rpc = strings.TrimPrefix(rpc, "git-")
	cmd, pipe := git.NewCommand(repoPath, rpc, "--stateless-rpc", "--advertise-refs", repoPath)
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer utils.CleanUpProcessGroup(cmd)

	w := bytes.Buffer{}
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = w.Write(packetWrite("# service=git-" + rpc + "\n"))
	_, _ = w.Write([]byte("0000"))
	if _, err := io.Copy(&w, pipe); err != nil {
		log.WithError(err).Error("failed to copy pipe")
		return
	}

	c.Data(http.StatusOK, fmt.Sprintf("application/x-git-%s-advertisement", rpc), w.Bytes())

	if err := cmd.Wait(); err != nil {
		log.WithError(err).Error("failed to wait")
		return
	}
}

// handleGetTextFile returns the file.
func handleGetTextFile(c *gin.Context, repoPath string, url string) {
	utils.SetHeaderNoCache(c)
	utils.SendLocalFile(c, path.Join(repoPath, url), "text/plain")
}

// handleGetInfoPacks returns the git info packs.
func handleGetInfoPacks(c *gin.Context, repoPath string, url string) {
	utils.SetHeaderCacheForever(c)
	utils.SendLocalFile(c, path.Join(repoPath, url), "text/plain; charset=utf-8")
}

// handleGetLooseObject returns the loose object.
func handleGetLooseObject(c *gin.Context, repoPath string, url string) {
	utils.SetHeaderCacheForever(c)
	utils.SendLocalFile(c, path.Join(repoPath, url), "application/x-git-loose-object")
}

// handleGetPackFile returns the pack file.
func handleGetPackFile(c *gin.Context, repoPath string, url string) {
	utils.SetHeaderCacheForever(c)
	utils.SendLocalFile(c, path.Join(repoPath, url), "application/x-git-packed-objects")
}

// handleGetIdxFile returns the index file.
func handleGetIdxFile(c *gin.Context, repoPath string, url string) {
	utils.SetHeaderCacheForever(c)
	utils.SendLocalFile(c, path.Join(repoPath, url), "application/x-git-packed-objects-toc")
}

// HandleGitUploadPack returns the git upload pack response.
//
// This API is used by the git client.
func HandleGitUploadPack(c *gin.Context) {
	handleRPC(c, "upload-pack")
}

// HandleGitReceivePack returns the git upload pack response.
//
// This API is used by the git client.
func HandleGitReceivePack(c *gin.Context) {
	handleRPC(c, "receive-pack")
}

// HandleGitGet handles the GET request.
// This API is used by the git client.
func HandleGitGet(c *gin.Context) {
	url := c.Param("url")
	for _, route := range routes {
		if !route.re.MatchString(url) {
			continue
		}
		log.WithField("url", url).WithField("reg", route.re.String()).Debug("matched")
		repoPath, ok := getRepo(c)
		if !ok {
			return
		}
		route.handler(c, repoPath, url)
		return
	}
	log.Warn("unhandled GET request: ", url)
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
