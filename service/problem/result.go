package problem

import "github.com/criyle/go-judge/pb"

// RunResult is a result of running a single task.
type RunResult struct {
	Finished bool                          `json:"finished"`
	Err      error                         `json:"err,omitempty"`
	Status   pb.Response_Result_StatusType `json:"status"`
	Time     uint64                        `json:"time"`
	Memory   uint64                        `json:"memory"`
	Stderr   string                        `json:"stderr"`
}

// ParseRunResult parses a run result from a judge response result.
func ParseRunResult(r *pb.Response_Result, err error) *RunResult {
	return &RunResult{
		Finished: err == nil && r.Status == pb.Response_Result_Accepted,
		Err:      err,
		Status:   r.Status,
		Time:     r.Time,
		Memory:   r.Memory,
		Stderr:   TruncateMessage(string(r.Files["stderr"])),
	}
}

// JudgeResult is a result of judging.
type JudgeResult struct {
	Status        pb.Response_Result_StatusType `json:"status"`
	Time          uint64                        `json:"time"`
	Memory        uint64                        `json:"memory"`
	CheckerResult string                        `json:"checker_result"`
	Inf           string                        `json:"inf"`
	Ouf           string                        `json:"ouf"`
	Ans           string                        `json:"ans"`
}
