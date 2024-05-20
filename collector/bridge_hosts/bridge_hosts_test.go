package bridge_hosts

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"

	"github.com/psolru/mikrotik-exporter/collector/context"
	"github.com/psolru/mikrotik-exporter/metrics"
	"github.com/psolru/mikrotik-exporter/routeros/mocks"
)

func Test_bridgeHostsCollector_Name(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	r.Equal("bridge_host", c.Name())
}

func Test_bridgeHostsCollector_Describe(t *testing.T) {
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
		metrics.BuildMetricDescription(prefix, "status", "bridge host status",
			[]string{"name", "address", "bridge", "mac_address", "on_interface", "dynamic", "local", "external"},
		),
	}, got)
}

func Test_bridgeHostsCollector_Collect(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	routerOSClientMock := mocks.NewClientMock(t)
	resetMocks := func() {
		routerOSClientMock = mocks.NewClientMock(t)
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
						"/interface/bridge/host/print",
						"?disabled=false",
						"=.proplist=bridge,mac-address,on-interface,dynamic,local,external",
					}, sentence)
				}).Return(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"bridge":       "bridge",
								"mac-address":  "mac-address",
								"on-interface": "ether1",
								"dynamic":      "true",
								"local":        "false",
								"external":     "true",
							},
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "status", "bridge host status",
						[]string{"name", "address", "bridge", "mac_address", "on_interface", "dynamic", "local", "external"},
					),
					prometheus.GaugeValue, 1.0, "device", "address", "bridge", "mac-address", "ether1", "true", "false", "true",
				),
			},
		},
		{
			name: "fetch error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/interface/bridge/host/print",
						"?disabled=false",
						"=.proplist=bridge,mac-address,on-interface,dynamic,local,external",
					}, sentence)
				}).Return(nil, errors.New("some fetch error"))
			},
			errWant: "failed to fetch bridge hosts metrics: some fetch error",
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
