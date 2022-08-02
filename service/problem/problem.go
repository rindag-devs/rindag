package problem

import (
	"io"

	"rindag/service/git"

	gogit "github.com/go-git/go-git/v5"
	"github.com/google/uuid"
)

// Problem represents a problem.
type Problem struct {
	ID   uuid.UUID
	Name string
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

// ProblemList represents a list of problems.
func ProblemList() ([]*Problem, error) {
	// TODO
	return nil, nil
}
