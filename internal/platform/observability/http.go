package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	inFlightRequests prometheus.Gauge
}

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return registry
}

func NewHTTPMetrics(registry *prometheus.Registry) *HTTPMetrics {
	metrics := &HTTPMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "med_go",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests handled by the service.",
			},
			[]string{"service", "method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "med_go",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"service", "method", "route", "status"},
		),
		inFlightRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "med_go",
				Name:      "http_in_flight_requests",
				Help:      "Current number of HTTP requests being served.",
			},
		),
	}

	registry.MustRegister(
		metrics.requestsTotal,
		metrics.requestDuration,
		metrics.inFlightRequests,
	)

	return metrics
}

func (m *HTTPMetrics) Middleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.inFlightRequests.Inc()
		defer m.inFlightRequests.Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())
		labels := prometheus.Labels{
			"service": service,
			"method":  c.Request.Method,
			"route":   route,
			"status":  status,
		}

		m.requestsTotal.With(labels).Inc()
		m.requestDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}

func MetricsHandler(registry *prometheus.Registry) gin.HandlerFunc {
	return gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}
