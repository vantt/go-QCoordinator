package queue

import (
	"errors"
	"strconv"
	"time"

	"github.com/beanstalkd/go-beanstalk"
)

// BeanstalkdConnectionPool ...
type BeanstalkdConnectionPool struct {
	address string
	conn    *beanstalk.Conn
	Queues  map[string]*beanstalk.TubeSet
}

const (
	// deadlineSoonDelay defines a period to sleep between receiving
	// DEADLINE_SOON in response to reserve, and re-attempting the reserve.
	DeadlineSoonDelay = 1 * time.Second
)

// NewBeanstalkdConnectionPool ...
func NewBeanstalkdConnectionPool(address string) *BeanstalkdConnectionPool {
	return &BeanstalkdConnectionPool{address: address, Queues: make(map[string]*beanstalk.TubeSet)}
}

// CheckConnection ...
func (bs *BeanstalkdConnectionPool) CheckConnection() bool {
	
	if conn, err := beanstalk.Dial("tcp", bs.address); err == nil && conn != nil {		
		conn.Close()
		return true
	}
   
	return false
}

// Address ... 
func (bs *BeanstalkdConnectionPool) Address() string {
	return bs.address
}

// ListQueues Returns a list of all queue names.
func (bs *BeanstalkdConnectionPool) ListQueues() (queueNames []string, err error) {
	queueNames, err = bs.getGlobalConnect().ListTubes()
	return
}

// CountMessages return total number of ready-items in the queue
func (bs *BeanstalkdConnectionPool) CountMessages(queueName string) (uint64, error) {
	return bs.statsUint64(queueName, "current-jobs-ready")

}

// statsUint64 return an Uint64 stat by queuName and key
func (bs *BeanstalkdConnectionPool) statsUint64(queueName string, key string) (uint64, error) {

	tubeStats := &beanstalk.Tube{
		Conn: bs.getGlobalConnect(),
		Name: queueName,
	}

	statsMap, err := tubeStats.Stats()

	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(statsMap[key], 10, 64)
}

//
// ConsumeMessage reserve-with-timeout until there's a job or something panic-worthy.
// Handles beanstalk.ErrTimeout by retrying immediately.
// Handles beanstalk.ErrDeadline by sleeping DeadlineSoonDelay before retry.
// panics for other errors.
func (bs *BeanstalkdConnectionPool) ConsumeMessage(queueName string, timeout time.Duration) (job *Job, err error) {
	var (
		id   uint64
		body []byte
		ttl  string
	)

	for {
		id, body, err = bs.GetQueue(queueName).Reserve(timeout)

		if err == nil {
			job = &Job{
				ID:          id,
				Payload:     body,
				QueueName:   queueName,
				NumReturns:  0,
				NumTimeOuts: 0,
				TimeLeft:    0,
			}

			job.NumReturns, err = bs.uint64JobStat(queueName, id, "releases")
			job.NumTimeOuts, err = bs.uint64JobStat(queueName, id, "timeouts")
			ttl, err = bs.statJob(queueName, id, "time-left")

			if err == nil {
				job.TimeLeft, err = time.ParseDuration(ttl + "s")
			}

			return
		} else if err.(beanstalk.ConnError).Err == beanstalk.ErrTimeout {
			err = errors.New("Timeout for reserving a job")
			return
		} else if err.(beanstalk.ConnError).Err == beanstalk.ErrDeadline {
			time.Sleep(DeadlineSoonDelay)
			continue
		} else {
			panic(err)
		}
	}
}

// DeleteMessage ...
func (bs *BeanstalkdConnectionPool) DeleteMessage(queueName string, ID uint64) error {
	return bs.GetQueue(queueName).Conn.Delete(ID)
}

// ReturnMessage ...
func (bs *BeanstalkdConnectionPool) ReturnMessage(queueName string, ID uint64, delay time.Duration) error {
	pri, err := bs.uint64JobStat(queueName, ID, "pri")

	if err == nil {
		return bs.GetQueue(queueName).Conn.Release(ID, uint32(pri), delay)
	}

	return err
}

// GiveupMessage ...
func (bs *BeanstalkdConnectionPool) GiveupMessage(queueName string, ID uint64) error {
	pri, err := bs.uint64JobStat(queueName, ID, "pri")

	if err != nil {
		return err
	}

	return bs.GetQueue(queueName).Conn.Bury(ID, uint32(pri))
}

func (bs *BeanstalkdConnectionPool) statJob(queueName string, ID uint64, key string) (string, error) {
	stats, err := bs.GetQueue(queueName).Conn.StatsJob(ID)

	if err != nil {
		return "", err
	}

	return stats[key], nil
}

func (bs *BeanstalkdConnectionPool) uint64JobStat(queueName string, ID uint64, key string) (uint64, error) {
	stat, err := bs.statJob(queueName, ID, key)

	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(stat, 10, 64)
}

// GetQueue ...
func (bs *BeanstalkdConnectionPool) GetQueue(queueName string) *beanstalk.TubeSet {

	if queue, found := bs.Queues[queueName]; found {
		return queue
	}

	conn := bs.connect()

	ts := beanstalk.NewTubeSet(conn, queueName)
	bs.Queues[queueName] = ts

	return ts
}

func (bs *BeanstalkdConnectionPool) getGlobalConnect() *beanstalk.Conn {
	if bs.conn == nil {
		bs.conn = bs.connect()
	}

	return bs.conn
}

func (bs *BeanstalkdConnectionPool) connect() *beanstalk.Conn {
	conn, err := beanstalk.Dial("tcp", bs.address)

	if err != nil {
		panic("Count not connect to beanstalkd server")
	}

	return conn
}
