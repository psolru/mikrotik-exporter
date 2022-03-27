package bridge_hosts

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/metrics"
	"github.com/ogi4i/mikrotik-exporter/parsers"
)

var (
	properties        = []string{"bridge", "mac-address", "on-interface", "vid", "dynamic", "local", "external", "age"}
	metricDescription = metrics.BuildMetricDescription(prefix, "age", "bridge host age in seconds",
		[]string{"name", "address", "bridge", "mac_address", "on_interface", "vid", "dynamic", "local", "external"},
	)
)

const prefix = "bridge_host"

type bridgeHostsCollector struct{}

func NewCollector() *bridgeHostsCollector {
	return &bridgeHostsCollector{}
}

func (c *bridgeHostsCollector) Name() string {
	return prefix
}

func (c *bridgeHostsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricDescription
}

func (c *bridgeHostsCollector) Collect(ctx *context.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch bridge hosts metrics: %w", err)
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *bridgeHostsCollector) fetch(ctx *context.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.RouterOSClient.Run(
		"/interface/bridge/host/print",
		"?disabled=false",
		"=.proplist="+strings.Join(properties, ","),
	)
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *bridgeHostsCollector) collectForStat(re *proto.Sentence, ctx *context.Context) {
	value := re.Map["age"]
	if len(value) == 0 {
		return
	}

	v, err := parsers.ParseDuration(value)
	if err != nil {
		log.WithFields(log.Fields{
			"collector": c.Name(),
			"device":    ctx.DeviceName,
			"value":     value,
			"error":     err,
		}).Error("failed to parse bridge host age metric value")
		return
	}

	ctx.MetricsChan <- prometheus.MustNewConstMetric(metricDescription, prometheus.GaugeValue, v,
		ctx.DeviceName, ctx.DeviceAddress,
		re.Map["bridge"], re.Map["mac-address"], re.Map["on-interface"], re.Map["vid"], re.Map["dynamic"],
		re.Map["local"], re.Map["external"],
	)
}
