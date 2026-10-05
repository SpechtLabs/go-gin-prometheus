package ginprometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGinPrometheusMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Each case registers its metrics on the default registry under a
	// subsystem of its own, so the cases don't collide.
	tests := []struct {
		name       string
		subsystem  string
		opts       []GinPrometheusOpt
		auth       bool
		wantSeries string
	}{
		{
			name:       "records the request path",
			subsystem:  "test_path",
			wantSeries: `test_path_requests_total{code="200",handler="github.com/spechtlabs/go-gin-prometheus.TestGinPrometheusMiddleware.func1.1",host="example.com",method="GET",url="/hello/alice"} 1`,
		},
		{
			name:       "low cardinality URL",
			subsystem:  "test_low_cardinality",
			opts:       []GinPrometheusOpt{WithLowCardinalityUrl()},
			wantSeries: `test_low_cardinality_requests_total{code="200",handler="github.com/spechtlabs/go-gin-prometheus.TestGinPrometheusMiddleware.func1.1",host="example.com",method="GET",url="/hello/:name"} 1`,
		},
		{
			name:       "metrics behind basic auth",
			subsystem:  "test_auth",
			opts:       []GinPrometheusOpt{WithMetricsAuth(gin.Accounts{"user": "secret"})},
			auth:       true,
			wantSeries: `test_auth_requests_total{code="200",handler="github.com/spechtlabs/go-gin-prometheus.TestGinPrometheusMiddleware.func1.1",host="example.com",method="GET",url="/hello/alice"} 1`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(GinPrometheusMiddleware(r, tt.subsystem, tt.opts...))
			r.GET("/hello/:name", func(c *gin.Context) { c.String(http.StatusOK, "hello") })

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://example.com/hello/alice", http.NoBody))
			assert.Equal(t, http.StatusOK, w.Code)

			if tt.auth {
				w = httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			}

			req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
			if tt.auth {
				req.SetBasicAuth("user", "secret")
			}
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantSeries)
			// Scraping the metrics endpoint isn't a request the metrics count.
			assert.NotContains(t, w.Body.String(), `url="/metrics"`)
		})
	}
}
