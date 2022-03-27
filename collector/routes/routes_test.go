package routes

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

func Test_routesCollector_Name(t *testing.T) {
	r := require.New(t)

	c := NewCollector()

	r.Equal("routes", c.Name())
}

func Test_routesCollector_Describe(t *testing.T) {
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
		metrics.BuildMetricDescription(prefix, "total", "number of routes in rib", labelNames),
		metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
	}, got)
}

func Test_routesCollector_Collect(t *testing.T) {
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
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "10.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?bgp",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?connect",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "10.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?ospf",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?dynamic",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?static",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)

				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "1.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?bgp",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?ospf",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?connect",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "1.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?dynamic",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?static",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0.0",
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "total", "number of routes in rib", labelNames),
					prometheus.GaugeValue, 10.0, "device", "address", "4",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "total", "number of routes in rib", labelNames),
					prometheus.GaugeValue, 1.0, "device", "address", "6",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "4", "bgp",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 10.0, "device", "address", "4", "connect",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "4", "dynamic",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "4", "static",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "4", "ospf",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "6", "bgp",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 1.0, "device", "address", "6", "connect",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "6", "dynamic",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "6", "static",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 0.0, "device", "address", "6", "ospf",
				),
			},
		},
		{
			name: "fetch error",
			setMocks: func() {
				routerOSClientMock.RunMock.Set(func(sentence ...string) (rp1 *routeros.Reply, err error) {
					return nil, errors.New("some fetch error")
				})
			},
			errWant: "failed to fetch routes by protocol: some fetch error",
		},
		{
			name: "parse error",
			setMocks: func() {
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "10.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?bgp",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?connect",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "10.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?ospf",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?dynamic",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ip/route/print",
					"?disabled=false",
					"?static",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)

				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "1.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?bgp",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?ospf",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?connect",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "1.0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?dynamic",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
				routerOSClientMock.RunMock.When([]string{
					"/ipv6/route/print",
					"?disabled=false",
					"?static",
					"=count-only=",
				}...).Then(&routeros.Reply{
					Done: &proto.Sentence{
						Map: map[string]string{
							"ret": "0,0",
						},
					},
				}, nil)
			},
			want: []prometheus.Metric{
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "total", "number of routes in rib", labelNames),
					prometheus.GaugeValue, 10.0, "device", "address", "4",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "total", "number of routes in rib", labelNames),
					prometheus.GaugeValue, 1.0, "device", "address", "6",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 10.0, "device", "address", "4", "connect",
				),
				prometheus.MustNewConstMetric(
					metrics.BuildMetricDescription(prefix, "by_protocol", "number of routes per protocol in rib", append(labelNames, "protocol")),
					prometheus.GaugeValue, 1.0, "device", "address", "6", "connect",
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
