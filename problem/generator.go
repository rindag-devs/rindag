package problem

import (
	"bytes"
	"io"
	"rindag/etc"
	"rindag/judge"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

// Generator is a generator to the problem.
type Generator struct {
	source   source
	binaryID *string
}

type problemGeneratorSource struct {
	Problem *Problem
	Rev     int64
}

// NewProblemGenerator creates a generator from a problem.
func NewProblemGenerator(problem *Problem, rev int64) *Generator {
	return &Generator{
		source: problemGeneratorSource{Problem: problem, Rev: rev}, binaryID: new(string)}
}

func (s problemGeneratorSource) ReadCloser() (io.ReadCloser, error) {
	repo, err := s.Problem.Repo()
	if err != nil {
		return nil, err
	}
	return repo.FileContent("generator.cpp", s.Rev)
}

type sourceGeneratorSource struct {
	Source []byte
}

// NewSourceGenerator creates a generator from the source code.
func NewSourceGenerator(source []byte) *Generator {
	return &Generator{source: sourceGeneratorSource{Source: source}, binaryID: new(string)}
}

func (s sourceGeneratorSource) ReadCloser() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.Source)), nil
}

// CompileTask returns the compile task of the generator.
func (g *Generator) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := g.source.ReadCloser()
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
		WithCmd(conf.Generator.Compile.Args...).
		WithCmd("generator.cpp", "-o", "generator").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("generator.cpp", bytes).
		WithCopyIn("testlib.h", TestlibSource).
		WithCopyOut("generator").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; finished {
				ok := false
				if *g.binaryID, ok = r.FileIDs["generator"]; !ok {
					// Impossible to happen.
					log.Fatal("generator compile successful, but binary ID not found")
				}
			}
			return cb(r, err)
		}), nil
}

// GenerateTask returns a judge task to run this generator.
func (g *Generator) GenerateTask(args []string, cb judge.CallbackFunction) *judge.Task {
	conf := &etc.Config.Generator
	generateCmd := append([]string{"generator"}, args...)
	return judge.DefaultTask().
		WithCmd(generateCmd...).
		WithTimeLimit(conf.Run.TimeLimit).
		WithMemoryLimit(conf.Run.MemoryLimit).
		WithStderrLimit(conf.Run.StderrLimit).
		WithCopyInCached("generator", g.binaryID).
		WithCallback(cb)
}
