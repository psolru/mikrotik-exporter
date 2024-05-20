package ip_pool

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/psolru/mikrotik-exporter/collector/context"
	"github.com/psolru/mikrotik-exporter/metrics"
)

var metricDescription = metrics.BuildMetricDescription(prefix, "used", "number of used ip/prefixes in pool",
	[]string{"name", "address", "ip_version", "pool"},
)

const prefix = "ip_pool"

type ipPoolCollector struct{}

func NewCollector() *ipPoolCollector {
	return &ipPoolCollector{}
}

func (c *ipPoolCollector) Name() string {
	return prefix
}

func (c *ipPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricDescription
}

func (c *ipPoolCollector) Collect(ctx *context.Context) error {
	return c.collectForIPVersion("4", "ip", ctx)
}

func (c *ipPoolCollector) collectForIPVersion(ipVersion, topic string, ctx *context.Context) error {
	reply, err := ctx.RouterOSClient.Run(
		fmt.Sprintf("/%s/pool/print", topic),
		"=.proplist=name",
	)
	if err != nil {
		return fmt.Errorf("failed to fetch ip pool names: %w", err)
	}

	eg := errgroup.Group{}
	for _, re := range reply.Re {
		m := re.Map
		eg.Go(func() error {
			return c.collectForPool(ipVersion, topic, m["name"], ctx)
		})
	}

	return eg.Wait()
}

func (c *ipPoolCollector) collectForPool(ipVersion, topic, pool string, ctx *context.Context) error {
	reply, err := ctx.RouterOSClient.Run(
		fmt.Sprintf("/%s/pool/used/print", topic),
		fmt.Sprintf("?pool=%s", pool),
		"=count-only=",
	)
	if err != nil {
		return fmt.Errorf("failed to fetch pool used info: %w", err)
	}

	value := reply.Done.Map["ret"]
	if len(value) == 0 {
		return nil
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"collector":  c.Name(),
			"ip_pool":    pool,
			"ip_version": ipVersion,
			"device":     ctx.DeviceName,
			"error":      err,
		}).Error("failed to parse used ip pool metric value")
		return nil
	}

	ctx.MetricsChan <- prometheus.MustNewConstMetric(metricDescription, prometheus.GaugeValue, v,
		ctx.DeviceName, ctx.DeviceAddress, ipVersion, pool,
	)

	return nil
}
