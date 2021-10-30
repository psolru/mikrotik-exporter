package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type poolCollector struct {
	usedCountDesc *prometheus.Desc
}

func (c *poolCollector) init() {
	const prefix = "ip_pool"

	labelNames := []string{"name", "address", "ip_version", "pool"}
	c.usedCountDesc = description(prefix, "pool_used_count", "number of used IP/prefixes in a pool", labelNames)
}

func newPoolCollector() routerOSCollector {
	c := &poolCollector{}
	c.init()
	return c
}

func (c *poolCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.usedCountDesc
}

func (c *poolCollector) collect(ctx *context) error {
	return c.collectForIPVersion("4", "ip", ctx)
}

func (c *poolCollector) collectForIPVersion(ipVersion, topic string, ctx *context) error {
	names, err := c.fetchPoolNames(topic, ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		if poolErr := c.collectForPool(ipVersion, topic, n, ctx); poolErr != nil {
			return poolErr
		}
	}

	return nil
}

func (c *poolCollector) fetchPoolNames(topic string, ctx *context) ([]string, error) {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/pool/print", topic), "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching pool names")
		return nil, err
	}

	names := make([]string, 0)
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *poolCollector) collectForPool(ipVersion, topic, pool string, ctx *context) error {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/pool/used/print", topic), fmt.Sprintf("?pool=%s", pool), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"pool":       pool,
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error fetching pool counts")
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"pool":       pool,
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error parsing pool counts")
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.usedCountDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion, pool)
	return nil
}
