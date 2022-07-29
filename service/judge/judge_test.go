package judge

import (
	"context"
	"sync"
	"testing"

	"github.com/criyle/go-judge/pb"
)

// TestEcho is a test for output "Hello, world!" by /bin/echo.
//
// It will do the following:
//
// 1. Get an idle judge.
// 2. Create a request.
// 3. Add the request to the judge.
// 4. Wait for the request to complete.
// 5. Check the request's result.
// 6. If the request's result is success, check the stdout.
// 7. If the request's result is failure, check the error.
func TestEcho(t *testing.T) {
	judge, err := GetIdleJudge()
	if err != nil {
		t.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	helloTask := DefaultTask().
		WithCmd("/bin/bash", "-c", "echo -n Hello, World!").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if err != nil {
				t.Log("failure callback")
				t.Error(err)
				wg.Done()
			}
			if r.Status != pb.Response_Result_Accepted {
				t.Logf("Status: %s", r.Status)
				t.Error(string(r.Files["stderr"]))
				wg.Done()
				return false
			}
			stdoutID := r.FileIDs["stdout"]
			t.Logf("stdoutID: %s", stdoutID)
			stdout, err := judge.GetFile(context.TODO(), stdoutID)
			if err != nil {
				t.Fatal(err)
				return false
			}
			t.Logf("stdout len: %d", len(stdout.Content))
			stdoutStr := string(stdout.Content)
			if stdoutStr != "Hello, World!" {
				t.Errorf("Expected stdout to be \"Hello, World!\", got \"%s\"", stdoutStr)
			}
			wg.Done()
			return true
		})
	judge.AddRequest(NewRequest(context.TODO()).Execute(helloTask))
	wg.Wait()
}

// TestAPlusB is a test for compile and run the "A+B Problem".
//
// It will do the following:
//
// 1. Create a new judge client.
// 2. Create a new request.
// 3. Execute the compile task.
// 4. Check the result of the compile task.
// 5. Execute the run task.
// 6. Check the output of the run task.
func TestAPlusB(t *testing.T) {
	judge, err := GetIdleJudge()
	if err != nil {
		t.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	sol := []byte(
		"#include<stdio.h>\nint main(){int a,b;scanf(\"%d%d\",&a,&b);printf(\"%d\\n\",a+b);}")
	inf := []byte("1 2\n")
	var ouf []byte
	ans := []byte("3\n")
	var solBinID string
	compileTask := DefaultTask().
		WithCmd("/usr/bin/gcc", "sol.c", "-o", "sol").
		WithTimeLimit(10*1000*1000*1000). // Compile time limit 10 s.
		WithCopyIn("sol.c", sol).
		WithCopyOut("sol").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if err != nil {
				t.Log("compile task failure callback")
				t.Error(err)
				wg.Done()
			}
			t.Log("compile success callback")
			if r.Status != pb.Response_Result_Accepted {
				t.Logf("compile failed: %s", r.Status)
				t.Error(string(r.Files["stderr"]))
				wg.Done()
				return false
			}
			if id, ok := r.FileIDs["sol"]; ok {
				solBinID = id
			} else {
				t.Fatal(err)
				return false
			}
			t.Logf("solBinID: %s", solBinID)
			return true
		})
	runTask := DefaultTask().
		WithCmd("sol").
		WithTimeLimit(1*1000*1000*1000). // Run time limit 1 s.
		WithStdin(inf).
		WithCopyInCached("sol", &solBinID).
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if err != nil {
				t.Log("run task failure callback")
				t.Error(err)
				wg.Done()
			}
			t.Logf("solBinID in run task: %s", solBinID)
			if r.Status != pb.Response_Result_Accepted {
				t.Logf("run failed: %s", r.Status)
				t.Error(string(r.Files["stderr"]))
				wg.Done()
				return false
			}
			stdoutID := r.FileIDs["stdout"]
			t.Logf("stdoutID: %s", stdoutID)
			stdout, err := judge.GetFile(context.TODO(), stdoutID)
			if err != nil {
				t.Error(err)
				wg.Done()
				return false
			}
			t.Logf("stdout len: %d", len(stdout.Content))
			ouf = stdout.Content
			if string(ouf) != string(ans) {
				t.Errorf("Expected stdout to be \"%s\", got \"%s\"", string(ans), string(ouf))
			}
			wg.Done()
			return true
		})
	judge.AddRequest(NewRequest(context.TODO()).Execute(compileTask).Then(runTask))
	wg.Wait()
}
