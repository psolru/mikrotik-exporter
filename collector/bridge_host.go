package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type bridgeHostCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newBridgeHostCollector() routerOSCollector {
	c := &bridgeHostCollector{}
	c.init()
	return c
}

func (c *bridgeHostCollector) init() {
	c.props = []string{"bridge", "mac-address", "on-interface", "vid", "dynamic", "local", "external", "age"}
	labelNames := []string{"name", "address", "bridge", "mac_address", "on_interface", "vid", "dynamic", "local", "external"}

	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[7:] {
		c.descriptions[p] = descriptionForPropertyName("bridge_host", p, labelNames)
	}
}

func (c *bridgeHostCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *bridgeHostCollector) fetch(ctx *context) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/interface/bridge/host/print", "?disabled=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching birdge host metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *bridgeHostCollector) collect(ctx *context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *bridgeHostCollector) collectForStat(re *proto.Sentence, ctx *context) {
	for _, p := range c.props[7:] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *bridgeHostCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *context) {
	desc := c.descriptions[property]
	value := re.Map[property]
	var v float64
	if value == "" {
		v = 0
	}

	v, err := parseDuration(value)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    value,
			"error":    err,
		}).Error("error parsing bridge host age metric value")
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address,
		re.Map["bridge"], re.Map["mac-address"], re.Map["on-interface"], re.Map["vid"], re.Map["dynamic"],
		re.Map["local"], re.Map["external"])
}
