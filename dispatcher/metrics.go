package dispatcher

import (
	"github.com/rcrowley/go-metrics"
)


func A() {
	s := metrics.NewExpDecaySample(100, 0.015) 
	h := metrics.NewHistogram(s)

	metrics.Register("baz", h)	
}

