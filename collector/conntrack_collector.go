package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type conntrackCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newConntrackCollector() routerOSCollector {
	c := &conntrackCollector{}
	c.init()
	return c
}

func (c *conntrackCollector) init() {
	c.props = []string{"total-entries", "max-entries"}

	labelNames := []string{"name", "address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("conntrack", p, labelNames)
	}
}

func (c *conntrackCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *conntrackCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/firewall/connection/tracking/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching conntrack table metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *conntrackCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *conntrackCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *conntrackCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	desc := c.descriptions[property]
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing conntrack metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address)
}
