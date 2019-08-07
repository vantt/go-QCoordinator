package dispatcher

import (
	"context"
	"sync"
	"time"
	"go.uber.org/zap"
	"github.com/vantt/go-QCoordinator/queue"
)

type ReservingTask struct {	
	ID uint64

	Queue string

	StartTime time.Time

	// the moment this task will be timeout
	TimeOut time.Time

	NumReturns uint64

	// Running Time in Seconds
	Runtime float64
}


// NewTaskResult ...
func NewReservingTask(job *queue.Job) *ReservingTask {
	return &ReservingTask{
		ID: job.ID, 
		Queue: job.QueueName, 
		NumReturns: job.NumReturns,
		StartTime: time.Now(),
		TimeOut: time.Now().Add(job.TimeLeft + 1 * time.Second),
	}
}

// CalcRuntime ...
func (t *ReservingTask) CalcRuntime() {
	t.Runtime = time.Since(t.StartTime).Seconds()	
}

// IsTimeOut ...
func (t *ReservingTask) IsTimeOut() bool {
	return time.Now().After(t.TimeOut)
}

////////////////////////////////////////

// TaskResultMap ...
type ReservingTaskMap struct {
	items map[uint64]*ReservingTask
	sync.RWMutex
	logger *zap.Logger
}

// NewTaskResultMap ...
func NewReservingTaskMap(ctx context.Context, logger *zap.Logger) *ReservingTaskMap{
	m := &ReservingTaskMap{
		items: make(map[uint64]*ReservingTask),
		logger: logger,
	}

	m.cleanupTimer(ctx)

	return m
}


func (m *ReservingTaskMap) cleanup() {
	var evictedItems []ReservingTask
	
	for _, item := range m.items {
		if item.IsTimeOut() {
			evictedItems = append(evictedItems, *item)
		}
	}

	m.Lock()
	for _, item := range evictedItems {
		delete(m.items, item.ID)
	}
	m.Unlock()

	for _, item := range evictedItems {
		m.logger.Info("Task TimeOut", zap.String("queue", item.Queue), zap.Uint64("job_id", item.ID))
	}
}


func (m *ReservingTaskMap) cleanupTimer(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-ctx.Done():	
				ticker.Stop()
				return

			case <-ticker.C:
				m.cleanup()
			}
		}
	}()
}

// AddItem ...
func (m *ReservingTaskMap) AddItem(item *ReservingTask) bool {
	var existed bool

	m.Lock()
	
	if _, existed = m.items[item.ID]; !existed {
		m.items[item.ID] = item
	}
	
	m.Unlock()

	return existed
}

// RemoveItem ...
func (m *ReservingTaskMap) RemoveItem(id uint64) bool {
	_, ok := m.items[id]

	if ok {
		m.Lock()
		defer m.Unlock()

		delete(m.items, id)
	}

	return ok
}

// GetItem ...
func (m *ReservingTaskMap) GetItem(ID uint64) *ReservingTask {
	if item, ok := m.items[ID]; ok {
		return item
	}

	return nil;
}