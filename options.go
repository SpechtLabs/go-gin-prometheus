package ginprometheus

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// GinPrometheusOpt is a functional option that allows customization of the exporter instance configuration.
type GinPrometheusOpt func(*exporter)

// WithPushGateway configures exporter to push metrics to a remote PushGateway using specified URL, metrics source, and interval.
func WithPushGateway(pushGatewayURL, metricsURL string, pushInterval time.Duration) GinPrometheusOpt {
	return func(p *exporter) {
		p.Ppg.PushGatewayURL = pushGatewayURL
		p.Ppg.MetricsURL = metricsURL
		p.Ppg.PushInterval = pushInterval
		p.startPushTicker()
	}
}

// WithPushGatewayJob job name, defaults to "gin"
func WithPushGatewayJob(j string) GinPrometheusOpt {
	return func(p *exporter) {
		p.Ppg.Job = j
	}
}

// WithListenAddress sets the listen address for the exporter server and initializes a default Gin router if provided.
// If not set, it will be exposed at the same address of the gin engine that is being used
func WithListenAddress(address string) GinPrometheusOpt {
	return func(p *exporter) {
		p.listenAddress = address
		if p.listenAddress != "" {
			p.router = gin.Default()
		}
	}
}

// WithListenAddressWithRouter creates a GinPrometheusOpt that sets a custom listen address and router for the exporter instance.
// If a non-empty listen address is provided, the specified router is also set on the exporter instance.
// This is useful for using a separate router to expose metrics. (this keeps things like GET /metrics out of
// your content's access log).
func WithListenAddressWithRouter(listenAddress string, r *gin.Engine) GinPrometheusOpt {
	return func(p *exporter) {
		p.listenAddress = listenAddress
		if len(p.listenAddress) > 0 {
			p.router = r
		}
	}
}

// WithMetricsAuth is a functional option that enables authentication for the metrics endpoint using provided accounts.
func WithMetricsAuth(accounts gin.Accounts) GinPrometheusOpt {
	return func(p *exporter) {
		p.metricsWithAuth = true
		p.accounts = accounts
	}
}

// WithCustomMetric adds a custom metric to the list of metrics in the exporter configuration.
func WithCustomMetric(metric *Metric) GinPrometheusOpt {
	return func(p *exporter) {
		p.MetricsList = append(p.MetricsList, metric)
	}
}

// WithRegisterer sets a custom prometheus.Registerer to the exporter, allowing metrics to be registered on it.
func WithRegisterer(r prometheus.Registerer) GinPrometheusOpt {
	return func(p *exporter) {
		p.registerer = r
	}
}

// WithRequestCounterURLLabelMappingFn allows the definition of a RequestCounterURLLabelMappingFn which
// can control the cardinality of the request counter's "url" label, which might be required in some contexts.
// For instance, if for a "/customer/:name" route you don't want to generate a time series for every
// possible customer name, you could use this function:
//
//	func(c *gin.Context) string {
//		url := c.Request.URL.Path
//		for _, p := range c.Params {
//			if p.Key == "name" {
//				url = strings.Replace(url, p.Value, ":name", 1)
//				break
//			}
//		}
//		return url
//	}
//
// which would map "/customer/alice" and "/customer/bob" to their template "/customer/:name".
func WithRequestCounterURLLabelMappingFn(fn RequestCounterURLLabelMappingFn) GinPrometheusOpt {
	return func(p *exporter) {
		p.ReqCntURLLabelMappingFn = fn
	}
}

// WithLowCardinalityUrl returns a GinPrometheusOpt that modifies the URL label to reduce cardinality in metrics tracking.
func WithLowCardinalityUrl() GinPrometheusOpt {
	return func(p *exporter) {
		p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
			url := c.Request.URL.Path
			for _, p := range c.Params {
				url = strings.Replace(url, p.Value, fmt.Sprintf(":%s", p.Key), 1)
			}

			return url
		}
	}
}
