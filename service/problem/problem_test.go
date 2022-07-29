package problem

import (
	"context"
	"testing"

	"rindag/service/etc"
	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
)

func TestParseTestlibOutputAC(t *testing.T) {
	output := "ok your answer is right"
	expectedOutput := "AC your answer is right"
	status, score, msg := ParseTestlibOutput(output, 123)
	if status != pb.Response_Result_Accepted {
		t.Errorf("status should be Accepted, but %v", status)
	}
	if score != 123 {
		t.Errorf("score should be 123, but %v", score)
	}
	if msg != expectedOutput {
		t.Errorf("message should be '%v', but '%v'", expectedOutput, msg)
	}
}

func TestParseTestlibOutputWA(t *testing.T) {
	output := "wrong answer your answer is wrong!"
	expectedOutput := "WA your answer is wrong!"
	status, score, msg := ParseTestlibOutput(output, 123)
	if status != pb.Response_Result_WrongAnswer {
		t.Errorf("status should be WrongAnswer, but %v", status)
	}
	if score != 0 {
		t.Errorf("score should be 0, but %v", score)
	}
	if msg != expectedOutput {
		t.Errorf("message should be '%v', but '%v'", expectedOutput, msg)
	}
}

func TestParseTestlibOutputPC(t *testing.T) {
	output := "partially correct (0.12) ABCDE"
	expectedOutput := "PC 0.12 ABCDE"
	status, score, msg := ParseTestlibOutput(output, 1000)
	if status != pb.Response_Result_PartiallyCorrect {
		t.Errorf("status should be PartiallyCorrect, but %v", status)
	}
	if score != 120 {
		t.Errorf("score should be 120, but %v", score)
	}
	if msg != expectedOutput {
		t.Errorf("message should be '%v', but '%v'", expectedOutput, msg)
	}
}

func TestParseTestlibOutputPoints(t *testing.T) {
	output := "points 0.12 you got points"
	expectedOutput := "PC 0.12 you got points"
	status, score, msg := ParseTestlibOutput(output, 1000)
	if status != pb.Response_Result_PartiallyCorrect {
		t.Errorf("status should be PartiallyCorrect, but %v", status)
	}
	if score != 120 {
		t.Errorf("score should be 120, but %v", score)
	}
	if msg != expectedOutput {
		t.Errorf("message should be '%v', but '%v'", expectedOutput, msg)
	}
}

func TestParseTestlibOutputPE(t *testing.T) {
	output := "wrong output format expected a space"
	expectedOutput := "PE expected a space"
	status, score, msg := ParseTestlibOutput(output, 123)
	if status != pb.Response_Result_WrongAnswer {
		t.Errorf("status should be WrongAnswer, but %v", status)
	}
	if score != 0 {
		t.Errorf("score should be 0, but %v", score)
	}
	if msg != expectedOutput {
		t.Errorf("message should be '%v', but '%v'", expectedOutput, msg)
	}
}

