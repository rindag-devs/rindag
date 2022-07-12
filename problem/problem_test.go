package problem

import (
	"testing"

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
