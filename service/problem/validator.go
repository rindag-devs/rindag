package problem

import (
	"bytes"
	"io"

	"rindag/service/etc"
	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

// Validator is a validator to the problem.
type Validator struct {
	// binaryID is the ID of the validator binary.
	//
	// If the validator is not compiled, the binaryID will be nil.
	binaryID *string

	// GetSource is a function returns the source code ReadCloser of the checker.
	GetSource func() (io.ReadCloser, error)
}

// NewValidator creates a validator.
func NewValidator(getSource func() (io.ReadCloser, error)) *Validator {
	return &Validator{binaryID: new(string), GetSource: getSource}
}

// NewValidatorFromProblem creates a validator from a problem.
func NewValidatorFromProblem(problem *Problem, rev [20]byte, path string) *Validator {
	return NewValidator(func() (io.ReadCloser, error) { return problem.File(rev, path) })
}

// NewValidatorFromBytes creates a validator from the source code.
func NewValidatorFromBytes(source []byte) *Validator {
	return NewValidator(
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(source)), nil })
}

// NewValidatorFromReadCloser creates a validator from the ReadCloser.
func NewValidatorFromReadCloser(r io.ReadCloser) *Validator {
	return NewValidator(func() (io.ReadCloser, error) { return r, nil })
}

// CompileTask returns the compile task of the validator.
func (v *Validator) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := v.GetSource()
	if err != nil {
		return nil, err
	}
	defer source.Close()
	bytes, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	return judge.DefaultTask().
		WithCmd(conf.Compile.Cmd...).
		WithCmd(conf.Validator.Compile.Args...).
		WithCmd("validator.cpp", "-o", "validator").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("validator.cpp", bytes).
		WithCopyIn("testlib.h", TestlibSource).
		WithCopyOut("validator").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; finished {
				ok := false
				if *v.binaryID, ok = r.FileIDs["validator"]; !ok {
					// Impossible to happen.
					log.Fatal("validator compile successful, but binary ID not found")
				}
			}
			return cb(r, err)
		}), nil
}

// ValidateTask needs a validator binary file ID and an input file which will be validated.
// Returns a judge task to run the validator.
func (v *Validator) ValidateTask(
	inf *pb.Request_File, args []string, cb judge.CallbackFunction,
) *judge.Task {
	conf := &etc.Config.Validator
	return judge.DefaultTask().
		WithCmd("validator").
		WithCmd(args...).
		WithTimeLimit(conf.Run.TimeLimit).
		WithMemoryLimit(conf.Run.MemoryLimit).
		WithStderrLimit(conf.Run.StderrLimit).
		WithStdinFile(inf).
		WithCopyInCached("validator", v.binaryID).
		WithCallback(cb)
}
