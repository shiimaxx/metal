package types

import "time"

type Metrics struct {
	Data []Metric
}

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
	Tags      map[string]string
}
