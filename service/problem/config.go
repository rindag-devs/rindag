package problem

import (
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Config is a configuration of a problem.
//
// Use YAML as configuration format
// as there is no toml parsing package to support custom struct as map keys.
type Config struct {
	// Statement is a map of language and statement of the problem.
	Statements map[language.Tag]string `yaml:"statements" json:"statements"`

	// Checker is the problem checker.
	//
	// - If Checker is the name of a built-in checker, it will use the built-in checker.
	// - If Checker is an existing path, it will use the Checker from that path.
	// - Otherwise an error will be returned.
	Checker string `yaml:"checker" json:"checker"`

	// Validator is path of problem validator.
	Validator string `yaml:"validator" json:"validator"`

	// Generators is a map of names and paths of generators.
	Generators map[string]string `yaml:"generators" json:"generators"`

	// Solutions is a map of names and paths to problem solutions.
	Solutions map[string]struct {
		// Path is path of solution.
		Path string `yaml:"path" json:"path"`

		// Accepts are the groups which the solution is acceptable to.
		Accepts []string `yaml:"accepts" json:"accepts"`
	} `yaml:"solutions" json:"solutions"`

	// StandardSolution is the name of the main correct solution.
	//
	// If a solution is the standard solution, it should be marked as accepted for all test groups.
	StandardSolution string `yaml:"standard_solution" json:"standard_solution"`

	// fixed_tests is a list of names and paths of fixed test cases.
	//
	// You can call these by name later in the "TestGroups" section.
	FixedTests map[string]struct {
		Inf string `yaml:"inf" json:"inf"`
		Ans string `yaml:"ans" json:"ans"`
	} `yaml:"fixed_tests" json:"fixed_tests"`

	// TestGroups are test groups of the problem.
	TestGroups map[string]TestGroupConfig `yaml:"test_groups" json:"test_groups"`
}

// GetConfig returns a configuration of a problem.
func (p *Problem) GetConfig(rev [20]byte) (*Config, error) {
	confReader, err := p.File(rev, "config.yaml")
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.NewDecoder(confReader).Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

// TestGroupConfig is a config of test group.
type TestGroupConfig struct {
	// Depends is a list of names of test groups that this group depends on.
	Depends []string `yaml:"depends" json:"depends"`

	// FullScore is the score of this group.
	//
	// patient's score = FullScore *
	//   min(min_{s \in dependencies} Score(s) / FullScore(s), min_{t \in tests} score(t) / 100)
	FullScore int32 `yaml:"full_score" json:"full_score"`

	// TimeLimit is the time limit in nanoseconds of this group.
	TimeLimit uint64 `yaml:"time_limit" json:"time_limit"`

	// MemoryLimit is the memory limit in bytes of this group.
	MemoryLimit uint64 `yaml:"memory_limit" json:"memory_limit"`

	// Tests is a list of test cases in the group.
	Tests []TestCaseConfig `yaml:"tests" json:"tests"`
}

// TestCaseConfig is a config of test case.
type TestCaseConfig struct {
	// Fixed is the name of the fixed test case.
	//
	// It is used when the test case is fixed.
	Fixed string `yaml:"fixed,omitempty" json:"fixed,omitempty"`

	// Generator is the generator of the input.
	//
	// It is used when the test case is generated.
	Generator string `yaml:"generator,omitempty" json:"generator,omitempty"`

	// ExtraArgs is the extra arguments of the generator.
	//
	// It is used when the test case is generated.
	ExtraArgs []string `yaml:"extra_args,omitempty" json:"extra_args,omitempty"`

	// IsSample is true if the test case is a sample.
	IsSample bool `yaml:"is_sample" json:"is_sample"`

	// Disable is true if this test is not contained in the test case.
	//
	// For example, you can use a test case with "IsSample" and "NoTest" to describe the rules
	// for interactive problems.
	Disable bool `yaml:"disable" json:"disable"`
}
