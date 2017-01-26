package main

import (
	"fmt"
	"time"
)

// Metric collected from a counter
type Metric struct {
	Name      string
	Value     string
	Timestamp int64
}

// NewMetric construct a Metric struct
func NewMetric(name, value string, timestamp int64) Metric {
	return Metric{
		Name:      name,
		Value:     value,
		Timestamp: timestamp,
	}
}

func (metric Metric) String() string {
	return fmt.Sprintf(
		"%s | %s | %s",
		metric.Name,
		metric.Value,
		time.Unix(metric.Timestamp, 0).Format("2006-01-02 15:04:05"),
	)
}
