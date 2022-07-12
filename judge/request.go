package judge

import (
	"context"

	"github.com/google/uuid"
)

// Request is the request for the Judge.
type Request struct {
	// ctx is the context for the request.
	ctx context.Context

	// ID is the ID of the request.
	ID uuid.UUID

	// Tasks is the list of tasks to be executed.
	// The tasks are executed in parallel.
	Tasks []*Task

	// SubRequest is the sub-request to be executed after the main request.
	SubRequest *Request
}

// NewRequest creates a new request.
func NewRequest(
	ctx context.Context) *Request {
	return &Request{
		ctx:        ctx,
		ID:         uuid.New(),
		Tasks:      make([]*Task, 0),
		SubRequest: nil,
	}
}

// Execute adds the task to be executed.
func (r *Request) Execute(task ...*Task) *Request {
	r.Tasks = append(r.Tasks, task...)
	return r
}

// Then pushes a new request with given tasks to the request chain.
// The new request will be executed after the main request,
// the sub-request of main, the sub-sub-request and so on, until there's no the next sub-request.
func (r *Request) Then(task ...*Task) *Request {
	curReq := r
	for curReq.SubRequest != nil {
		curReq = curReq.SubRequest
	}
	curReq.SubRequest = NewRequest(r.ctx)
	curReq.SubRequest.Execute(task...)
	return r
}
