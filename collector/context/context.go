package context

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/psolru/mikrotik-exporter/routeros"
)

// Context - represents context, which is passed to feature collectors
type Context struct {
	RouterOSClient routeros.Client
	MetricsChan    chan<- prometheus.Metric
	DeviceName     string
	DeviceAddress  string
}
