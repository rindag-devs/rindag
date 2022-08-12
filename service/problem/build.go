package problem

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ValidatorResultTextLimit = 128

// ParseInfo is a build information of parsing config part.
type ParseInfo struct {
	OK     bool    `json:"ok"`
	Err    string  `json:"error,omitempty"`
	Config *Config `json:"config,omitempty"`
}

// GenerateInfo is a build information of generate part.
type GenerateInfo struct {
	// OK is true if no error occurred.
	OK bool `json:"ok"`

	Err string `json:"error,omitempty"`

	// GeneratorCompileResults is a map of generator name to compile result.
	GeneratorCompileResults map[string]*RunResult `json:"generator_compile_results,omitempty"`

	// StdCompileResult is a compile result of standard solution.
	StdCompileResult *RunResult `json:"std_compile_result,omitempty"`

	// GenerateResults is a map of test case path to generate result.
	GenerateResults map[string]*RunResult `json:"generate_results,omitempty"`

	// StdRunResults is a map of test case path to run result of standard solution.
	StdRunResults map[string]*RunResult `json:"std_run_results,omitempty"`

	// TestGroups are generated test groups.
	TestGroups map[string]*TestGroup `json:"test_groups,omitempty"`
}

// ValidateInfo is a build information of validate part.
type ValidateInfo struct {
	// OK is true if the all the input files of the problem is valid.
	OK bool `json:"ok"`

	Err string `json:"error,omitempty"`

	// ValidatorCompileResult compile result of validator.
	ValidatorCompileResult *RunResult `json:"validator_compile_result,omitempty"`

	// ValidateResults is a map of test case id and the validate result.
	ValidateResults map[string]*RunResult `json:"validate_results,omitempty"`
}

// SolutionTestCasePair is a pair of solution and test case.
type SolutionTestCasePair struct {
	Solution string `json:"solution"`
	TestCase string `json:"test_case"`
}

// CheckInfo is a build information of check part.
type CheckInfo struct {
	// OK is true if all solutions pass and only all test cases it should pass.
	OK bool `json:"ok"`

	Err string `json:"error,omitempty"`

	// SolutionCompileResults is a map of solution name to compile result.
	SolutionCompileResults map[string]*RunResult `json:"solution_compile_results,omitempty"`

	// CheckerCompileResult is a compile result of checker.
	CheckerCompileResult *RunResult `json:"checker_compile_result,omitempty"`

	// JudgeResults is a map of test case id and the judge result.
	JudgeResults map[string]map[string]*JudgeResult `json:"solution_run_results,omitempty"`

	// NotPassGroups are test groups that should pass but actually not.
	NotPassGroups map[string][]string `json:"not_pass_groups,omitempty"`

	// ExtraPassGroups are test groups that should not pass but actually pass.
	ExtraPassGroups map[string][]string `json:"extra_pass_groups,omitempty"`
}

// BuildInfo is a build information of a problem.
type BuildInfo struct {
	OK       bool          `json:"ok"`
	Parse    *ParseInfo    `json:"parse,omitempty"`
	Generate *GenerateInfo `json:"generate,omitempty"`
	Validate *ValidateInfo `json:"validate,omitempty"`
	Check    *CheckInfo    `json:"check,omitempty"`
}

func (b *BuildInfo) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.Errorf("Failed to unmarshal JSONB value:", value)
	}

	result := BuildInfo{}
	err := json.Unmarshal(bytes, &result)
	*b = result
	return err
}

func (b BuildInfo) Value() (driver.Value, error) {
	return json.Marshal(b)
}

