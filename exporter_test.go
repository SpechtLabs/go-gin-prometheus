package ginprometheus

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMetrics(t *testing.T) {
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "gin_requests_total 1\n")
	}))
	t.Cleanup(metrics.Close)

	// A server that's closed refuses the connection.
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{name: "reads the metrics", url: metrics.URL, want: "gin_requests_total 1\n"},
		{name: "unreachable endpoint", url: gone.URL, wantErr: true},
		{name: "invalid URL", url: "http://[::1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &exporter{Ppg: PrometheusPushGateway{MetricsURL: tt.url}}
			got, err := p.getMetrics()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestSendMetricsToPushGateway(t *testing.T) {
	type push struct {
		method, path, body string
	}
	pushes := make(chan push, 1)
	gateway := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		pushes <- push{method: r.Method, path: r.URL.Path, body: string(body)}
	}))
	t.Cleanup(gateway.Close)

	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()

	host, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		job      string
		wantPath string
		wantPush bool
	}{
		{name: "default job", url: gateway.URL, wantPath: "/metrics/job/gin/instance/" + host, wantPush: true},
		{name: "configured job", url: gateway.URL, job: "api", wantPath: "/metrics/job/api/instance/" + host, wantPush: true},
		// Neither failure reaches the gateway; both are logged.
		{name: "unreachable gateway", url: gone.URL},
		{name: "invalid URL", url: "http://[::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &exporter{Ppg: PrometheusPushGateway{PushGatewayURL: tt.url, Job: tt.job}}
			p.sendMetricsToPushGateway([]byte("m 1\n"))
			if !tt.wantPush {
				assert.Empty(t, pushes)
				return
			}
			got := <-pushes
			assert.Equal(t, push{method: http.MethodPost, path: tt.wantPath, body: "m 1\n"}, got)
		})
	}
}

func TestPushMetrics(t *testing.T) {
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "m 1\n")
	}))
	t.Cleanup(metrics.Close)

	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()

	tests := []struct {
		name       string
		metricsURL string
		wantPush   bool
	}{
		{name: "pushes what the metrics endpoint serves", metricsURL: metrics.URL, wantPush: true},
		// The failure is logged, and nothing reaches the gateway.
		{name: "unreachable metrics endpoint", metricsURL: gone.URL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pushed := make(chan string, 1)
			gateway := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				pushed <- string(body)
			}))
			t.Cleanup(gateway.Close)

			p := &exporter{Ppg: PrometheusPushGateway{PushGatewayURL: gateway.URL, MetricsURL: tt.metricsURL}}
			p.pushMetrics()
			if !tt.wantPush {
				assert.Empty(t, pushed)
				return
			}
			assert.Equal(t, "m 1\n", <-pushed)
		})
	}
}

func TestPushTicker(t *testing.T) {
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "m 1\n")
	}))
	t.Cleanup(metrics.Close)

	pushed := make(chan string, 1)
	gateway := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		select {
		case pushed <- string(body):
		default:
		}
	}))
	t.Cleanup(gateway.Close)

	// The interval counts in seconds; see startPushTicker.
	p := &exporter{}
	WithPushGateway(gateway.URL, metrics.URL, 1)(p)

	select {
	case body := <-pushed:
		assert.Equal(t, "m 1\n", body)
	case <-time.After(5 * time.Second):
		t.Fatal("no push reached the gateway")
	}
}
