package problem

import (
	"bytes"
	"embed"
	"io"

	"rindag/service/etc"
	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

// Checker is a checker for special judge.
type Checker struct {
	// binaryID is the ID of the checker binary.
	//
	// If the checker is not compiled, the binaryID will be nil.
	binaryID *string

	// GetSource is a function returns the source code ReadCloser of the checker.
	GetSource func() (io.ReadCloser, error)
}

// NewChecker creates a checker.
func NewChecker(getSource func() (io.ReadCloser, error)) *Checker {
	return &Checker{
		binaryID:  new(string),
		GetSource: getSource,
	}
}

//go:embed third_party/testlib/checkers/*
var builtinCheckersFS embed.FS

// BuiltinChecker creates a checker from the builtin checkers.
// Available builtin checkers:
//   - "wcmp" : Compare sequences of tokens (default).
//   - "lcmp" : Compare files as sequence of tokens in lines.
//   - "yesno" : Compare one token "YES" or "NO" (case insensitive).
//   - "nyesno" : Like "yesno", but multiple tokens are allowed.
//   - Other checkers in "third_party/testlib/checkers/".
func BuiltinChecker(name string) *Checker {
	return NewChecker(func() (io.ReadCloser, error) { return builtinCheckersFS.Open(name) })
}

// NewCheckerFromProblem creates a checker from a problem.
func NewCheckerFromProblem(problem *Problem, rev [20]byte, path string) *Checker {
	return NewChecker(func() (io.ReadCloser, error) { return problem.File(path, rev) })
}

// NewCheckerFromBytes creates a checker from the source code.
func NewCheckerFromBytes(source []byte) *Checker {
	return NewChecker(
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(source)), nil })
}

// NewCheckerFromReadCloser creates a checker from the ReadCloser.
func NewCheckerFromReadCloser(r io.ReadCloser) *Checker {
	return NewChecker(func() (io.ReadCloser, error) { return r, nil })
}

// CompileTask returns the compile task of the checker.
func (c *Checker) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := c.GetSource()
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
		WithCmd(conf.Checker.Compile.Args...).
		WithCmd("checker.cpp", "-o", "checker").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("checker.cpp", code).
		WithCopyIn("testlib.h", TestlibSource).
		WithCopyOut("checker").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; finished {
				ok := false
				if *c.binaryID, ok = r.FileIDs["checker"]; !ok {
					// Impossible to happen.
					log.Fatal("checker compile successful, but binary ID not found")
				}
			}
			return cb(r, err)
		}), nil
}

// CheckTask needs a checker binary file ID, an input file, and output file, and a standard answer.
// Returns a judge task to run the checker.
func (c *Checker) CheckTask(inf *pb.Request_File, ouf *pb.Request_File, ans *pb.Request_File,
	cb judge.CallbackFunction,
) *judge.Task {
	conf := &etc.Config.Checker
	return judge.DefaultTask().
		WithCmd("checker", "input.txt", "output.txt", "answer.txt").
		WithTimeLimit(conf.Run.TimeLimit).
		WithMemoryLimit(conf.Run.MemoryLimit).
		WithStderrLimit(conf.Run.StderrLimit).
		WithCopyInCached("checker", c.binaryID).
		WithCopyInFile("input.txt", inf).
		WithCopyInFile("output.txt", ouf).
		WithCopyInFile("answer.txt", ans).
		WithCallback(cb)
}