// TestJudgeAPlusB is used to test a complete problem "A+B Problem".
//
// This test will test the compilation and running of the generator, checker, validator,
// and patient's code.
//
// It will do the following:
//
// 1. Compile the generator, the checker, the validator, and the patient's code.
// 2. Run the generator, get the cache file of generated data and save it to variable "inf".
// 3. Run the validator to verify that the generated data is legitimate.
// 4. Run the patient's code. Save patient's output to variable "ouf".
// 5. Run the checker to verify that the patient's output is correct.
func TestJudgeAPlusB(t *testing.T) {
	solutionSource := []byte(`
		#include<iostream>
		int main(){int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;return 0;}
	`)
	checkerSource := []byte(`
		#include "testlib.h"
		int main(int argc, char **argv) {
			registerTestlibCmd(argc, argv);
			int a = inf.readInt(0, 100), b = inf.readInt(0, 100);
			int pans = ouf.readInt(0, 200), jans = a + b;
			if (pans != jans) quitf(_wa, "expected %d, found %d", jans, pans);
			quitf(_ok, "the answer of %d + %d is %d", a, b, jans);
			return 0;
		}
	`)
	validatorSource := []byte(`
		#include "testlib.h"
		int main(int argc, char **argv) {
			registerValidation(argc, argv);
			inf.readInt(0, 100), inf.readSpace(), inf.readInt(0, 100), inf.readEoln(), inf.readEof();
			return 0;
		}
	`)
	generatorSource := []byte(`
		#include <iostream>
		#include "testlib.h"
		int main(int argc, char **argv) {
			registerGen(argc, argv, 1);
			std::cout << rnd.next(0, 100) << ' ' << rnd.next(0, 100) << std::endl;
			return 0;
		}
	`)

	generator := NewSourceGenerator(generatorSource)
	checker := NewSourceChecker(checkerSource)
	validator := NewSourceValidator(validatorSource)

	result := make(chan bool)
	defer close(result)

	var solutionBinaryID string
	var inf pb.Request_File
	var ouf pb.Request_File
	ans := pb.Request_File{File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{
		Content: []byte("the checker doesn't need an answer file"),
	}}}

	checkerCompileTask, err := checker.CompileTask(func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("checker compile task not finished")
			t.Log(r.Status)
			t.Log(string(r.Files["stderr"]))
			result <- false
			return false
		}
		t.Log("checker compile task finished")
		return true
	})
	if err != nil {
		t.Errorf("checker compile task error: %v", err)
		result <- false
		return
	}

	checkTask := checker.CheckTask(&inf, &ouf, &ans, func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("check task not finished")
			t.Log(r.Status)
			t.Log(string(r.Files["stderr"]))
			result <- false
			return false
		}
		t.Log("Accepted")
		result <- true
		return true
	})

	generatorCompileTask, err := generator.CompileTask(func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("generator compile task not finished")
			t.Log(err)
			t.Log(r.Status)
			t.Log(string(r.Files["stderr"]))
			result <- false
			return false
		}
		t.Log("generator compile task finished")
		return true
	})
	if err != nil {
		t.Errorf("generator compile task error: %v", err)
		result <- false
		return
	}

	generateTask := generator.GenerateTask([]string{}, func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("generate task not finished")
			t.Log(r.Status)
			t.Log(r.Error)
			result <- false
			return false
		}
		t.Logf("inf: %s", r.FileIDs["stdout"])
		inf = pb.Request_File{File: &pb.Request_File_Cached{Cached: &pb.Request_CachedFile{
			FileID: r.FileIDs["stdout"],
		}}}
		return true
	})

	validatorCompileTask, err := validator.CompileTask(func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("validator compile task not finished")
			t.Log(r.Status)
			t.Log(r.Error)
			result <- false
			return false
		}
		t.Log("validator compile task finished")
		return true
	})
	if err != nil {
		t.Errorf("validator compile task error: %v", err)
		result <- false
		return
	}

	validateTask := validator.ValidateTask(&inf, func(r *pb.Response_Result, err error) bool {
		if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
			t.Error("validate task not finished")
			t.Log(r.Status)
			t.Log(string(r.Files["stderr"]))
			result <- false
			return false
		}
		return true
	})

	conf := etc.Config
	solutionCompileTask := judge.DefaultTask().
		WithCmd(conf.Compile.Cmd...).WithCmd("sol.cpp", "-o", "sol").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("sol.cpp", solutionSource).
		WithCopyOut("sol").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
				t.Error("solution compile task not finished")
				t.Log(r.Status)
				t.Log(string(r.Files["stderr"]))
				result <- false
				return false
			}
			solutionBinaryID = r.FileIDs["sol"]
			t.Log("solution compile task finished")
			return true
		})

	solutionRunTask := judge.DefaultTask().
		WithCmd("sol").
		WithTimeLimit(1*1000*1000*1000).
		WithMemoryLimit(64*1024*1024).
		WithStdinFile(&inf).
		WithCopyInCached("sol", &solutionBinaryID).
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; !finished {
				t.Error("solution run task not finished")
				result <- false
				return false
			}
			ouf = pb.Request_File{File: &pb.Request_File_Cached{Cached: &pb.Request_CachedFile{
				FileID: r.FileIDs["stdout"],
			}}}
			return true
		})

	j, err := judge.GetIdleJudge()
	if err != nil {
		t.Fatal(err)
	}

	j.AddRequest(judge.NewRequest(context.TODO()).
		Execute(
			checkerCompileTask,
			validatorCompileTask,
			generatorCompileTask,
			validatorCompileTask,
			solutionCompileTask).
		Then(generateTask).
		Then(validateTask).
		Then(solutionRunTask).
		Then(checkTask))

	status := <-result
	t.Log(status)
}
