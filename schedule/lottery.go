package schedule

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"context"
	"go.uber.org/zap"
	"github.com/olekukonko/tablewriter"
	"github.com/vantt/go-QCoordinator/config"
	"github.com/vantt/go-QCoordinator/stats"
)

// Lottery ...
type Lottery struct {
	statAgent    *stats.StatisticAgent
	config       *config.BrokerConfig
	tickets      map[string]int64
	priority     map[string]uint64
	metrics		 *scheduleMetrics
	totalTickets int64
	sync.RWMutex
	logger *zap.Logger
}

// NewLotteryScheduler ...
func NewLotteryScheduler(c *config.BrokerConfig, sa *stats.StatisticAgent, logger *zap.Logger ) *Lottery {
	return &Lottery{
		statAgent:    sa,
		config:       c,
		priority:     make(map[string]uint64),
		metrics:      NewScheduleMetrics(),
		totalTickets: 1,
		logger: logger,
	}
}

// GetNextQueue ..
func (lt *Lottery) GetNextQueue() (queueName string, found bool) {
	var ticket int64

	queueName = ""
	found = false
	
	winner := rand.Int63n(lt.totalTickets)
	counter := int64(0)

	lt.RLock()
	defer lt.RUnlock()

	for queueName, ticket = range lt.tickets {
		counter = counter + ticket

		if counter > winner {
			found = true
			break
		}
	}

	return
}

// Start do re-assign the tickets everytime that Statistic changed
func (lt *Lottery) Start(ctx context.Context, wg *sync.WaitGroup, readyChan chan<- string) {

	chanUpdate := lt.statAgent.GetUpdateChan()

	go func() {
		defer func() {
			wg.Done()
			lt.logger.Info("Lottery Scheduler QUIT")
		}()

		readyChan <- "Lottery Scheduler started"

		for {
			select {
			case stats, ok := <-chanUpdate:

				// closed channel
				if ok == false {
					return	
				}

				lt.assignTickets(stats)

				for queue, tickets := range lt.tickets {
					lt.metrics.Set("tickets", float64(tickets), []string{queue})
				}

			case <-ctx.Done():				
				return
			}
		}
	}()
}

func (lt *Lottery) assignTickets(stats *stats.ServerStatistic) {	
	tmpTickets := make(map[string]int64)
	tmpTotal := int64(0)

	// assign WSJF for every queue
	for _, queueName := range stats.GetQueueNames() {

		if stat, found := stats.GetQueue(queueName); found {
			wsjf := lt.wsjfInt64(stat, lt.getQueuePriority(queueName))

			if wsjf > 0 {				
				tmpTotal += wsjf
				tmpTickets[queueName] = tmpTotal
			}
		}
	}

	// convert WSJF to Percent
	// if isPercent := true; isPercent {
	// 	for k, v := range tmpTickets {
	// 		tmpTickets[k] = (v / tmpTotal) * 100
	// 	}
		
	// 	tmpTotal = 100
	// } 

	lt.Lock()
	defer lt.Unlock()

	lt.totalTickets = tmpTotal
	lt.tickets = tmpTickets

	//dumpStats(stats, tickets)
}

func (lt *Lottery) getQueuePriority(queueName string) uint64 {
	var (
		priority uint64
		found    bool
	)

	if priority, found = lt.priority[queueName]; !found {
		priority = lt.config.GetTopicPriority(queueName)
		lt.priority[queueName] = priority
	}

	return priority
}

// WSJF = Cost of Delay / Job Duration(Size)
// Cost of Delay = NumJobs * Priority
// JobDuration = NumJobs * JobAvgTime
func (lt *Lottery) wsjfFloat(stat *stats.QueueStatistic, priority uint64) float64 {
	NumJobs := float64(stat.GetTotalItems())
	AvgCost := stat.GetAvgJobCost()

	CostOfDelay := NumJobs * float64(priority)
	JobDuration := NumJobs * AvgCost
	
	return CostOfDelay / JobDuration
}

func (lt *Lottery) wsjfInt64(stat *stats.QueueStatistic, priority uint64) int64 {
	NumJobs := float64(stat.GetTotalItems())
	AvgCost := stat.GetAvgJobCost()

	CostOfDelay := NumJobs * float64(priority)
	JobDuration := NumJobs * AvgCost

	lt.metrics.Set("ready", NumJobs, []string{stat.Name})

	return int64(CostOfDelay / JobDuration)
}

// UpdateJobCost ...
func (lt *Lottery) UpdateJobCost(queueName string, jobCost float64) {
	lt.statAgent.UpdateJobCost(queueName, jobCost)
}

func dumpStats(stats *stats.ServerStatistic, tickets map[string]uint64) {

	var ticket uint64
	var found bool

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Queue", "Ready", "Ticket", "AvgCost"})

	for queueName, stat := range stats.Queues {

		if ticket, found = tickets[queueName]; !found {
			ticket = 0
		}

		table.Append([]string{queueName, strconv.FormatUint(stat.GetTotalItems(), 10), strconv.FormatUint(ticket, 10), strconv.FormatFloat(stat.GetAvgJobCost(), 'f', 10, 64)})
	}

	table.Render() // Send output
}
