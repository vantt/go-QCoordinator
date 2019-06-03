package schedule

import (
	
	"github.com/vantt/go-QCoordinator/metric"	
)

type scheduleMetrics struct {
	metric.PrometheusMetrics
}

func NewScheduleMetrics() *scheduleMetrics {	
	m := new(scheduleMetrics)
	
	m.InitMetrics("qCoordinator", "lottery", map[string]byte{
		"ready":'g',
		"tickets":'g',
	})

	return m 
}