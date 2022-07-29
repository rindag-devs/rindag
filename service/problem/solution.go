package problem

import (
	"io"

	mapset "github.com/deckarep/golang-set/v2"
)

type SolutionType uint32

const (
	SolutionTypeUnknown SolutionType = iota
	SolutionTypeAccepted
	SolutionTypeWrongAnswer
	SolutionTypeBruteForce
)

// Solution is a solution to a problem.
type Solution struct {
	// Problem is the problem to which the solution belongs.
	Problem *Problem
	// Rev is the revision of the repository which the solution belongs.
	Rev int64
	// Name is the file name of the solution.
	Name string
	// Type is the type of solution.
	Type SolutionType
	// Subtasks is a set of subtasks that this solution solves.
	Subtasks mapset.Set[int32]
}

// SourceReadCloser returns a ReadCloser of the solution source.
func (s *Solution) SourceReadCloser() (io.ReadCloser, error) {
	repo, err := s.Problem.Repo()
	if err != nil {
		return nil, err
	}
	return repo.FileContent(s.Name, s.Rev)
}
