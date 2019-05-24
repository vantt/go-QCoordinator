package dispatcher

// TaskResult ...
type TaskResult struct {

	// JobID from beanstalkd.
	ID uint64

	// Executed is true if the job command was executed (or attempted).
	isExecuted bool


	isFail bool

	// ExitStatus of the command; 0 for success.
	ExitCode  int


	ErrorMsg string 

	// Error raised while attempting to handle the job.
	Error error

	// Body of the Result
	ReturnBody []string
}

// NewTaskResult ...
func NewTaskResult(id uint64, exitCode int) *TaskResult {
	return &TaskResult{
		ID: id,	
		ExitCode: exitCode,
	}
}

func (r *TaskResult) IsFail() bool  {
	return (r.ExitCode > 0)
}
