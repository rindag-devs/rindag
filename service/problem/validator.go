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
	source   source
	binaryID *string
}

type problemValidatorSource struct {
	Problem *Problem
	Rev     [20]byte
}

// NewProblemValidator creates a validator from a problem.
func NewProblemValidator(problem *Problem, rev [20]byte) *Validator {
	return &Validator{
		source: problemValidatorSource{Problem: problem, Rev: rev}, binaryID: new(string),
	}
}

func (s problemValidatorSource) ReadCloser() (io.ReadCloser, error) {
	return s.Problem.File("validator.cpp", s.Rev)
}

type sourceValidatorSource struct {
	Source []byte
}

// NewSourceValidator creates a validator from the source code.
func NewSourceValidator(source []byte) *Validator {
	return &Validator{source: sourceValidatorSource{Source: source}, binaryID: new(string)}
}

func (s sourceValidatorSource) ReadCloser() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.Source)), nil
}

// CompileTask returns the compile task of the validator.
func (v *Validator) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := v.source.ReadCloser()
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
func (v *Validator) ValidateTask(inf *pb.Request_File, cb judge.CallbackFunction) *judge.Task {
	conf := &etc.Config.Validator
	return judge.DefaultTask().
		WithCmd("validator").
		WithTimeLimit(conf.Run.TimeLimit).
		WithMemoryLimit(conf.Run.MemoryLimit).
		WithStderrLimit(conf.Run.StderrLimit).
		WithStdinFile(inf).
		WithCopyInCached("validator", v.binaryID).
		WithCallback(cb)
}
