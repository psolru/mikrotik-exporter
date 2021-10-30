package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type routesCollector struct {
	protocols         []string
	countDesc         *prometheus.Desc
	countProtocolDesc *prometheus.Desc
}

func newRoutesCollector() routerOSCollector {
	c := &routesCollector{}
	c.init()
	return c
}

func (c *routesCollector) init() {
	const prefix = "routes"

	labelNames := []string{"name", "address", "ip_version"}
	c.countDesc = description(prefix, "total_count", "number of routes in RIB", labelNames)
	c.countProtocolDesc = description(prefix, "protocol_count", "number of routes per protocol in RIB", append(labelNames, "protocol"))
	c.protocols = []string{"bgp", "static", "ospf", "dynamic", "connect"}
}

func (c *routesCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.countDesc
	ch <- c.countProtocolDesc
}

func (c *routesCollector) collect(ctx *context) error {
	err := c.collectForIPVersion("4", ctx)
	if err != nil {
		return err
	}

	return c.collectForIPVersion("6", ctx)
}

func (c *routesCollector) collectForIPVersion(ipVersion string, ctx *context) error {
	err := c.collectCount(ipVersion, ctx)
	if err != nil {
		return err
	}

	for _, p := range c.protocols {
		err := c.collectCountByIPVersion(ipVersion, p, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *routesCollector) collectCount(ipVersion string, ctx *context) error {
	reply, err := ctx.client.Run("/ip/route/print", "?disabled=false", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return err
	}

	if len(reply.Done.Map["ret"]) == 0 {
		return nil
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.countDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion)
	return nil
}

func (c *routesCollector) collectCountByIPVersion(ipVersion, protocol string, ctx *context) error {
	reply, err := ctx.client.Run("/ip/route/print", "?disabled=false", fmt.Sprintf("?%s", protocol), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return err
	}

	if len(reply.Done.Map["ret"]) == 0 {
		return nil
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.countProtocolDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion, protocol)
	return nil
}
