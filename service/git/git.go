package git

import (
	"io"
	"os"
	"os/exec"
	"path"
	"syscall"

	"rindag/service/etc"

	gogit "github.com/go-git/go-git/v5"
)

// GetRepoPath returns the path to the git repository.
func GetRepoPath(repo string) string {
	return path.Join(etc.Config.Git.RepoDir, repo)
}

// RepoExists returns true if the git repository exists.
func RepoExists(repo string) bool {
	path := GetRepoPath(repo)
	// check the directory exists
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

// OpenRepo opens the git repository.
func OpenRepo(repo string) (*gogit.Repository, error) {
	path := GetRepoPath(repo)
	return gogit.PlainOpen(path)
}

// NewCommand returns the git command and a pipe of it's stdout.
func NewCommand(dir string, args ...string) (*exec.Cmd, io.Reader) {
	cmd := exec.Command("git", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = os.Environ()
	cmd.Dir = dir

	r, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	return cmd, r
}
