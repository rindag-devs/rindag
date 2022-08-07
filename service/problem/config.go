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

	// FixedTests is a list of names and paths to problem fixed tests.
	//
	// You can call these names later in the "TestGroups" section.
	FixedTests map[string]string `yaml:"fixed_tests" json:"fixed_tests"`

	// TestGroups are test groups of the problem.
	TestGroups map[string]TestGroup `yaml:"test_groups" json:"test_groups"`
}

// GetConfig returns a configuration of a problem.
func (p *Problem) GetConfig(rev [20]byte) (*Config, error) {
	confReader, err := p.File("config.yaml", rev)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.NewDecoder(confReader).Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
