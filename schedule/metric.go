package schedule

import (
	
	"github.com/vantt/go-QCoordinator/metric"	
)

type ScheduleMetrics struct {
	metric.PrometheusMetrics
}

func NewScheduleMetrics() *ScheduleMetrics {	
	m := new(ScheduleMetrics)

	m.SetMetricTypes(map[string]byte{
		"read":'g',
		"tickets":'g',		
	})

	return m 
}