package judge

import (
	"context"
	"errors"
	"rindag/etc"
	"sync"
	"time"

	"github.com/criyle/go-judge/pb"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Judge is a judge server.
// It is a backend for go-judge.
type Judge struct {
	// execClient is the client for executing programs.
	execClient pb.ExecutorClient

	// requests is the channel for receiving requests.
	requests chan *Request
}

// judges is a collection of judges.
var (
	judges = make(map[string]*Judge)
)

// NewJudge creates a new Judge.
func newJudge(execClient pb.ExecutorClient) *Judge {
	return &Judge{
		execClient: execClient,
		requests:   make(chan *Request, 64),
	}
}

func (j *Judge) processSingleTask(
	parentCtx context.Context, parentCancel context.CancelFunc, task *Task) {
	// Create a new context for the task, and cancel it when the parent context is ended.
	ctx, cancel := context.WithTimeout(
		parentCtx, time.Duration(2*task.TimeLimit)*time.Millisecond+30*time.Second)
	defer cancel()
	pbr := task.ToPbRequest()
	result, err := j.execClient.Exec(ctx, pbr)
	if err != nil || len(result.Results) == 0 {
		// Failed to execute
		select {
		case <-parentCtx.Done():
			// If the parent context is cancelled, do nothing
		default:
			parentCancel()
			log.WithField("task", task.ID).WithError(err).Error("Failed to execute")
			task.Callback(nil, err)
		}
		return
	}
	// Executed successfully
	log.WithField("task", task.ID).Debug("Executed successfully")
	if !task.Callback(result.Results[0], nil) {
		log.WithField("task", task.ID).Info("Aborted")
		parentCancel()
	}
	log.Debug("Finished processing task")
}

// process processes a request.
func (j *Judge) process(req *Request) {
	log.WithField("request", req.ID).Debug("Processing request")
	parentCtx, parentCancel := context.WithCancel(req.ctx)
	defer parentCancel()
	wg := sync.WaitGroup{}
	wg.Add(len(req.Tasks))
	// Process each task in parallel
	for _, task := range req.Tasks {
		go func(task *Task) {
			j.processSingleTask(parentCtx, parentCancel, task)
			wg.Done()
		}(task)
	}
	wg.Wait()
	select {
	case <-parentCtx.Done():
		// If the parent context is cancelled, do nothing
		log.WithField("request", req.ID).Info("Aborted")
		return
	default:
		break // All tasks are finished, do nothing
	}
	// Add the sub-request to process channel
	if req.SubRequest != nil {
		j.requests <- req.SubRequest
	}
	log.WithField("request", req.ID).Debug("Finished processing request")
}

// Start starts the judge.
func (j *Judge) start() {
	go func() {
		for req := range j.requests {
			go j.process(req)
		}
	}()
}

func (j *Judge) AddRequest(req *Request) {
	j.requests <- req
}

func (j *Judge) GetFile(ctx context.Context, fileID string) (*pb.FileContent, error) {
	return j.execClient.FileGet(ctx, &pb.FileID{FileID: fileID})
}

// GetJudge returns a judge by its id.
// If the judge does not exist, returns an error.
func GetJudge(id string) (*Judge, error) {
	j, ok := judges[id]
	if !ok {
		return nil, errors.New("judge not found")
	}
	return j, nil
}

// GetIdleJudge returns a judge that has the least number of tasks.
// If there is no idle judge, returns an error.
func GetIdleJudge() (*Judge, error) {
	var (
		idleJudge *Judge = nil
		idleCount int
	)
	for _, j := range judges {
		if idleJudge == nil || len(j.requests) < idleCount {
			idleJudge = j
			idleCount = len(j.requests)
		}
	}
	if idleJudge == nil {
		return nil, errors.New("no idle judge")
	}
	return idleJudge, nil
}

func init() {
	// Initialize judges from config
	for id, c := range etc.Config.Judges {
		log.WithField("id", id).Debug("Initializing judge")
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
				grpc_prometheus.UnaryClientInterceptor,
				grpc_logrus.UnaryClientInterceptor(log.NewEntry(log.StandardLogger())),
			)),
			grpc.WithStreamInterceptor(
				grpc_middleware.ChainStreamClient(
					grpc_prometheus.StreamClientInterceptor,
					grpc_logrus.StreamClientInterceptor(log.NewEntry(log.StandardLogger())),
				)),
		}
		if c.Token != "" {
			opts = append(opts, grpc.WithPerRPCCredentials(tokenAuth(c.Token)))
		}
		client, err := grpc.Dial(c.Host, opts...)
		if err != nil {
			log.WithError(err).Fatal("Failed to dial")
		}
		execClient := pb.NewExecutorClient(client)
		j := newJudge(execClient)
		j.start()
		judges[id] = j
	}
}

type tokenAuth string

// GetRequestMetadata return value is mapped to request headers.
func (t tokenAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + string(t)}, nil
}

func (tokenAuth) RequireTransportSecurity() bool { return false }
