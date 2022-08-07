package problem

import (
	"bytes"
	"io"

	"rindag/service/etc"
	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

// Solution is a solution to a problem.
type Solution struct {
	// binaryID is the ID of the solution binary.
	//
	// If the checker is not compiled, the binaryID will be nil.
	binaryID *string

	// GetSource is a function returns the source code ReadCloser of the checker.
	GetSource func() (io.ReadCloser, error)
}

// NewSolution creates a solution.
func NewSolution(getSource func() (io.ReadCloser, error)) *Solution {
	return &Solution{
		binaryID:  new(string),
		GetSource: getSource,
	}
}

// NewSolutionFromProblem creates a solution from a problem.
func NewSolutionFromProblem(problem *Problem, rev [20]byte, path string) *Solution {
	return NewSolution(func() (io.ReadCloser, error) { return problem.File(path, rev) })
}

// NewSolutionFromBytes creates a solution from the source code.
func NewSolutionFromBytes(source []byte) *Solution {
	return NewSolution(
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(source)), nil })
}

// NewSolutionFromReadCloser creates a solution from the ReadCloser.
func NewSolutionFromReadCloser(r io.ReadCloser) *Solution {
	return NewSolution(func() (io.ReadCloser, error) { return r, nil })
}

// CompileTask returns a compile task of the solution.
func (s *Solution) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := s.GetSource()
	if err != nil {
		return nil, err
	}
	defer source.Close()
	code, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	return judge.DefaultTask().
		WithCmd(conf.Compile.Cmd...).
		WithCmd("sol.cpp", "-o", "sol").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("sol.cpp", code).
		WithCopyOut("sol").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; finished {
				ok := false
				if *s.binaryID, ok = r.FileIDs["sol"]; !ok {
					// Impossible to happen.
					log.Fatal("checker compile successful, but binary ID not found")
				}
			}
			return cb(r, err)
		}), nil
}
