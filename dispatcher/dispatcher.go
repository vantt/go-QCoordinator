package dispatcher

import (
	"errors"	
	"time"
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"github.com/vantt/go-QCoordinator/config"
	"github.com/vantt/go-QCoordinator/queue"
	"github.com/vantt/go-QCoordinator/stats"
	"github.com/vantt/go-QCoordinator/schedule"
)

const (
	// TimeoutTries is the number of timeouts a job must reach before it is
	// buried. Zero means never execute.
	TimeoutTries = 3

	// ReleaseTries is the number of releases a job must reach before it is
	// buried. Zero means never execute.
	ReturnTries = 3
)

// Dispatcher ...
type Dispatcher struct {
	config     config.BrokerConfig
	logger     *zap.Logger	

	connPool       queue.InterfaceQueueConnectionPool
	scheduler      schedule.InterfaceScheduler
	statAgent      *stats.StatisticAgent

	wgChild sync.WaitGroup

	processingJobs int32
	
	
	aliveChan      chan chan struct{}
	reserveChan    chan struct{}
	taskChan       chan *queue.Job
	resultChan     chan *TaskResult

	reservingTasks *ReservingTaskMap
	metrics *DispatcherMetrics
	
}

// NewDispatcher ...
func NewDispatcher(cfg config.BrokerConfig,logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		config: cfg, 		
		logger: logger,		
		reserveChan:    make(chan struct{}, 1000),
		taskChan:       make(chan *queue.Job, 1000),
		resultChan:     make(chan *TaskResult, 1000),	
		metrics:        NewDispatcherMetrics(),	
		processingJobs: 0,
	}
}

// IsAlive ...
func (tc *Dispatcher) IsAlive() chan struct{} {	
	returnChan := make(chan struct{})

	go func() { tc.aliveChan <- returnChan }()

	return returnChan
}

// Start ...
func (tc *Dispatcher) Start(ctx context.Context, wg *sync.WaitGroup, readyChan chan<- string) error {
	if err := tc.setup(ctx); err != nil {
		return err
	}

	go func() {			
				
		defer func() {
			// wait for all submitted jobs finished
			tc.wgChild.Wait()

			// now close the resultChan
			close(tc.reserveChan)
			close(tc.taskChan)
			close(tc.resultChan)

			// ok I am done
			wg.Done()

			tc.logger.Info("Dispatcher QUIT.")
		}()

		readyChan <- "Dispatcher started."

		for {
			select {

			case <-ctx.Done():	
				// receive cancel signal from the context
				return
		
			case _, ok := <-tc.reserveChan:
				if ok { 					
					tc.wgChild.Add(1)
					go func() {
						defer tc.wgChild.Done()

						if job := tc.reserveJob(); job != nil {
							tc.taskChan <- job
							tc.metrics.Inc("reserved", []string{job.QueueName})
							tc.metrics.Inc("reserving", []string{job.QueueName})
						}
					}()
				}

			case result, ok := <-tc.resultChan:
				if ok { 
					// handle result
					tc.wgChild.Add(1)
					go func() {
						defer tc.wgChild.Done()

						if task := tc.reservingTasks.GetItem(result.ID); task != nil {
							tc.handleJobResult(task, result)				
							tc.metrics.Dec("reserving", []string{task.Queue})			
						}
					}()					
				}
			}
		}
	}()

	return nil
}

func (tc *Dispatcher) setup(ctx context.Context) error {
	var err error

	tc.logger.Info("Dispatcher setting up ....")
	
	tc.reservingTasks = NewReservingTaskMap(ctx, tc.logger)

	if tc.connPool, err = tc.createQueueConnectionPool(); err != nil {
		return err
	}
		
	if tc.statAgent,err = tc.createStatAgent(tc.connPool); err != nil {
		return err
	}

	if tc.scheduler,err = tc.createScheduler(tc.statAgent); err != nil {
		return err
	}

	childReady := make(chan string, 2)
	
	tc.wgChild.Add(1)
	tc.statAgent.Start(ctx, &(tc.wgChild), childReady)
	
	tc.wgChild.Add(1)
	tc.scheduler.Start(ctx, &(tc.wgChild), childReady)
	
	// wait for sub-goroutine to ready
	for i:=0; i < 2; i++  {
		tc.logger.Info(<-childReady)
	}

	close(childReady)

	return nil
}

// Reserve ...
func (tc *Dispatcher) Reserve() <-chan *queue.Job {
	tc.reserveChan <- struct{}{}
	
	return tc.taskChan
}

// Done ...
func (tc *Dispatcher) Done(ts *TaskResult) {	
	tc.resultChan <- ts	
}

