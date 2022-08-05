package problem

import (
	"io"
	"path"
	"time"

	"rindag/service/etc"
	"rindag/service/git"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/google/uuid"
)

// Problem represents a problem.
type Problem struct {
	ID uuid.UUID
}

// Repo returns the repository of the problem.
func (p *Problem) Repo() (*gogit.Repository, error) {
	return git.OpenRepo(p.ID.String())
}

// File returns a ReadCloser of the file of the problem.
func (p *Problem) File(path string, rev [20]byte) (io.ReadCloser, error) {
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

	repo, err := gogit.Init(
		filesystem.NewStorage(osfs.New(repoPath), cache.NewObjectLRUDefault()), memfs.New())
	if err != nil {
		return nil, err
	}

	if err := repo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")); err != nil {
		return nil, err
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	fs := w.Filesystem

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

	if _, err := w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "RinDAG",
			Email: "system@rindag.local",
			When:  now,
		},
	}); err != nil {
		return nil, err
	}

	return repo, nil
}

// GetOrInitRepo gets or initializes the repository of the problem.
func (p *Problem) GetOrInitRepo() (*gogit.Repository, error) {
	if git.RepoExists(p.ID.String()) {
		return git.OpenRepo(p.ID.String())
	} else {
		return p.initRepo()
	}
}

// NewProblem creates a new problem.
func NewProblem(id uuid.UUID) *Problem {
	return &Problem{ID: id}
}
