// Metrics holds Prometheus collectors for the service.
package monitoring

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yaninyzwitty/chat/packages/shared/util"
)

type Metrics struct {
	Stage    prometheus.Gauge
	Duration *prometheus.HistogramVec
	Errors   *prometheus.CounterVec
}

// NewMetrics registers and returns a Metrics instance.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Stage: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "myapp",
			Name:      "stage",
			Help:      "Current stage of the application/test run",
		}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "myapp",
			Name:      "request_duration_seconds",
			Help:      "Duration of gRPC or DB requests in seconds",
			Buckets:   prometheus.DefBuckets, // standard buckets: 0.005 → 10s
		}, []string{"op", "db"}),
		Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "myapp",
			Name:      "errors_total",
			Help:      "Count of errors by operation and backend",
		}, []string{"op", "db"}),
	}

	// register metrics with prometheus
	reg.MustRegister(m.Stage, m.Duration, m.Errors)
	return m
}

// StartPrometheusServer launches an HTTP server for Prometheus scraping.
func StartPrometheusServer(reg *prometheus.Registry, addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		slog.Info("[METRICS] Starting Prometheus Server on ", "addr", addr)
		err := http.ListenAndServe(addr, mux)
		util.Fail(err, "failed to start Prometheus server on %s", addr)
	}()

}