// BuildParse is a function to execute the parsing part of build.
func (p *Problem) BuildParse(rev [20]byte) *ParseInfo {
	conf, err := p.GetConfig(rev)
	if err != nil {
		return &ParseInfo{
			OK:  false,
			Err: err.Error(),
		}
	}

	repo, err := p.Repo()
	if err != nil {
		return &ParseInfo{
			OK:  false,
			Err: err.Error(),
		}
	}

	commit, err := repo.CommitObject(rev)
	if err != nil {
		return &ParseInfo{
			OK:  false,
			Err: err.Error(),
		}
	}

	// Ensure checker is valid.
	if _, err := commit.File(conf.Checker); err != nil {
		// Checker is not found in repo.
		// Ensure it is a built-in checker.
		c := BuiltinChecker(conf.Checker)
		if _, err := c.GetSource(); err != nil {
			return &ParseInfo{
				OK:  false,
				Err: fmt.Sprintf("checker '%s' is not found: %s", conf.Checker, err),
			}
		}
	}

	// Ensure validator is valid.
	if _, err := commit.File(conf.Validator); err != nil {
		return &ParseInfo{
			OK:  false,
			Err: fmt.Sprintf("validator '%s' is not found: %s", conf.Validator, err),
		}
	}

	// Ensure generators are valid.
	for name, g := range conf.Generators {
		if _, err := commit.File(g); err != nil {
			return &ParseInfo{
				OK:  false,
				Err: fmt.Sprintf("generator '%s' (path: %s) is not found: %s", name, g, err),
			}
		}
	}

	// Ensure fixed test cases are valid.
	for name, t := range conf.FixedTests {
		if _, err := commit.File(t.Inf); err != nil {
			return &ParseInfo{
				OK:  false,
				Err: fmt.Sprintf("fixed test '%s' (path: %s) is not found: %s", name, t.Inf, err),
			}
		}

		if t.Ans != "" {
			if _, err := commit.File(t.Ans); err != nil {
				return &ParseInfo{
					OK:  false,
					Err: fmt.Sprintf("fixed test '%s' (path: %s) is not found: %s", name, t.Ans, err),
				}
			}
		}
	}

	// Ensure test cases are valid.
	for groupName, g := range conf.TestGroups {
		// Ensure depends are exist.
		for _, d := range g.Depends {
			if _, ok := conf.TestGroups[d]; !ok {
				return &ParseInfo{
					OK:  false,
					Err: fmt.Sprintf("test group '%s' depends on '%s' but it is not found", groupName, d),
				}
			}
		}

		// Ensure full score >= 0.
		if g.FullScore < 0 {
			return &ParseInfo{
				OK:  false,
				Err: fmt.Sprintf("test group '%s' has invalid full score", groupName),
			}
		}

		// Ensure test cases are valid.
		for i, t := range g.Tests {
			// A test case is either a fixed test or a generated test.
			if t.Fixed != "" {
				if _, ok := conf.FixedTests[t.Fixed]; !ok {
					return &ParseInfo{
						OK:  false,
						Err: fmt.Sprintf("test group '%s' has invalid test case '%s'", groupName, t.Fixed),
					}
				}
			} else if t.Generator != "" {
				if _, ok := conf.Generators[t.Generator]; !ok {
					return &ParseInfo{
						OK:  false,
						Err: fmt.Sprintf("test group '%s' has invalid generator '%s'", groupName, t.Generator),
					}
				}
			} else {
				return &ParseInfo{
					OK:  false,
					Err: fmt.Sprintf("test group '%s' has invalid test case '%d'", groupName, i),
				}
			}
		}
	}

	return &ParseInfo{
		OK:     true,
		Config: conf,
	}
}

func getTestCasePathPrefix(group string, idx int) string {
	return fmt.Sprintf("%s-%d", group, idx)
}

