package dhcp_ipv6

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/metrics"
)

var metricDescription = metrics.BuildMetricDescription(prefix, "binding_count", "number of active bindings per dhcp ipv6 server",
	[]string{"name", "address", "server"},
)

const prefix = "dhcp_ipv6"

type dhcpIPv6Collector struct{}

func NewCollector() *dhcpIPv6Collector {
	return &dhcpIPv6Collector{}
}

func (c *dhcpIPv6Collector) Name() string {
	return prefix
}

func (c *dhcpIPv6Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricDescription
}

func (c *dhcpIPv6Collector) Collect(ctx *context.Context) error {
	reply, err := ctx.RouterOSClient.Run(
		"/ipv6/dhcp-server/print",
		"=.proplist=name",
	)
	if err != nil {
		return fmt.Errorf("failed to fetch dhcp ipv6 server name: %w", err)
	}

	eg := errgroup.Group{}
	for _, re := range reply.Re {
		m := re.Map
		eg.Go(func() error {
			return c.collectForDHCPServer(ctx, m["name"])
		})
	}

	return eg.Wait()
}

func (c *dhcpIPv6Collector) collectForDHCPServer(ctx *context.Context, server string) error {
	reply, err := ctx.RouterOSClient.Run(
		"/ipv6/dhcp-server/binding/print",
		fmt.Sprintf("?server=%s", server),
		"=count-only=",
	)
	if err != nil {
		return fmt.Errorf("failed to fetch dhcp ipv6 server bindings: %w", err)
	}

	value := reply.Done.Map["ret"]
	if len(value) == 0 {
		return nil
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"collector": c.Name(),
			"device":    ctx.DeviceName,
			"value":     value,
			"error":     err,
		}).Error("error parsing float value")
		return nil
	}

	ctx.MetricsChan <- prometheus.MustNewConstMetric(metricDescription, prometheus.GaugeValue, v, ctx.DeviceName, ctx.DeviceAddress, server)

	return nil
}
