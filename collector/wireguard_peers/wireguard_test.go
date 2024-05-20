package wireguard_peers

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

func Test_wireguardPeersCollector_Name(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	r.Equal("wireguard_peers", c.Name())
}

func Test_wireguardPeersCollector_Describe(t *testing.T) {
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
		metrics.BuildMetricDescription(prefix, "since_last_handshake", "time in seconds since wireguard peer last handshake", labelNames),
		metrics.BuildMetricDescription(prefix, "rx_bytes", "received bytes from wireguard peer", labelNames),
		metrics.BuildMetricDescription(prefix, "tx_bytes", "sent bytes to wireguard peer", labelNames),
	}, got)
}

func Test_wireguardPeersCollector_Collect(t *testing.T) {
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
						"/interface/wireguard/peers/print",
						"?disabled=false",
						"=.proplist=interface,current-endpoint-address,current-endpoint-port,allowed-address,rx,tx,last-handshake",
					}, sentence)
				}).Return(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"interface":                "wg0",
								"current-endpoint-address": "192.168.1.1",
								"current-endpoint-port":    "12345",
								"allowed-address":          "0.0.0.0/0",
								"rx":                       "100",
								"tx":                       "10",
								"last-handshake":           "10s",
							},
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "since_last_handshake", "time in seconds since wireguard peer last handshake",
						[]string{"name", "address", "interface", "current_endpoint_address", "current_endpoint_port", "allowed_address"},
					),
					prometheus.GaugeValue, 10.0, "device", "address", "wg0", "192.168.1.1", "12345", "0.0.0.0/0",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "rx_bytes", "received bytes from wireguard peer",
						[]string{"name", "address", "interface", "current_endpoint_address", "current_endpoint_port", "allowed_address"},
					),
					prometheus.CounterValue, 100.0, "device", "address", "wg0", "192.168.1.1", "12345", "0.0.0.0/0",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "tx_bytes", "sent bytes to wireguard peer",
						[]string{"name", "address", "interface", "current_endpoint_address", "current_endpoint_port", "allowed_address"},
					),
					prometheus.CounterValue, 10.0, "device", "address", "wg0", "192.168.1.1", "12345", "0.0.0.0/0",
				),
			},
		},
		{
			name: "fetch error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/interface/wireguard/peers/print",
						"?disabled=false",
						"=.proplist=interface,current-endpoint-address,current-endpoint-port,allowed-address,rx,tx,last-handshake",
					}, sentence)
				}).Return(nil, errors.New("some fetch error"))
			},
			errWant: "failed to fetch wireguard peers metrics: some fetch error",
		},
		{
			name: "parse error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/interface/wireguard/peers/print",
						"?disabled=false",
						"=.proplist=interface,current-endpoint-address,current-endpoint-port,allowed-address,rx,tx,last-handshake",
					}, sentence)
				}).Return(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"interface":                "wg0",
								"current-endpoint-address": "192.168.1.1",
								"current-endpoint-port":    "12345",
								"allowed-address":          "0.0.0.0/0",
								"rx":                       "100",
								"tx":                       "10",
								"last-handshake":           "s10s",
							},
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "rx_bytes", "received bytes from wireguard peer",
						[]string{"name", "address", "interface", "current_endpoint_address", "current_endpoint_port", "allowed_address"},
					),
					prometheus.CounterValue, 100.0, "device", "address", "wg0", "192.168.1.1", "12345", "0.0.0.0/0",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "tx_bytes", "sent bytes to wireguard peer",
						[]string{"name", "address", "interface", "current_endpoint_address", "current_endpoint_port", "allowed_address"},
					),
					prometheus.CounterValue, 10.0, "device", "address", "wg0", "192.168.1.1", "12345", "0.0.0.0/0",
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