// BuildGenerate is a function to execute the generate part of build.
//
// - The first return value is the GeneratingInfo.
// - The second return value is a file system of the test data.
//
// It will do following things:
//
// 1. Compile generators and the standard output.
// 2. For all test data, if its input is fixed, copy it from problem repo;
//    otherwise, use generator to generate it.
// 3. For all test data, if its output is fixed, copy it from problem repo;
//    otherwise, use the standard solution to generate it.
// 4. Create a memory file system with the input data.
func (p *Problem) BuildGenerate(
	rev [20]byte, conf *Config, fs billy.Filesystem,
) *GenerateInfo {
	type compileResponse struct {
		Name   string
		Result *RunResult
	}

	type generateRunResponse struct {
		Path   string
		Result *RunResult
		FileID string
	}

	generators := make(map[string]*Generator)

	generatorCompileTasks := []*judge.Task{}
	generatorCompileResponses := make(chan compileResponse, 16)
	generatorCompileWG := &sync.WaitGroup{}

	for name, path := range conf.Generators {
		g := NewGeneratorFromProblem(p, rev, path)
		generators[name] = g

		// Use closure to pass the generator name.
		// If we don't do this, the for loop will change the value before the function being executed.
		cTask, err := func(name string, path string) (*judge.Task, error) {
			return g.CompileTask(func(r *pb.Response_Result, err error) bool {
				result := ParseRunResult(r, err)
				generatorCompileResponses <- compileResponse{Name: name, Result: result}
				generatorCompileWG.Done()
				if !result.Finished {
					return false
				}
				return true
			})
		}(name, path)
		if err != nil {
			return &GenerateInfo{
				OK:  false,
				Err: fmt.Sprintf("failed to get compile task for generator '%s': %s", name, err),
			}
		}

		generatorCompileWG.Add(1)
		generatorCompileTasks = append(generatorCompileTasks, cTask)
	}

	go func() {
		generatorCompileWG.Wait()
		close(generatorCompileResponses)
	}()

	std := NewSolutionFromProblem(p, rev, conf.Solutions[conf.StandardSolution].Path)
	stdCompileResponses := make(chan *RunResult, 1)
	stdCompileTask, err := std.CompileTask(func(r *pb.Response_Result, err error) bool {
		result := ParseRunResult(r, err)
		stdCompileResponses <- result
		if !result.Finished {
			return false
		}
		return true
	})

	defer close(stdCompileResponses)

	generateTasks := []*judge.Task{}
	generateResponses := make(chan generateRunResponse, 16)
	generateWG := &sync.WaitGroup{}

	stdRunTasks := []*judge.Task{}
	stdRunResponses := make(chan generateRunResponse, 16)
	stdRunWG := &sync.WaitGroup{}

	info := &GenerateInfo{OK: true}
	info.TestGroups = make(map[string]*TestGroup)

	for groupName, group := range conf.TestGroups {
		info.TestGroups[groupName] = &TestGroup{
			Depends:     group.Depends,
			FullScore:   group.FullScore,
			TimeLimit:   group.TimeLimit,
			MemoryLimit: group.MemoryLimit,
			Tests:       []TestCase{},
		}

		for i, test := range group.Tests {
			prefix := getTestCasePathPrefix(groupName, i)
			infPath := prefix + ".in"
			ansPath := prefix + ".ans"
			testCase := TestCase{Prefix: prefix}

			var inf pb.Request_File

			if test.Disable {
				continue
			}

			if test.Fixed != "" {
				// Fixed input.
				memFile, err := fs.Create(infPath)
				if err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to create fixed input file '%s': %s", infPath, err),
					}
				}

				source, err := p.File(rev, conf.FixedTests[test.Fixed].Inf)
				if err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to get source of fixed input '%s': %s", infPath, err),
					}
				}

				infContent, err := ioutil.ReadAll(source)
				if err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to read content of fixed input '%s': %s", infPath, err),
					}
				}

				if _, err := memFile.Write(infContent); err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to copy fixed input '%s': %s", infPath, err),
					}
				}

				testCase.InfFrom = []string{conf.FixedTests[test.Fixed].Inf}

				inf = pb.Request_File{
					File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: infContent}},
				}
			} else if test.Generator != "" {
				// Generated input.
				g := generators[test.Generator]
				generatorArgs := append([]string{"--group", groupName}, test.ExtraArgs...)

				// Use closure to pass the generator name.
				// If we don't do this, the for loop will change the value before the function being executed.
				task := func(infPath string) *judge.Task {
					return g.GenerateTask(generatorArgs,
						func(r *pb.Response_Result, err error) bool {
							result := ParseRunResult(r, err)
							inf = pb.Request_File{File: &pb.Request_File_Cached{
								Cached: &pb.Request_CachedFile{FileID: r.FileIDs["stdout"]},
							}}
							generateResponses <- generateRunResponse{
								Path: infPath, Result: result, FileID: r.FileIDs["stdout"],
							}
							generateWG.Done()
							if !result.Finished {
								return false
							}
							return true
						})
				}(infPath)

				generateWG.Add(1)
				generateTasks = append(generateTasks, task)

				testCase.InfFrom = append(
					[]string{conf.Generators[test.Generator]}, generatorArgs...)
			} else {
				// This branch should not be reached.
				// Because we have already checked the test case in parse part.
				log.Panic("test case must be either fixed or generated")
			}

			if test.Fixed != "" && conf.FixedTests[test.Fixed].Ans != "" {
				// Fixed answer.
				memFile, err := fs.Create(ansPath)
				if err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to create fixed answer '%s': %s", ansPath, err),
					}
				}

				source, err := p.File(rev, conf.FixedTests[test.Fixed].Ans)
				if err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to get source of fixed answer '%s': %s", ansPath, err),
					}
				}

				if _, err := io.Copy(memFile, source); err != nil {
					return &GenerateInfo{
						OK:  false,
						Err: fmt.Sprintf("failed to copy fixed answer '%s': %s", ansPath, err),
					}
				}

				testCase.AnsFrom = []string{conf.FixedTests[test.Fixed].Ans}
			} else {
				// Generated answer.
				task := func(ansPath string) *judge.Task {
					return std.RunTask(
						group.TimeLimit,
						group.MemoryLimit,
						&inf,
						[]string{},
						func(r *pb.Response_Result, err error) bool {
							result := ParseRunResult(r, err)
							stdRunResponses <- generateRunResponse{
								Path: ansPath, Result: result, FileID: r.FileIDs["stdout"],
							}
							stdRunWG.Done()
							if !result.Finished {
								return false
							}
							return true
						})
				}(ansPath)

				stdRunWG.Add(1)
				stdRunTasks = append(stdRunTasks, task)

				testCase.AnsFrom = []string{conf.Solutions[conf.StandardSolution].Path}
			}

			info.TestGroups[groupName].Tests = append(info.TestGroups[groupName].Tests, testCase)
		}
	}

	go func() {
		generateWG.Wait()
		close(generateResponses)
	}()

	go func() {
		stdRunWG.Wait()
		close(stdRunResponses)
	}()

	_, j, err := judge.GetIdleJudge()
	if err != nil {
		return &GenerateInfo{
			OK:  false,
			Err: fmt.Sprintf("failed to get idle judge: %s", err),
		}
	}

	j.AddRequest(judge.NewRequest(context.Background()).
		Execute(generatorCompileTasks...).
		Execute(stdCompileTask).
		Then(generateTasks...).
		Then(stdRunTasks...))

	info.GeneratorCompileResults = make(map[string]*RunResult)

	for resp := range generatorCompileResponses {
		info.GeneratorCompileResults[resp.Name] = resp.Result

		if !resp.Result.Finished {
			info.OK = false
			info.Err = fmt.Sprintf("failed to compile generator '%s': %s", resp.Name, resp.Result.Err)
			break
		}
	}

	if !info.OK {
		return info
	}

	info.StdCompileResult = <-stdCompileResponses
	if !info.StdCompileResult.Finished {
		info.OK = false
		info.Err = fmt.Sprintf("failed to compile standard solution: %s", info.StdCompileResult.Err)
		return info
	}

	info.GenerateResults = make(map[string]*RunResult)

	for resp := range generateResponses {
		info.GenerateResults[resp.Path] = resp.Result

		if !resp.Result.Finished {
			info.OK = false
			info.Err = fmt.Sprintf("failed to generate input file '%s': %s", resp.Path, resp.Result.Err)
			if resp.Result.Err != nil {
				break
			}
			continue
		}

		inf, err := fs.Create(resp.Path)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to create input file '%s': %s", resp.Path, err)
			break
		}

		infContent, err := j.FileGet(context.Background(), resp.FileID)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to get input file '%s': %s", resp.Path, err)
			break
		}

		if _, err := inf.Write(infContent.Content); err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to write input file '%s': %s", resp.Path, err)
			break
		}
	}

	if !info.OK {
		return info
	}

	info.StdRunResults = make(map[string]*RunResult)

	for resp := range stdRunResponses {
		info.StdRunResults[resp.Path] = resp.Result

		if !resp.Result.Finished {
			info.OK = false
			info.Err = fmt.Sprintf("failed to run std on input file '%s': %s", resp.Path, resp.Result.Err)
			if resp.Result.Err != nil {
				break
			}
			continue
		}

		ans, err := fs.Create(resp.Path)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to create answer file '%s': %s", resp.Path, err)
			break
		}

		ansContent, err := j.FileGet(context.Background(), resp.FileID)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to get answer file '%s': %s", resp.Path, err)
			break
		}

		if _, err := ans.Write(ansContent.Content); err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to write answer file '%s': %s", resp.Path, err)
			break
		}
	}

	if !info.OK {
		return info
	}

	return info
}

