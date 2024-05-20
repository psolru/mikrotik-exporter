package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/psolru/mikrotik-exporter/collector/context"
	"github.com/psolru/mikrotik-exporter/dns"
	"github.com/psolru/mikrotik-exporter/routeros"
)

var (
	scrapeDurationMetricDescription = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, scrapePrefix, "duration_seconds"),
		"Duration of a device scrape step",
		[]string{"device", "step", "success"},
		nil,
	)
	collectorDurationMetricDescription = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, scrapePrefix, "collector_duration_seconds"),
		"Duration of a device collector scrape",
		[]string{"device", "collector", "success"},
		nil,
	)

	timeNowUTC = func() time.Time {
		return time.Now().UTC()
	}

	timeSince = func(start time.Time) time.Duration {
		return time.Since(start)
	}
)

type (
	// FeatureCollector - describes the feature collector interface
	FeatureCollector interface {
		Name() string
		Describe(ch chan<- *prometheus.Desc)
		Collect(ctx *context.Context) error
	}

	// Device - represents device configuration for collector
	Device struct {
		// Name - device name
		Name string
		// Address - device address (IP or FQDN), optional
		Address string
		// Port - device port, optional
		Port string
		// Username - device authentication username
		Username string
		// Password - device authentication password
		Password string
		// Client - represents device level routerOS client configuration, optional
		Client Client
		// DNSRecord - represents SRV DNS record for dynamic address lookup, optional
		DNSRecord *Record
		// Collectors - list of enabled collectors for device
		Collectors []FeatureCollector
	}

	// Client - represents routerOS client configuration
	Client struct {
		// DialTimeout - timeout for establishing connection
		DialTimeout time.Duration
		// EnableTLS - enables TLS connection
		EnableTLS bool
		// InsecureTLSSkipVerify - enables TLS connection with skipped server certificate verification
		InsecureTLSSkipVerify bool
	}

	// Record - represents DNS record
	Record struct {
		// Name - represents SRV record name
		Name string
		// ServerAddress - represents DNS server address
		ServerAddress string
	}

	clientCreatorFunc func(*Device) (routeros.Client, error)

	dnsLookupFunc func(name, server string) (string, error)

	// routerosCollector - represents the RouterOS collector instance
	routerosCollector struct {
		clientCreatorFunc clientCreatorFunc
		dnsLookupFunc     dnsLookupFunc
		devices           []*Device
		collectors        []FeatureCollector
	}
)

const (
	namespace    = "mikrotik"
	scrapePrefix = "scrape"

	stepConnect = "connect"
	stepCollect = "collect"

	resultError   = "false"
	resultSuccess = "true"
)

func buildCollectorContext(
	ch chan<- prometheus.Metric,
	device *Device,
	runner routeros.Client,
) *context.Context {
	return &context.Context{
		MetricsChan:    ch,
		RouterOSClient: runner,
		DeviceName:     device.Name,
		DeviceAddress:  device.Address,
	}
}

// WithCustomClientCreatorFunc - sets custom client creator func
func WithCustomClientCreatorFunc(ccf clientCreatorFunc) Option {
	return func(c *routerosCollector) {
		c.clientCreatorFunc = ccf
	}
}

// WithCustomDNSLookupFunc - sets custom DNS lookup func
func WithCustomDNSLookupFunc(dlf dnsLookupFunc) Option {
	return func(c *routerosCollector) {
		c.dnsLookupFunc = dlf
	}
}

// WithCollectors - adds custom feature metrics collectors
func WithCollectors(fc ...FeatureCollector) Option {
	return func(c *routerosCollector) {
		c.collectors = append(c.collectors, fc...)
	}
}

// Option - represents a function on routeros collector instance
type Option func(*routerosCollector)

// NewMikrotikCollector - mikrotik collector instance constructor
func NewMikrotikCollector(devices []*Device, opts ...Option) prometheus.Collector {
	log.WithFields(log.Fields{
		"devices": len(devices),
	}).Info("creating mikrotik collector")

	c := &routerosCollector{
		clientCreatorFunc: createClient,
		dnsLookupFunc:     dns.LookupAddressFromSRVRecord,
		devices:           devices,
		collectors:        make([]FeatureCollector, 0),
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// Describe - implements the prometheus.Collector interface.
func (c *routerosCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationMetricDescription
	ch <- collectorDurationMetricDescription

	for _, co := range c.collectors {
		co.Describe(ch)
	}
}

// Collect - implements the prometheus.Collector interface.
func (c *routerosCollector) Collect(ch chan<- prometheus.Metric) {
	wg := &sync.WaitGroup{}
	for _, d := range c.devices {
		if d.DNSRecord != nil &&
			len(d.DNSRecord.Name) != 0 {
			address, err := c.dnsLookupFunc(d.DNSRecord.Name, d.DNSRecord.ServerAddress)
			if err != nil {
				log.WithFields(log.Fields{
					"device": d.Name,
					"error":  err,
				}).Error("failed to lookup device address")
				continue
			}

			d.Address = address
		}

		wg.Add(1)
		go func(d *Device) {
			defer wg.Done()
			c.collectForDevice(d, ch)
		}(d)
	}

	wg.Wait()
}

func (c *routerosCollector) collectForDevice(d *Device, ch chan<- prometheus.Metric) {
	if err := c.connectAndCollect(d, ch); err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("failed to collect metrics")
	}
}

func (c *routerosCollector) connectAndCollect(d *Device, ch chan<- prometheus.Metric) error {
	startConnect := timeNowUTC()

	cl, err := c.clientCreatorFunc(d)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			scrapeDurationMetricDescription,
			prometheus.GaugeValue,
			timeSince(startConnect).Seconds(),
			d.Name, stepConnect, resultError,
		)
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer cl.Close()

	ch <- prometheus.MustNewConstMetric(
		scrapeDurationMetricDescription,
		prometheus.GaugeValue,
		timeSince(startConnect).Seconds(),
		d.Name, stepConnect, resultSuccess,
	)

	startCollect := timeNowUTC()

	cl.Async()

	// Merge app level collectors and device level collectors
	collectors := append(c.collectors, d.Collectors...) // nolint:gocritic

	ctx := buildCollectorContext(ch, d, cl)
	var wg sync.WaitGroup
	wg.Add(len(collectors))
	for _, co := range collectors {
		go func(co FeatureCollector) {
			defer wg.Done()

			start := timeNowUTC()

			if err = co.Collect(ctx); err != nil {
				log.WithFields(log.Fields{
					"collector": co.Name(),
					"device":    d.Name,
					"error":     err,
				}).Error("failed to collect feature metrics")
				ch <- prometheus.MustNewConstMetric(
					collectorDurationMetricDescription,
					prometheus.GaugeValue,
					timeSince(start).Seconds(),
					d.Name, co.Name(), resultError,
				)
				return
			}

			ch <- prometheus.MustNewConstMetric(
				collectorDurationMetricDescription,
				prometheus.GaugeValue,
				timeSince(start).Seconds(),
				d.Name, co.Name(), resultSuccess,
			)
		}(co)
	}

	wg.Wait()

	ch <- prometheus.MustNewConstMetric(
		scrapeDurationMetricDescription,
		prometheus.GaugeValue,
		timeSince(startCollect).Seconds(),
		d.Name, stepCollect, resultSuccess,
	)

	return nil
}
