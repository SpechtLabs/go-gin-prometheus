package ginprometheus

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spechtlabs/go-otel-utils/otelzap"
)

var defaultMetricPath = "/metrics"

// RequestCounterURLLabelMappingFn is a function which can be supplied to the middleware to control
// the cardinality of the request counter's "url" label, which might be required in some contexts.
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
type RequestCounterURLLabelMappingFn func(c *gin.Context) string

// PrometheusPushGateway contains the configuration for pushing to a exporter pushgateway (optional)
type PrometheusPushGateway struct {

	// Push interval in seconds
	PushIntervalSeconds time.Duration

	// Push Gateway URL in format http://domain:port
	// where JOBNAME can be any string of your choice
	PushGatewayURL string

	// Local metrics URL where metrics are fetched from, this could be ommited in the future
	// if implemented using prometheus common/expfmt instead
	MetricsURL string

	// pushgateway job name, defaults to "gin"
	Job string
}

// exporter contains the metrics gathered by the instance and its path
type exporter struct {
	reqCnt        *prometheus.CounterVec
	reqDur        *prometheus.HistogramVec
	reqSz, resSz  prometheus.Summary
	router        *gin.Engine
	listenAddress string
	Ppg           PrometheusPushGateway
	registerer    prometheus.Registerer

	MetricsList []*Metric
	MetricsPath string

	ReqCntURLLabelMappingFn RequestCounterURLLabelMappingFn

	// gin.Context string to use as a prometheus URL label
	URLLabelFromContext string

	// Authenticated metrics endpoint
	metricsWithAuth bool
	accounts        gin.Accounts
}

// newExporter generates a new set of metrics with a certain subsystem name
func newExporter(subsystem string) *exporter {
	p := &exporter{
		MetricsList: make([]*Metric, 0),
		MetricsPath: defaultMetricPath,
		ReqCntURLLabelMappingFn: func(c *gin.Context) string {
			return c.Request.URL.Path // i.e. by default do nothing, i.e. return URL as is
		},
		metricsWithAuth: false,
		registerer:      prometheus.DefaultRegisterer,
	}

	p.registerMetrics(subsystem)

	return p
}

func (p *exporter) runServer() {
	if p.listenAddress != "" {
		go func() {
			err := p.router.Run(p.listenAddress)
			if err != nil {
				otelzap.L().WithError(err).Error("Error running server")
			}
		}()
	}
}

func (p *exporter) getMetrics() []byte {
	response, _ := http.Get(p.Ppg.MetricsURL)

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	body, _ := io.ReadAll(response.Body)

	return body
}

func (p *exporter) getPushGatewayURL() string {
	h, _ := os.Hostname()
	if p.Ppg.Job == "" {
		p.Ppg.Job = "gin"
	}
	return p.Ppg.PushGatewayURL + "/metrics/job/" + p.Ppg.Job + "/instance/" + h
}

func (p *exporter) sendMetricsToPushGateway(metrics []byte) {
	req, err := http.NewRequest("POST", p.getPushGatewayURL(), bytes.NewBuffer(metrics))
	client := &http.Client{}
	if _, err = client.Do(req); err != nil {
		otelzap.L().WithError(err).Error("Error sending to push gateway")
	}
}

func (p *exporter) startPushTicker() {
	ticker := time.NewTicker(time.Second * p.Ppg.PushIntervalSeconds)
	go func() {
		for range ticker.C {
			p.sendMetricsToPushGateway(p.getMetrics())
		}
	}()
}

func (p *exporter) registerMetrics(subsystem string) {
	m := standardMetrics
	m = append(m, p.MetricsList...)

	for _, metricDef := range m {
		metric := NewMetric(metricDef, subsystem)

		if err := p.registerer.Register(metric); err != nil {
			otelzap.L().WithError(err).Error(fmt.Sprintf("%s could not be registered in exporter", metricDef.Name))
		}
		switch metricDef {
		case reqCnt:
			p.reqCnt = metric.(*prometheus.CounterVec)
		case reqDur:
			p.reqDur = metric.(*prometheus.HistogramVec)
		case resSz:
			p.resSz = metric.(prometheus.Summary)
		case reqSz:
			p.reqSz = metric.(prometheus.Summary)
		}
		metricDef.collector = metric
	}
}

// setMetricsPath set metrics paths
func (p *exporter) setMetricsPath(e *gin.Engine) {
	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, prometheusHandler())
	}
}

// setMetricsPathWithAuth set metrics paths with authentication
func (p *exporter) setMetricsPathWithAuth(e *gin.Engine) {
	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, gin.BasicAuth(p.accounts), prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, gin.BasicAuth(p.accounts), prometheusHandler())
	}
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// From https://github.com/DanielHeckrath/gin-prometheus/blob/master/gin_prometheus.go
func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