// BuildValidate is a function to execute the validate part of build.
//
// It will do the following:
//
// 1. Compile the validator.
// 2. Run the validator at input files of all test cases.
// 3. Return the result of the validation.
func (p *Problem) BuildValidate(
	rev [20]byte, conf *Config, testGroups map[string]*TestGroup, fs billy.Filesystem,
) *ValidateInfo {
	type validateResponse struct {
		Path   string
		Result *RunResult
	}

	validator := NewValidatorFromProblem(p, rev, conf.Validator)

	compileResponses := make(chan *RunResult, 1)
	compileTask, err := validator.CompileTask(func(r *pb.Response_Result, err error) bool {
		result := ParseRunResult(r, err)
		compileResponses <- result
		if !result.Finished {
			return false
		}
		return true
	})
	if err != nil {
		return &ValidateInfo{
			OK:  false,
			Err: fmt.Sprintf("failed to get compile task for validator: %s", err),
		}
	}

	defer close(compileResponses)

	validateTasks := []*judge.Task{}
	validateResponses := make(chan validateResponse, 16)
	validateWG := &sync.WaitGroup{}

	for groupName, group := range testGroups {
		for _, test := range group.Tests {
			infPath := test.Prefix + ".in"
			file, err := fs.Open(infPath)
			if err != nil {
				return &ValidateInfo{
					OK:  false,
					Err: fmt.Sprintf("failed to open test case input '%s': %s", infPath, err),
				}
			}

			inputData, err := io.ReadAll(file)
			if err != nil {
				return &ValidateInfo{
					OK:  false,
					Err: fmt.Sprintf("failed to read test case input '%s': %s", infPath, err),
				}
			}

			// Validated test case.
			validatorArgs := []string{"--group", groupName}

			task := func(infPath string) *judge.Task {
				return validator.ValidateTask(
					&pb.Request_File{File: &pb.Request_File_Memory{
						Memory: &pb.Request_MemoryFile{Content: inputData},
					}}, validatorArgs,
					func(r *pb.Response_Result, err error) bool {
						result := ParseRunResult(r, err)
						validateResponses <- validateResponse{
							Path: infPath, Result: result,
						}
						validateWG.Done()
						if !result.Finished {
							return false
						}
						return true
					})
			}(infPath)

			validateWG.Add(1)
			validateTasks = append(validateTasks, task)
		}
	}

	go func() {
		validateWG.Wait()
		close(validateResponses)
	}()

	_, j, err := judge.GetIdleJudge()
	if err != nil {
		return &ValidateInfo{
			OK:  false,
			Err: fmt.Sprintf("failed to get idle judge: %s", err),
		}
	}

	j.AddRequest(judge.NewRequest(context.Background()).Execute(compileTask).Then(validateTasks...))

	info := &ValidateInfo{OK: true}

	info.ValidatorCompileResult = <-compileResponses
	if !info.ValidatorCompileResult.Finished {
		info.OK = false
		info.Err = fmt.Sprintf("failed to compile validator: %s", info.ValidatorCompileResult.Err)
		return info
	}

	info.ValidateResults = make(map[string]*RunResult)

	for resp := range validateResponses {
		info.ValidateResults[resp.Path] = resp.Result

		if !resp.Result.Finished {
			info.OK = false
			info.Err = fmt.Sprintf("failed to run validator on input file '%s': %s", resp.Path, resp.Result.Err)
			if resp.Result.Err != nil {
				break
			}
			continue
		}
	}

	return info
}

