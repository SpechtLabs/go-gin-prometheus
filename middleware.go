package ginprometheus

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GinPrometheusMiddleware(e *gin.Engine, subsystem string, opts ...GinPrometheusOpt) gin.HandlerFunc {
	p := newExporter(subsystem)

	// Build exporter with all the options
	for _, opt := range opts {
		opt(p)
	}
	if p.metricsWithAuth {
		p.setMetricsPathWithAuth(e)
	} else {
		p.setMetricsPath(e)
	}

	return func(c *gin.Context) {
		if c.Request.URL.Path == p.MetricsPath {
			c.Next()
			return
		}

		start := time.Now()
		reqSz := computeApproximateRequestSize(c.Request)

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		resSz := float64(c.Writer.Size())

		url := p.ReqCntURLLabelMappingFn(c)
		// jlambert Oct 2018 - sidecar specific mod
		if len(p.URLLabelFromContext) > 0 {
			u, found := c.Get(p.URLLabelFromContext)
			if !found {
				u = "unknown"
			}
			url = u.(string)
		}
		p.reqDur.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
		p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, url).Inc()
		p.reqSz.Observe(float64(reqSz))
		p.resSz.Observe(resSz)
	}
}
