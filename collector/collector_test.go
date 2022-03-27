package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/collector/mocks"
	"github.com/ogi4i/mikrotik-exporter/routeros"
	routerosMocks "github.com/ogi4i/mikrotik-exporter/routeros/mocks"
)

func TestNewMikrotikCollector(t *testing.T) {
	r := require.New(t)
	t.Parallel()

	validDevices := []*Device{
		{
			Name:     "test1",
			Address:  "192.168.1.1",
			Port:     "",
			Username: "test1-user",
			Password: "test1-pass",
			Client: Client{
				DialTimeout:           time.Second,
				EnableTLS:             true,
				InsecureTLSSkipVerify: true,
			},
		},
		{
			Name:     "test2",
			Address:  "192.168.3.1",
			Port:     "",
			Username: "test2-user",
			Password: "test2-pass",
			Client: Client{
				DialTimeout: 2 * time.Second,
			},
		},
	}

	t.Run("no options", func(t *testing.T) {
		got := NewMikrotikCollector(validDevices)
		v, ok := got.(*routerosCollector)
		r.True(ok)
		r.ElementsMatch(validDevices, v.devices)
		r.Empty(v.collectors)
	})

	t.Run("with options", func(t *testing.T) {
		got := NewMikrotikCollector(validDevices, WithCollectors(mocks.NewFeatureCollectorMock(t)))
		v, ok := got.(*routerosCollector)
		r.True(ok)
		r.ElementsMatch(validDevices, v.devices)
		r.ElementsMatch([]FeatureCollector{
			mocks.NewFeatureCollectorMock(t),
		}, v.collectors)
	})
}

func Test_collector_Describe(t *testing.T) {
	r := require.New(t)

	validDevices := []*Device{
		{
			Name: "test1",
		},
		{
			Name: "test2",
		},
	}

	co := NewMikrotikCollector(validDevices)

	describeChan := make(chan *prometheus.Desc)
	doneChan := make(chan struct{})
	var gotDescriptions []*prometheus.Desc
	go func() {
		defer close(doneChan)
		for desc := range describeChan {
			gotDescriptions = append(gotDescriptions, desc)
		}
	}()

	co.Describe(describeChan)
	close(describeChan)
	<-doneChan
	r.ElementsMatch([]*prometheus.Desc{
		scrapeDurationMetricDescription,
		scrapeSuccessMetricDescription,
	}, gotDescriptions)
}

