package problem

// TestGroup is a group of test cases.
type TestGroup struct {
	// Depends is a list of names of test groups that this group depends on.
	Depends []string `yaml:"depends" json:"depends"`

	// FullScore is the score of this group.
	//
	// patient's score = FullScore *
	//   min(min_{s \in dependencies} Score(s) / FullScore(s), min_{t \in tests} score(t) / 100)
	FullScore int `yaml:"full_score" json:"full_score"`

	// Tests is a list of test cases in the group.
	Tests []TestCase `yaml:"tests" json:"tests"`
}
