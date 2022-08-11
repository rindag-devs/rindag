package problem

import (
	"io"
	"path"
	"time"

	"rindag/service/etc"
	"rindag/service/git"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Problem represents a problem.
type Problem struct {
	ID uuid.UUID
}

// NewProblem creates a new problem.
func NewProblem(id uuid.UUID) *Problem {
	return &Problem{ID: id}
}

// File returns a ReadCloser of the file of the problem.
func (p *Problem) File(rev [20]byte, path string) (io.ReadCloser, error) {
	repo, err := p.Repo()
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(rev)
	if err != nil {
		return nil, err
	}

	file, err := commit.File(path)
	if err != nil {
		return nil, err
	}

	return file.Reader()
}

func (p *Problem) initRepo() (*gogit.Repository, error) {
	now := time.Now()
	repoPath := git.GetRepoPath(p.ID.String())

	// Init a bare repository.
	sRepo, err := gogit.PlainInit(repoPath, true)
	if err != nil {
		log.WithError(err).Error("failed to init repo")
		return nil, err
	}

	// Link HEAD to the main branch.
	if err := sRepo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")); err != nil {
		log.WithError(err).Error("failed to set HEAD")
		return nil, err
	}

	// You can not edit the worktree of a bare repository.
	// So we init a new repository in memory, edit it, and push it back.
	repo, err := gogit.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		log.WithError(err).Error("failed to clone repo")
		return nil, err
	}

	// Link HEAD to the main branch.
	if err := repo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")); err != nil {
		log.WithError(err).Error("failed to set HEAD")
		return nil, err
	}

	w, err := repo.Worktree()
	if err != nil {
		log.WithError(err).Error("failed to get worktree")
		return nil, err
	}
	fs := w.Filesystem

	// Create default files in the worktree.
	for pa, da := range etc.Config.Problem.InitialWorktree {
		if err := fs.MkdirAll(path.Dir(pa), 0o755); err != nil {
			return nil, err
		}
		file, err := fs.Create(pa)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		if _, err = file.Write([]byte(da)); err != nil {
			return nil, err
		}
		if _, err := w.Add(pa); err != nil {
			return nil, err
		}
	}

	// Make a commit.
	if _, err := w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "RinDAG",
			Email: "system@rindag.local",
			When:  now,
		},
	}); err != nil {
		log.WithError(err).Error("failed to commit")
		return nil, err
	}

	remote, err := repo.CreateRemoteAnonymous(&config.RemoteConfig{
		Name: "anonymous",
		URLs: []string{repoPath},
	})
	if err != nil {
		log.WithError(err).Error("failed to create remote")
		return nil, err
	}

	if err := remote.Push(&gogit.PushOptions{
		RemoteName: "anonymous",
	}); err != nil {
		log.WithError(err).Error("failed to push")
		return nil, err
	}

	return sRepo, nil
}

// Repo gets or initializes the repository of the problem.
func (p *Problem) Repo() (*gogit.Repository, error) {
	if git.RepoExists(p.ID.String()) {
		return git.OpenRepo(p.ID.String())
	} else {
		return p.initRepo()
	}
}
