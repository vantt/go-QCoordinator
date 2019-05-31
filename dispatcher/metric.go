package dispatcher

import (
	
	"github.com/vantt/go-QCoordinator/metric"	
)

type DispatcherMetrics struct {
	metric.PrometheusMetrics
}

func NewDispatcherMetrics() *DispatcherMetrics {	
	m := new(DispatcherMetrics)

	m.SetMetricTypes(map[string]byte{
		"reserved":'c',
		"reserving":'g',
		"exec_success": 'c',
		"exec_fail": 'c',
		"exec_timeout": 'c',
		"delete_success": 'c',
		"delete_fail": 'c',
		"return_success": 'c',
		"return_fail": 'c',
		"latency":'s',
	})

	return m 
}

/*
Dispatcher:
	queue_name_xx: 
		total, num_success, num_fail, num_timeout : counter
		serving: gauge, 
		avg_cost (latency): histogram, 
		
Scheduler:
	queue_name_xx: 
		current_ready: gauge
		tickets: histogram
*/