func Test_collector_Collect(t *testing.T) {
	r := require.New(t)

	validDevices := []*Device{
		{
			Name:    "test1",
			Address: "192.168.1.1",
		},
		{
			Name:    "test2",
			Address: "192.168.3.1",
			DNSRecord: &Record{
				Name:          "test2.fqdn.com",
				ServerAddress: "1.1.1.1",
			},
		},
	}
	timeSince = func(start time.Time) time.Duration {
		return 2 * time.Second
	}

	mc := minimock.NewController(t)
	routerOSClientMock := routerosMocks.NewRouterOSClientMock(mc)
	featureCollectorMock := mocks.NewFeatureCollectorMock(mc)
	featureCollectorMock.NameMock.Return("testCollector")
	featureCollectorMock.CollectMock.Set(func(ctx *context.Context) error {
		switch ctx.DeviceAddress {
		case "192.168.1.1":
			r.Equal("test1", ctx.DeviceName)
		case "192.168.3.1":
			r.Equal("test2", ctx.DeviceName)
		case "192.168.5.1":
			r.Equal("test3", ctx.DeviceName)
			return errors.New("some collector error")
		default:
			r.FailNow("unexpected device address")
		}

		return nil
	})

	resetMocks := func() {
		mc = minimock.NewController(t)
		routerOSClientMock = routerosMocks.NewRouterOSClientMock(mc)
	}

	testCases := []struct {
		name     string
		devices  []*Device
		opts     []Option
		setMocks func()
		want     []prometheus.Metric
	}{
		{
			name:    "default collectors",
			devices: validDevices,
			opts: []Option{
				WithCustomClientCreatorFunc(func(device *Device) (routeros.Client, error) {
					return routerOSClientMock, nil
				}),
				WithCustomDNSLookupFunc(func(name, server string) (string, error) {
					return "192.168.3.1", nil
				}),
			},
			setMocks: func() {
				routerOSClientMock.AsyncMock.Set(func() <-chan error {
					ch := make(chan error)
					defer close(ch)
					return ch
				})
				routerOSClientMock.CloseMock.Return()
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test1"),
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test2"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test1"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test2"),
			},
		},
		{
			name:    "success with custom collector",
			devices: validDevices,
			opts: []Option{
				WithCustomClientCreatorFunc(func(device *Device) (routeros.Client, error) {
					return routerOSClientMock, nil
				}),
				WithCustomDNSLookupFunc(func(name, server string) (string, error) {
					return "192.168.3.1", nil
				}),
				WithCollectors(featureCollectorMock),
			},
			setMocks: func() {
				routerOSClientMock.AsyncMock.Set(func() <-chan error {
					ch := make(chan error)
					defer close(ch)
					return ch
				})
				routerOSClientMock.CloseMock.Return()
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test1"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test1"),
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test2"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test2"),
				prometheus.MustNewConstMetric(collectorSuccessMetricDescription, prometheus.GaugeValue, 1, "test1", "testCollector"),
				prometheus.MustNewConstMetric(collectorDurationMetricDescription, prometheus.GaugeValue, 2, "test1", "testCollector"),
				prometheus.MustNewConstMetric(collectorSuccessMetricDescription, prometheus.GaugeValue, 1, "test2", "testCollector"),
				prometheus.MustNewConstMetric(collectorDurationMetricDescription, prometheus.GaugeValue, 2, "test2", "testCollector"),
			},
		},
		{
			name: "error with custom collector",
			devices: []*Device{
				{
					Name:    "test1",
					Address: "192.168.1.1",
				},
				{
					Name:    "test3",
					Address: "192.168.5.1",
					DNSRecord: &Record{
						Name:          "test3.fqdn.com",
						ServerAddress: "1.1.1.1",
					},
				},
			},
			opts: []Option{
				WithCustomClientCreatorFunc(func(device *Device) (routeros.Client, error) {
					return routerOSClientMock, nil
				}),
				WithCustomDNSLookupFunc(func(name, server string) (string, error) {
					return "192.168.5.1", nil
				}),
				WithCollectors(featureCollectorMock),
			},
			setMocks: func() {
				routerOSClientMock.AsyncMock.Set(func() <-chan error {
					ch := make(chan error)
					defer close(ch)
					return ch
				})
				routerOSClientMock.CloseMock.Return()
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test1"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test1"),
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test3"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test3"),
				prometheus.MustNewConstMetric(collectorSuccessMetricDescription, prometheus.GaugeValue, 1, "test1", "testCollector"),
				prometheus.MustNewConstMetric(collectorDurationMetricDescription, prometheus.GaugeValue, 2, "test1", "testCollector"),
				prometheus.MustNewConstMetric(collectorSuccessMetricDescription, prometheus.GaugeValue, 0, "test3", "testCollector"),
				prometheus.MustNewConstMetric(collectorDurationMetricDescription, prometheus.GaugeValue, 2, "test3", "testCollector"),
			},
		},
		{
			name:    "skipped device with DNS lookup",
			devices: validDevices,
			opts: []Option{
				WithCustomClientCreatorFunc(func(device *Device) (routeros.Client, error) {
					return routerOSClientMock, nil
				}),
				WithCustomDNSLookupFunc(func(name, server string) (string, error) {
					return "", errors.New("some dns lookup error")
				}),
			},
			setMocks: func() {
				routerOSClientMock.AsyncMock.Set(func() <-chan error {
					ch := make(chan error)
					defer close(ch)
					return ch
				})
				routerOSClientMock.CloseMock.Return()
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 1, "test1"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test1"),
			},
		},
		{
			name:    "failed to connect to device",
			devices: validDevices,
			opts: []Option{
				WithCustomClientCreatorFunc(func(device *Device) (routeros.Client, error) {
					return nil, errors.New("some connection error")
				}),
				WithCustomDNSLookupFunc(func(name, server string) (string, error) {
					return "", errors.New("some dns lookup error")
				}),
			},
			setMocks: func() {},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(scrapeSuccessMetricDescription, prometheus.GaugeValue, 0, "test1"),
				prometheus.MustNewConstMetric(scrapeDurationMetricDescription, prometheus.GaugeValue, 2, "test1"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetMocks()
			tc.setMocks()
			defer mc.Finish()

			co := NewMikrotikCollector(tc.devices, tc.opts...)

			metricsChan := make(chan prometheus.Metric)
			doneChan := make(chan struct{})
			var gotMetrics []prometheus.Metric
			go func() {
				defer close(doneChan)
				for metric := range metricsChan {
					gotMetrics = append(gotMetrics, metric)
				}
			}()

			co.Collect(metricsChan)
			close(metricsChan)
			<-doneChan
			r.ElementsMatch(tc.want, gotMetrics)
		})
	}
}
