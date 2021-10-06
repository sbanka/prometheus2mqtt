package prometheus

import (
	"time"
)

type Metrics map[string]Metric

type Metric struct {
	Time  time.Time
	Value float64
}
