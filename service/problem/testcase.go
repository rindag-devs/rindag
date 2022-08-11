package problem

// TestGroup is a group of test case.
type TestGroup struct {
	Depends []string `json:"depends"`

	// FullScore is the score of this group.
	//
	// patient's score = FullScore *
	//   min(min_{s \in dependencies} Score(s) / FullScore(s), min_{t \in tests} score(t) / 100)
	FullScore int `json:"full_score"`

	// TimeLimit is the time limit in nanoseconds of this group.
	TimeLimit uint64 `json:"time_limit"`

	// MemoryLimit is the memory limit in bytes of this group.
	MemoryLimit uint64 `json:"memory_limit"`

	// Tests is a list of test cases in the group.
	Tests []TestCase `json:"tests"`
}

// TestCase is a test case.
type TestCase struct {
	// Prefix is prefix of the path of input and answer file of the test case.
	// Input file is <prefix>.in and answer file is <prefix>.ans.
	Prefix string `json:"inf_path"`

	// InfFrom is a string slice to describe where the input file from.
	//
	// - If the input is a fixed input, it will be the path of the input file.
	// - If the input is a generated input, it will be the command to generate the input.
	InfFrom []string `json:"inf_from"`

	// AnsFrom is a string slice to describe where the answer file from.
	//
	// - If the answer is a fixed answer, it will be the path of the answer file.
	// - If the answer will be generated in check part, it will be an empty string.
	AnsFrom []string `json:"ans_from,omitempty"`
}
