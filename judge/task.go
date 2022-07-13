package judge

import (
	"github.com/criyle/go-judge/pb"
	"github.com/google/uuid"
)

const (
	DefaultTimeLimit   = 5 * 1000 * 1000 * 1000 // 5 second
	DefaultMemoryLimit = 256 * 1024 * 1024      // 256 MB
	DefaultStdoutLimit = 256 * 1024 * 1024      // 256 MB
	DefaultStderrLimit = 10 * 1024              // 10 kB
)

// DefaultEnv is the default environment variables.
var DefaultEnv = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/tmp"}

// CallbackFunction is the callback function when a task is finished.
// If the task is successful, the callback function will be called with the result.
// If the task is failed, the callback function will be called with the error.
// Return true to continue, false to stop.
type CallbackFunction func(*pb.Response_Result, error) bool

// Task is the task to be judged.
// All the fields are required and should not be nil.
type Task struct {
	// ID is the task ID.
	// It is generated automatically.
	ID uuid.UUID

	// Cmd is the command to be executed.
	Cmd []string

	// TimeLimit is the time limit in nanoseconds.
	TimeLimit uint64

	// MemoryLimit is the memory limit in bytes.
	MemoryLimit uint64

	// StdoutLimit is the stdout limit in bytes.
	StdoutLimit int64

	// StderrLimit is the stderr limit in bytes.
	StderrLimit int64

	// Env is the environment variables.
	Env []string

	// Stdin is the input data.
	Stdin *pb.Request_File

	// CopyIn is the files to be copied in.
	CopyIn map[string]*pb.Request_File

	// CopyInCached is the files to be copied in from the cache.
	// If a key is both in CopyInCached and CopyIn, the value in CopyInCached will be used.
	// The value type is a pointer, that means it can be changed after the task had been created.
	// If the value is nil, it will be ignored.
	CopyInCached map[string]*string

	// CopyOut is the files to be copied out.
	CopyOut []string

	// Callback is the callback function when a task is finished.
	Callback CallbackFunction
}

// DefaultTask returns a default (empty) task.
func DefaultTask() *Task {
	return &Task{
		ID:          uuid.New(),
		Cmd:         []string{},
		TimeLimit:   DefaultTimeLimit,
		MemoryLimit: DefaultMemoryLimit,
		StdoutLimit: DefaultStdoutLimit,
		StderrLimit: DefaultStderrLimit,
		Env:         DefaultEnv,
		Stdin: &pb.Request_File{
			File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: []byte{}}}},
		CopyIn:       map[string]*pb.Request_File{},
		CopyInCached: map[string]*string{},
		CopyOut:      []string{},
		Callback: func(*pb.Response_Result, error) bool {
			return true
		},
	}
}

// WithCmd sets the command to be executed.
func (t *Task) WithCmd(cmd ...string) *Task {
	t.Cmd = cmd
	return t
}

// WithTimeLimit sets the time limit in nanoseconds.
func (t *Task) WithTimeLimit(timeLimit uint64) *Task {
	t.TimeLimit = timeLimit
	return t
}

// WithMemoryLimit sets the memory limit in bytes.
func (t *Task) WithMemoryLimit(memoryLimit uint64) *Task {
	t.MemoryLimit = memoryLimit
	return t
}

// WithStderrLimit sets the stderr limit in bytes.
func (t *Task) WithStderrLimit(stderrLimit int64) *Task {
	t.StderrLimit = stderrLimit
	return t
}

// WithEnv adds the environment variables.
func (t *Task) WithEnv(env ...string) *Task {
	t.Env = append(t.Env, env...)
	return t
}

// WithStdin sets the input data.
func (t *Task) WithStdin(stdin []byte) *Task {
	t.Stdin = &pb.Request_File{
		File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: stdin}}}
	return t
}

// WithStdinCached sets the input data to be cached.
func (t *Task) WithStdinCached(fileID string) *Task {
	t.Stdin = &pb.Request_File{
		File: &pb.Request_File_Cached{Cached: &pb.Request_CachedFile{FileID: fileID}}}
	return t
}

// WithStdinFile sets the input data to be a file.
func (t *Task) WithStdinFile(file *pb.Request_File) *Task {
	t.Stdin = file
	return t
}

// WithCopyIn adds the files to be copied in.
func (t *Task) WithCopyIn(path string, data []byte) *Task {
	t.CopyIn[path] = &pb.Request_File{
		File: &pb.Request_File_Memory{Memory: &pb.Request_MemoryFile{Content: data}}}
	return t
}

// WithCopyInCached adds the files to be copied in from the cache.
func (t *Task) WithCopyInCached(path string, fileID *string) *Task {
	t.CopyInCached[path] = fileID
	return t
}

// WithCopyInFile adds the files to be copied in from a file.
func (t *Task) WithCopyInFile(path string, file *pb.Request_File) *Task {
	t.CopyIn[path] = file
	return t
}

// WithCopyOut adds the files to be copied out.
func (t *Task) WithCopyOut(paths ...string) *Task {
	t.CopyOut = append(t.CopyOut, paths...)
	return t
}

// WithCallback sets the callback function when a task is finished.
func (t *Task) WithCallback(callback CallbackFunction) *Task {
	t.Callback = callback
	return t
}

// ToPbRequest converts the task to a protobuf request.
func (t *Task) ToPbRequest() *pb.Request {
	appendedCopyOut := make([]*pb.Request_CmdCopyOutFile, len(t.CopyOut))
	for i, f := range t.CopyOut {
		appendedCopyOut[i] = &pb.Request_CmdCopyOutFile{
			Name: f,
		}
	}
	copyIn := t.CopyIn
	for k, v := range t.CopyInCached {
		if v == nil {
			continue
		}
		copyIn[k] = &pb.Request_File{
			File: &pb.Request_File_Cached{
				Cached: &pb.Request_CachedFile{
					FileID: *v,
				},
			},
		}
	}
	req := &pb.Request{
		Cmd: []*pb.Request_CmdType{
			{
				Args: t.Cmd,
				Env:  t.Env,
				Files: []*pb.Request_File{
					t.Stdin,
					{
						File: &pb.Request_File_Pipe{
							Pipe: &pb.Request_PipeCollector{
								Name: "stdout",
								Max:  t.StdoutLimit,
							},
						},
					},
					{
						File: &pb.Request_File_Pipe{
							Pipe: &pb.Request_PipeCollector{
								Name: "stderr",
								Max:  t.StderrLimit,
							},
						},
					},
				},
				CpuTimeLimit:   t.TimeLimit,
				ClockTimeLimit: t.TimeLimit * 2,
				MemoryLimit:    t.MemoryLimit,
				CopyIn:         copyIn,
				CopyOut:        []*pb.Request_CmdCopyOutFile{{Name: "stderr"}},
				CopyOutCached:  append([]*pb.Request_CmdCopyOutFile{{Name: "stdout"}}, appendedCopyOut...),
			},
		},
	}
	return req
}
