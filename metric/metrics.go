package metric


type Metrics interface {
	Inc(name string, labels []string)
	Dec(name string, labels []string)
	Set(name string, value float64, lables []string)
	Add(name string, value float64, labels []string)
	Sub(name string, value float64, labels []string)
	Observe(name string, value float64, labels []string)
}