// BuildCheck is a function to execute the check part of build.
//
// It will do the following:
//
// 1. Compile the solutions and checker.
// 2. Run the solutions at input files of all test cases, and record these output file ID.
// 3. Run the checker at all test cases, and record the results.
// 4. Check if all the solutions passed the test groups which they should pass.
func (p *Problem) BuildCheck(
	rev [20]byte, conf *Config, testGroups map[string]*TestGroup, fs billy.Filesystem,
) *CheckInfo {
	type compileResponse struct {
		Name   string
		Result *RunResult
	}

	type runResponse struct {
		Solution  string
		TestGroup string
		TestCase  string
		Result    *RunResult
		OufID     string
	}

	type checkResponse struct {
		Solution  string
		TestGroup string
		TestCase  string
		Result    *RunResult
	}

	solutions := make(map[string]*Solution)

	solutionCompileTasks := []*judge.Task{}
	solutionCompileResponses := make(chan compileResponse, 16)
	solutionCompileWG := &sync.WaitGroup{}

	for name, solConf := range conf.Solutions {
		s := NewSolutionFromProblem(p, rev, solConf.Path)
		solutions[name] = s

		cTask, err := func(name string) (*judge.Task, error) {
			return s.CompileTask(func(r *pb.Response_Result, err error) bool {
				result := ParseRunResult(r, err)
				solutionCompileResponses <- compileResponse{Name: name, Result: result}
				solutionCompileWG.Done()
				if !result.Finished {
					return false
				}
				return true
			})
		}(name)
		if err != nil {
			return &CheckInfo{
				OK:  false,
				Err: fmt.Sprintf("failed to get compile task for solution '%s': %s", name, err),
			}
		}

		solutionCompileWG.Add(1)
		solutionCompileTasks = append(solutionCompileTasks, cTask)
	}

	go func() {
		solutionCompileWG.Wait()
		close(solutionCompileResponses)
	}()

	var checker *Checker
	if _, err := p.File(rev, conf.Checker); err != nil {
		// Use built-in checker.
		checker = BuiltinChecker(conf.Checker)
	} else {
		checker = NewCheckerFromProblem(p, rev, conf.Checker)
	}

	checkerCompileResponses := make(chan *RunResult, 1)

	checkerCompileTask, err := checker.CompileTask(func(r *pb.Response_Result, err error) bool {
		result := ParseRunResult(r, err)
		checkerCompileResponses <- result
		if !result.Finished {
			return false
		}
		return true
	})
	if err != nil {
		return &CheckInfo{
			OK:  false,
			Err: fmt.Sprintf("failed to get compile task for checker '%s': %s", conf.Checker, err),
		}
	}

	defer close(checkerCompileResponses)

	runTasks := []*judge.Task{}
	runResponses := make(chan runResponse, 16)
	runWG := &sync.WaitGroup{}

	for groupName, group := range testGroups {
		for _, test := range group.Tests {
			infPath := test.Prefix + ".in"

			memInf, err := fs.Open(infPath)
			if err != nil {
				return &CheckInfo{
					OK:  false,
					Err: fmt.Sprintf("failed to open test case input '%s': %s", infPath, err),
				}
			}

			infContent, err := io.ReadAll(memInf)
			if err != nil {
				return &CheckInfo{
					OK:  false,
					Err: fmt.Sprintf("failed to read test case input '%s': %s", infPath, err),
				}
			}

			inf := &pb.Request_File{File: &pb.Request_File_Memory{
				Memory: &pb.Request_MemoryFile{Content: infContent},
			}}

			for solName := range conf.Solutions {
				solution := solutions[solName]

				runTask := func(solName string, groupName string, test TestCase) *judge.Task {
					return solution.RunTask(
						group.TimeLimit,
						group.MemoryLimit,
						inf,
						[]string{},
						func(r *pb.Response_Result, err error) bool {
							result := ParseRunResult(r, err)
							stdoutID := r.FileIDs["stdout"]
							runResponses <- runResponse{
								Solution:  solName,
								TestGroup: groupName,
								TestCase:  test.Prefix,
								Result:    result,
								OufID:     stdoutID,
							}
							runWG.Done()
							return true
						})
				}(solName, groupName, test)

				runWG.Add(1)
				runTasks = append(runTasks, runTask)
			}
		}
	}

	go func() {
		runWG.Wait()
		close(runResponses)
	}()

	_, j, err := judge.GetIdleJudge()
	if err != nil {
		return &CheckInfo{
			OK:  false,
			Err: fmt.Sprintf("failed to get idle judge: %s", err),
		}
	}

	j.AddRequest(judge.NewRequest(context.Background()).
		Execute(solutionCompileTasks...).
		Execute(checkerCompileTask).
		Then(runTasks...))

	info := &CheckInfo{OK: true}

	info.SolutionCompileResults = make(map[string]*RunResult)

	for resp := range solutionCompileResponses {
		info.SolutionCompileResults[resp.Name] = resp.Result

		if !resp.Result.Finished {
			info.OK = false
			info.Err = fmt.Sprintf("failed to compile solution '%s': %s", resp.Name, err)
			break
		}
	}

	if !info.OK {
		return info
	}

	info.CheckerCompileResult = <-checkerCompileResponses

	if !info.CheckerCompileResult.Finished {
		info.OK = false
		info.Err = fmt.Sprintf("failed to compile checker: %s", err)
		return info
	}

	runResults := make(map[SolutionTestCasePair]*RunResult)
	oufIDs := make(map[SolutionTestCasePair]string)
	info.JudgeResults = make(map[string]map[string]*JudgeResult)

	notPass := make(map[SolutionTestCasePair]bool)

	checkTasks := []*judge.Task{}
	checkResponses := make(chan checkResponse, 16)
	checkWG := &sync.WaitGroup{}

	for sol := range conf.Solutions {
		info.JudgeResults[sol] = make(map[string]*JudgeResult)
	}

	for resp := range runResponses {
		key := SolutionTestCasePair{resp.Solution, resp.TestCase}
		runResults[key] = resp.Result
		oufIDs[key] = resp.OufID

		infPath := resp.TestCase + ".in"
		memInf, err := fs.Open(infPath)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to open test case input '%s': %s", infPath, err)
			break
		}
		infContent, err := io.ReadAll(memInf)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to read test case input '%s': %s", infPath, err)
			break
		}

		oufContent, err := j.FileGet(context.TODO(), resp.OufID)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to get output file for test case '%s': %s", resp.TestCase, err)
			break
		}

		ansPath := resp.TestCase + ".ans"
		memAns, err := fs.Open(ansPath)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to open test case answer '%s': %s", ansPath, err)
			break
		}
		ansContent, err := io.ReadAll(memAns)
		if err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to read test case answer '%s': %s", ansPath, err)
			break
		}

		info.JudgeResults[resp.Solution][resp.TestCase] = &JudgeResult{
			Status: resp.Result.Status,
			Time:   resp.Result.Time,
			Memory: resp.Result.Memory,
			Inf:    TruncateMessage(string(infContent)),
			Ouf:    TruncateMessage(string(oufContent.Content)),
		}

		if resp.Result.Status != pb.Response_Result_Accepted {
			notPass[SolutionTestCasePair{resp.Solution, resp.TestGroup}] = false
			continue
		}

		checkTask := func(resp runResponse) *judge.Task {
			return checker.CheckTask(
				&pb.Request_File{File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: infContent}}},
				&pb.Request_File{File: &pb.Request_File_Cached{Cached: &pb.Request_CachedFile{FileID: resp.OufID}}},
				&pb.Request_File{File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: ansContent}}},
				func(r *pb.Response_Result, err error) bool {
					result := ParseRunResult(r, err)
					checkResponses <- checkResponse{
						Solution:  resp.Solution,
						TestGroup: resp.TestGroup,
						TestCase:  resp.TestCase,
						Result:    result,
					}
					checkWG.Done()
					return true
				},
			)
		}(resp)

		checkTasks = append(checkTasks, checkTask)
		checkWG.Add(1)
	}

	if !info.OK {
		return info
	}

	go func() {
		checkWG.Wait()
		close(checkResponses)
	}()

	j.AddRequest(judge.NewRequest(context.Background()).Execute(checkTasks...))

	for resp := range checkResponses {
		if err := resp.Result.Err; err != nil {
			info.OK = false
			info.Err = fmt.Sprintf("failed to check test case '%s': %s", resp.TestCase, err)
			break
		}
		chkMsg := string(resp.Result.Stderr)
		status, _, msg := ParseTestlibOutput(chkMsg, 100)
		info.JudgeResults[resp.Solution][resp.TestCase].Status = status
		info.JudgeResults[resp.Solution][resp.TestCase].CheckerResult = msg

		if status != pb.Response_Result_Accepted {
			notPass[SolutionTestCasePair{resp.Solution, resp.TestGroup}] = true
		}
	}

	if !info.OK {
		return info
	}

	shouldPass := make(map[SolutionTestCasePair]bool)

	info.NotPassGroups = make(map[string][]string)

	for solName, sol := range conf.Solutions {
		for _, t := range sol.Accepts {
			shouldPass[SolutionTestCasePair{solName, t}] = true
			if notPass[SolutionTestCasePair{solName, t}] {
				info.OK = false
				info.Err = fmt.Sprintf("test case '%s' of solution '%s' should be accepted but not", t, solName)
				info.NotPassGroups[solName] = append(info.NotPassGroups[solName], t)
			}
		}
	}

	info.ExtraPassGroups = make(map[string][]string)

	for stp := range notPass {
		sol := stp.Solution
		test := stp.TestCase

		if shouldPass[stp] {
			info.OK = false
			info.Err = fmt.Sprintf("test case '%s' of solution '%s' should not be accepted but it is", test, sol)
			info.ExtraPassGroups[sol] = append(info.NotPassGroups[sol], test)
		}
	}

	return info
}

// Build builds problem.
func (p *Problem) Build(rev [20]byte) (*BuildInfo, billy.Filesystem) {
	result := &BuildInfo{
		OK:       false,
		Parse:    nil,
		Generate: nil,
		Validate: nil,
		Check:    nil,
	}

	result.Parse = p.BuildParse(rev)
	log.Debugf("build parse: %v", result.Parse)
	if !result.Parse.OK {
		return result, nil
	}

	fs := memfs.New()
	result.Generate = p.BuildGenerate(rev, result.Parse.Config, fs)
	log.Debugf("build generate: %v", result.Generate)
	if !result.Generate.OK {
		return result, nil
	}

	result.Validate = p.BuildValidate(rev, result.Parse.Config, result.Generate.TestGroups, fs)
	log.Debugf("build validate: %v", result.Validate)
	if !result.Validate.OK {
		return result, nil
	}

	result.Check = p.BuildCheck(rev, result.Parse.Config, result.Generate.TestGroups, fs)
	log.Debugf("build check: %v", result.Check)
	if !result.Check.OK {
		return result, nil
	}

	result.OK = true
	return result, fs
}
