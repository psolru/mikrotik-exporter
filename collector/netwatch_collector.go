package collector

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type netwatchCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newNetwatchCollector() routerOSCollector {
	c := &netwatchCollector{}
	c.init()
	return c
}

func (c *netwatchCollector) init() {
	c.props = []string{"host", "comment", "status"}
	labelNames := []string{"name", "address", "host", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[2:] {
		c.descriptions[p] = descriptionForPropertyName("netwatch", p, labelNames)
	}
}

func (c *netwatchCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *netwatchCollector) collect(ctx *context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *netwatchCollector) fetch(ctx *context) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/tool/netwatch/print", "?disabled=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching netwatch metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *netwatchCollector) collectForStat(re *proto.Sentence, ctx *context) {
	host := re.Map["host"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, host, comment, re, ctx)
	}
}

func (c *netwatchCollector) collectMetricForProperty(property, host, comment string, re *proto.Sentence, ctx *context) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var v float64
		switch value {
		case "up":
			v = 1
		case "unknown":
			v = 0
		case "down":
			v = -1
		default:
			log.WithFields(log.Fields{
				"device":   ctx.device.Name,
				"host":     host,
				"property": property,
				"value":    value,
				"error":    fmt.Errorf("unexpected netwatch status value"),
			}).Error("error parsing netwatch metric value")
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, host, comment)
	}
}
