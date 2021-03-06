package metric

import (		
	"github.com/prometheus/client_golang/prometheus"	
)

type PrometheusMetrics struct {
	nameSpace string
	subsystem string
	mTypes map[string]byte
	metrics map[string]prometheus.Collector

}

func (b *PrometheusMetrics) InitMetrics(nameSpace string, subSystem string, m map[string]byte) {
	b.nameSpace = nameSpace
	b.subsystem = subSystem
	b.mTypes = m
	b.metrics = make(map[string]prometheus.Collector)
}

func(b *PrometheusMetrics) Inc(name string, labels []string)  {

	metric := b.getMetric(name);

	if metric != nil {
		if (b.isCounter(name))  {
			metric.(*prometheus.CounterVec).WithLabelValues(labels...).Inc()
		} else if b.isGauge(name) {
			metric.(*prometheus.GaugeVec).WithLabelValues(labels...).Inc()
		}
	}
}

func(b *PrometheusMetrics) Dec(name string, labels []string) {
	metric := b.getMetric(name);

	if metric != nil {
		if b.isGauge(name) {
			metric.(*prometheus.GaugeVec).WithLabelValues(labels...).Dec()
		}
	}
}

func(b *PrometheusMetrics) Set(name string, value float64, labels []string) {
	metric := b.getMetric(name);

	if metric != nil {
		if b.isGauge(name) {
			metric.(*prometheus.GaugeVec).WithLabelValues(labels...).Set(value)
		}
	}
}

func(b *PrometheusMetrics) Add(name string, value float64, labels []string) {
	metric := b.getMetric(name);

	if metric != nil {
		if (b.isCounter(name))  {
			metric.(*prometheus.CounterVec).WithLabelValues(labels...).Add(value)
		} else if b.isGauge(name) {
			metric.(*prometheus.GaugeVec).WithLabelValues(labels...).Add(value)
		}
	}
}

func(b *PrometheusMetrics) Sub(name string, value float64, labels []string) {
	metric := b.getMetric(name);

	if metric != nil {
		if (b.isGauge(name))  {
			metric.(*prometheus.GaugeVec).WithLabelValues(labels...).Sub(value)
		} 
	}
	
}

func(b *PrometheusMetrics) Observe(name string, value float64, labels []string) {
	metric := b.getMetric(name);

	if metric != nil {
		if (b.isSummary(name))  {
			metric.(*prometheus.SummaryVec).WithLabelValues(labels...).Observe(value)
		} 
	}
}

func (b *PrometheusMetrics) getMetric(name string) prometheus.Collector {
	if metric, ok := b.metrics[name]; ok  {
		return metric
	} else {
		metric := b.createMetric(name)

		if (metric != nil) {
			return metric
		} 

		if !b.isDefined(name) {
			panic("Metric is not defined: " + name)
		}
		
		panic("Undefined error for metric: " + name)
	}
}

func (b *PrometheusMetrics) isDefined(name string) bool {
	_, ok := b.mTypes[name]

	return ok
}

func(b *PrometheusMetrics) createCounter(name string) prometheus.Collector {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: b.nameSpace,
			Subsystem: b.subsystem,
			Name: name,
			Help: name,
		},
		[]string{"queue"},
	)
}

func(b *PrometheusMetrics) createGauge(name string)  prometheus.Collector {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: b.nameSpace,
			Subsystem: b.subsystem,
			Name: name,
			Help: name,
		},
		[]string{"queue"},
	)
}

func(b *PrometheusMetrics) createSummary(name string)  prometheus.Collector {
	return prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: b.nameSpace,
			Subsystem: b.subsystem,
			Name: name,
			Help: name,
		},
		[]string{"queue"},
	)
}

func (b *PrometheusMetrics) createMetric(name string) prometheus.Collector {	
	var metric prometheus.Collector

	if b.isCounter(name) {		
		metric = b.createCounter(name)
	} else if b.isGauge(name) { 		
		metric = b.createGauge(name)
	} else if b.isSummary(name) { 		
		metric = b.createSummary(name)
	}

	if metric != nil {
		prometheus.MustRegister(metric)
		b.metrics[name] = metric
	}

	if metric == nil {
		panic("can not create metric")
	}

	return metric
}

func(b *PrometheusMetrics) isCounter(name string) bool {
	return (b.mTypes[name] == 'c')
}

func(b *PrometheusMetrics) isGauge(name string) bool {
	return (b.mTypes[name] == 'g')
}

func(b *PrometheusMetrics) isSummary(name string) bool {
	return (b.mTypes[name] == 's')
}


