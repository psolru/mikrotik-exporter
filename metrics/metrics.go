package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricDescription - represents prometheus metric description and value type
type MetricDescription struct {
	*prometheus.Desc
	ValueType prometheus.ValueType
}

func BuildMetricDescription(prefix, name, helpText string, labelNames []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName("mikrotik", prefix, name),
		helpText,
		labelNames,
		nil,
	)
}
