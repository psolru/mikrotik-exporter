package bgp

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"

	"github.com/ogi4i/mikrotik-exporter/collector/context"
	"github.com/ogi4i/mikrotik-exporter/metrics"
	"github.com/ogi4i/mikrotik-exporter/routeros/mocks"
)

func Test_bgpCollector_Name(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	r.Equal("bgp_session", c.Name())
}

func Test_bgpCollector_Describe(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	ch := make(chan *prometheus.Desc)
	done := make(chan struct{})
	var got []*prometheus.Desc
	go func() {
		defer close(done)
		for desc := range ch {
			got = append(got, desc)
		}
	}()

	c.Describe(ch)
	close(ch)

	<-done
	r.ElementsMatch([]*prometheus.Desc{
		metrics.BuildMetricDescription(prefix, "state", "bgp session state (up = 1)", labelNames),
		metrics.BuildMetricDescription(prefix, "prefix_count", "number of prefixes per session", labelNames),
		metrics.BuildMetricDescription(prefix, "updates_sent", "number of bgp updates sent per session", labelNames),
		metrics.BuildMetricDescription(prefix, "updates_received", "number of bgp updates received per session", labelNames),
		metrics.BuildMetricDescription(prefix, "withdrawn_sent", "number of bgp withdrawns sent per session", labelNames),
		metrics.BuildMetricDescription(prefix, "withdrawn_received", "number of bgp withdrawns received per session", labelNames),
	}, got)
}

func Test_bgpCollector_Collect(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	routerOSClientMock := mocks.NewRouterOSClientMock(t)
	resetMocks := func() {
		routerOSClientMock = mocks.NewRouterOSClientMock(t)
	}

	testCases := []struct {
		name     string
		setMocks func()
		want     []prometheus.Metric
		errWant  string
	}{
		{
			name: "success",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/routing/bgp/peer/print",
						"=.proplist=name,remote-as,state,prefix-count,updates-sent,updates-received,withdrawn-sent,withdrawn-received",
					}, sentence)
				}).Return(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"name":               "session",
								"remote-as":          "65000",
								"state":              "established",
								"prefix-count":       "111",
								"updates-sent":       "11",
								"updates-received":   "21",
								"withdrawn-sent":     "1",
								"withdrawn-received": "2",
							},
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "state", "bgp session state (up = 1)", labelNames),
					prometheus.GaugeValue, 1, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "prefix_count", "number of prefixes per session", labelNames),
					prometheus.GaugeValue, 111, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "updates_sent", "number of bgp updates sent per session", labelNames),
					prometheus.GaugeValue, 11, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "updates_received", "number of bgp updates received per session", labelNames),
					prometheus.GaugeValue, 21, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "withdrawn_sent", "number of bgp withdrawns sent per session", labelNames),
					prometheus.GaugeValue, 1, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "withdrawn_received", "number of bgp withdrawns received per session", labelNames),
					prometheus.GaugeValue, 2, "device", "address", "session", "65000",
				),
			},
		},
		{
			name: "fetch error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/routing/bgp/peer/print",
						"=.proplist=name,remote-as,state,prefix-count,updates-sent,updates-received,withdrawn-sent,withdrawn-received",
					}, sentence)
				}).Return(nil, errors.New("some fetch error"))
			},
			errWant: "failed to fetch bgp metrics: some fetch error",
		},
		{
			name: "parse error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/routing/bgp/peer/print",
						"=.proplist=name,remote-as,state,prefix-count,updates-sent,updates-received,withdrawn-sent,withdrawn-received",
					}, sentence)
				}).Return(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"name":               "session",
								"remote-as":          "65000",
								"state":              "other-state",
								"prefix-count":       "a111",
								"updates-sent":       "11",
								"updates-received":   "21",
								"withdrawn-sent":     "1",
								"withdrawn-received": "2",
							},
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "state", "bgp session state (up = 1)", labelNames),
					prometheus.GaugeValue, 0, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "updates_sent", "number of bgp updates sent per session", labelNames),
					prometheus.GaugeValue, 11, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "updates_received", "number of bgp updates received per session", labelNames),
					prometheus.GaugeValue, 21, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "withdrawn_sent", "number of bgp withdrawns sent per session", labelNames),
					prometheus.GaugeValue, 1, "device", "address", "session", "65000",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "withdrawn_received", "number of bgp withdrawns received per session", labelNames),
					prometheus.GaugeValue, 2, "device", "address", "session", "65000",
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetMocks()
			tc.setMocks()
			defer routerOSClientMock.MinimockFinish()

			ch := make(chan prometheus.Metric)
			done := make(chan struct{})
			var got []prometheus.Metric
			go func() {
				defer close(done)
				for desc := range ch {
					got = append(got, desc)
				}
			}()

			errGot := c.Collect(&context.Context{
				RouterOSClient: routerOSClientMock,
				MetricsChan:    ch,
				DeviceName:     "device",
				DeviceAddress:  "address",
			})
			close(ch)
			if len(tc.errWant) != 0 {
				r.EqualError(errGot, tc.errWant)
			} else {
				r.NoError(errGot)
			}

			<-done
			r.ElementsMatch(tc.want, got)
		})
	}
}
