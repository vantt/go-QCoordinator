package dispatcher

import (
	"sync"
	"time"

	"github.com/vantt/go-QCoordinator/queue"
)

// TaskRequest ...
type TaskRequest struct {
	*queue.Job
}

// TaskResult ...
type TaskResult struct {

	// JobID from beanstalkd.
	ID uint64

	queue string

	StartTime time.Time

	NumReturns uint64

	// Body of the Result
	Body []string

	// Executed is true if the job command was executed (or attempted).
	isExecuted bool

	// TimedOut indicates the worker exceeded TTR for the job.
	// Note this is tracked by a timer, separately to beanstalkd.
	isTimedOut bool

	// Buried is true if the job was buried.
	isFail bool

	// ExitStatus of the command; 0 for success.
	ExitCode  int

	// ExitStatus of the command; 0 for success.
	ExitStatus int

	// Error raised while attempting to handle the job.
	Error error

	ErrorMsg string 

	// Running Time in Seconds
	Runtime float64
}

// NewTaskResult ...
func NewTaskResult(job *queue.Job) *TaskResult {
	return &TaskResult{
		ID: job.ID, 
		queue: job.QueueName, 
		NumReturns: job.NumReturns,
		StartTime: time.Now(),
	}
}

// CalcRuntime ...
func (tr *TaskResult) CalcRuntime() {
	tr.Runtime = time.Since(tr.StartTime).Seconds()	
}

//////////////////////////////////////////////////////////////////////////////////////////////////////


// TaskResultMap ...
type TaskResultMap struct {
	tasks map[uint64]*TaskResult
	sync.RWMutex
}

// NewTaskResultMap ...
func NewTaskResultMap() *TaskResultMap{
	return &TaskResultMap{
		tasks: make(map[uint64]*TaskResult),
	}
}

// CreateTaskResult ...
func (m *TaskResultMap) CreateTaskResult(job *queue.Job) bool {
	_, ok := m.tasks[job.ID]; 

	if !ok {
		m.Lock()
		defer m.Unlock()

		m.tasks[job.ID] = NewTaskResult(job)
	}

	return ok
}

// RemoveTaskResult ...
func (m *TaskResultMap) RemoveTaskResult(ID uint64) bool {
	_, ok := m.tasks[ID]; 

	if ok {
		m.Lock()
		defer m.Unlock()

		delete(m.tasks, ID)
	}

	return ok
}

// GetResult ...
func (m *TaskResultMap) GetResult(ID uint64) *TaskResult {
	if result, ok := m.tasks[ID]; ok {
		return result
	}

	return nil;
}