func (tc *Dispatcher) handleJobResult(task *ReservingTask, result *TaskResult) (err error) {
	defer func() {
		atomic.AddInt32(&(tc.processingJobs), -1)
	}()

	logger := tc.logger.With(
		zap.String("queue", task.Queue),
		zap.Uint64("job_id", task.ID),
		zap.Int("exitCode", result.ExitCode),
	)

	logger.Info("Handle job result")

	task.CalcRuntime()	
	tc.scheduler.UpdateJobCost(task.Queue, task.Runtime)

	tc.metrics.Observe("latency", task.Runtime, []string{task.Queue})

	// we dont process TIMEOUT task here
	// because the queue will automatically give task to another worker
	if task.IsTimeOut() {
		logger.Info("Task time out, handle nothing")
		return
	}

	switch result.ExitCode {
	case 0:

		err = tc.connPool.DeleteMessage(task.Queue, task.ID)
		tc.metrics.Inc("exec_success", []string{task.Queue})

		if err == nil {			
			tc.metrics.Inc("delete_success", []string{task.Queue})	
			logger.Info("Delete job")			
		} else {		
			tc.metrics.Inc("delete_fail", []string{task.Queue})	
			logger.With(zap.String("error", err.Error())).Error("Deleting job FAIL")			
		}

	default:
		r := task.NumReturns

		if r <= 0 {
			r = 1
		}

		// r*r*r*r means final of 10 tries has 1h49m21s delay, 4h15m33s total.
		// See: http://play.golang.org/p/I15lUWoabI
		delay := time.Duration(r*r*r*r) * time.Second
		tc.metrics.Inc("exec_fail", []string{task.Queue})
		logger.With(zap.String("error", result.ErrorMsg)).Error("Job execution FAIL")

		err = tc.connPool.ReturnMessage(task.Queue, task.ID, delay)

		if err != nil {
			tc.metrics.Inc("return_fail", []string{task.Queue})
			logger.With(zap.String("error", result.ErrorMsg)).Error("Return job Fail")
		} else {
			tc.metrics.Inc("return_success", []string{task.Queue})
			logger.Info("Return job")
		}
	}

	tc.reservingTasks.RemoveItem(task.ID)

	return
}

func (tc *Dispatcher) reserveJob() *queue.Job {
	
	for numTries := 0; numTries < 3; numTries++ {
		job := tc.getNextJob()

		if job != nil {

			// if job was processed too many times, just GiveUp (burry job)
			if job.NumReturns > ReturnTries || job.NumTimeOuts > TimeoutTries {
				err := tc.connPool.GiveupMessage(job.QueueName, job.ID)

				if err != nil {
					tc.logger.With(zap.String("queue", job.QueueName), zap.Uint64("job_id", job.ID)).Error("Give up job Fail")
				} else {
					tc.logger.With(zap.String("queue", job.QueueName), zap.Uint64("job_id", job.ID)).Info("Give up job")
				}
			} else {
				tc.reservingTasks.AddItem(NewReservingTask(job))
				atomic.AddInt32(&(tc.processingJobs), 1)

				tc.logger.With(zap.String("queue", job.QueueName), zap.Uint64("job_id", job.ID)).Info("Process job")

				return job
			}
		}
	}

	return nil
}

func (tc *Dispatcher) getNextJob() *queue.Job {
	if queueName, found := tc.scheduler.GetNextQueue(); found {
		if job, err := tc.connPool.ConsumeMessage(queueName, 3 * time.Second); err == nil && job != nil {
			tc.logger.With(zap.String("queue", job.QueueName), zap.Uint64("job_id", job.ID)).Info("Reserve job")
			return job
		}
	}

	return nil
}


func (tc *Dispatcher) createQueueConnectionPool() (queue.InterfaceQueueConnectionPool, error) {	
	connPool := queue.NewBeanstalkdConnectionPool(tc.config.QueueHost)
	
	if !connPool.CheckConnection() {
		return nil, errors.New("Could not connect to beanstalkd: " + tc.config.QueueHost)
	}

	return connPool, nil
}


func (tc *Dispatcher) createStatAgent(connPool queue.InterfaceQueueConnectionPool) (*stats.StatisticAgent, error) {	
	statAgent := stats.NewStatisticAgent(connPool, tc.logger)

	if statAgent == nil {
		return nil , errors.New("Could not create Statistic Agent")		
	}

	return statAgent, nil
}


func (tc *Dispatcher) createScheduler(statAgent *stats.StatisticAgent) (schedule.InterfaceScheduler, error) {
	if tc.config.Scheduler == "lottery" {
		if scheduler := schedule.NewLotteryScheduler(&(tc.config), statAgent, tc.logger); scheduler != nil {
			return scheduler, nil
		}
	}	
	
	return nil, errors.New("Could not create Scheduler: " + tc.config.Scheduler)
}