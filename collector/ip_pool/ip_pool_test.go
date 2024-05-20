package ip_pool

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

func Test_ipPoolCollector_Name(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	r.Equal("ip_pool", c.Name())
}

func Test_ipPoolCollector_Describe(t *testing.T) {
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
		metrics.BuildMetricDescription(prefix, "used", "number of used ip/prefixes in pool",
			[]string{"name", "address", "ip_version", "pool"},
		),
	}, got)
}

func Test_ipPoolCollector_Collect(t *testing.T) {
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
				routerOSClientMock.RunMock.When([]string{
					"/ip/pool/print",
					"=.proplist=name",
				}...).Then(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"name": "pool1",
							},
						},
					},
				}, nil)

				routerOSClientMock.RunMock.When([]string{
					"/ip/pool/used/print",
					"?pool=pool1",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "3",
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "used", "number of used ip/prefixes in pool",
						[]string{"name", "address", "ip_version", "pool"},
					),
					prometheus.GaugeValue, 3, "device", "address", "4", "pool1",
				),
			},
		},
		{
			name: "fetch error",
			setMocks: func() {
				routerOSClientMock.RunMock.Inspect(func(sentence ...string) {
					r.Equal([]string{
						"/ip/pool/print",
						"=.proplist=name",
					}, sentence)
				}).Return(nil, errors.New("some fetch error"))
			},
			errWant: "failed to fetch ip pool names: some fetch error",
		},
		{
			name: "parse error",
			setMocks: func() {
				routerOSClientMock.RunMock.When([]string{
					"/ip/pool/print",
					"=.proplist=name",
				}...).Then(&routeros.Reply{
					Re: []*proto.Sentence{
						{
							Map: map[string]string{
								"name": "pool1",
							},
						},
					},
				}, nil)

				routerOSClientMock.RunMock.When([]string{
					"/ip/pool/used/print",
					"?pool=pool1",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "a3",
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{},
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
