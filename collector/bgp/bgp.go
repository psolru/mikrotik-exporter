package bgp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/metrics"
)

var (
	properties         = []string{"name", "remote-as", "state", "prefix-count", "updates-sent", "updates-received", "withdrawn-sent", "withdrawn-received"}
	labelNames         = []string{"name", "address", "session", "asn"}
	metricDescriptions = map[string]*prometheus.Desc{
		"state":              metrics.BuildMetricDescription(prefix, "state", "bgp session state (up = 1)", labelNames),
		"prefix-count":       metrics.BuildMetricDescription(prefix, "prefix_count", "number of prefixes per session", labelNames),
		"updates-sent":       metrics.BuildMetricDescription(prefix, "updates_sent", "number of bgp updates sent per session", labelNames),
		"updates-received":   metrics.BuildMetricDescription(prefix, "updates_received", "number of bgp updates received per session", labelNames),
		"withdrawn-sent":     metrics.BuildMetricDescription(prefix, "withdrawn_sent", "number of bgp withdrawns sent per session", labelNames),
		"withdrawn-received": metrics.BuildMetricDescription(prefix, "withdrawn_received", "number of bgp withdrawns received per session", labelNames),
	}
)

const prefix = "bgp_session"

type bgpCollector struct{}

func NewCollector() *bgpCollector {
	return &bgpCollector{}
}

func (c *bgpCollector) Name() string {
	return prefix
}

func (c *bgpCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range metricDescriptions {
		ch <- d
	}
}

func (c *bgpCollector) Collect(ctx *context.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch bgp metrics: %w", err)
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *bgpCollector) fetch(ctx *context.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.RouterOSClient.Run(
		"/routing/bgp/peer/print",
		"=.proplist="+strings.Join(properties, ","),
	)
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *bgpCollector) collectForStat(re *proto.Sentence, ctx *context.Context) {
	for p := range metricDescriptions {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *bgpCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *context.Context) {
	value := re.Map[property]
	if len(value) == 0 {
		return
	}

	session := re.Map["name"]

	var (
		v   float64
		err error
	)
	switch property {
	case "state":
		if value == "established" {
			v = 1
		}
	default:
		v, err = strconv.ParseFloat(value, 64)
	}
	if err != nil {
		log.WithFields(log.Fields{
			"collector": c.Name(),
			"device":    ctx.DeviceName,
			"session":   session,
			"property":  property,
			"value":     value,
			"error":     err,
		}).Error("failed to parse bgp metric value")
		return
	}

	ctx.MetricsChan <- prometheus.MustNewConstMetric(metricDescriptions[property], prometheus.GaugeValue, v, ctx.DeviceName, ctx.DeviceAddress, session, re.Map["remote-as"])
}
