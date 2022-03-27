package firmware

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/metrics"
)

var metricDescription = metrics.BuildMetricDescription(prefix, "package_active", "active firmware packages",
	[]string{"name", "package", "version", "build_time"},
)

const prefix = "firmware"

type firmwareCollector struct{}

func NewCollector() *firmwareCollector {
	return &firmwareCollector{}
}

func (c *firmwareCollector) Name() string {
	return prefix
}

func (c *firmwareCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricDescription
}

func (c *firmwareCollector) Collect(ctx *context.Context) error {
	reply, err := ctx.RouterOSClient.Run("/system/package/getall")
	if err != nil {
		return fmt.Errorf("failed to fetch package: %w", err)
	}

	for _, re := range reply.Re {
		var v float64
		if re.Map["disabled"] == "false" {
			v = 1
		}

		ctx.MetricsChan <- prometheus.MustNewConstMetric(metricDescription, prometheus.GaugeValue, v,
			ctx.DeviceName, re.Map["name"], re.Map["version"], re.Map["build-time"],
		)
	}

	return nil
}
