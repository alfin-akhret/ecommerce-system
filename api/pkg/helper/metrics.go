package helper

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
)

// 1. Traffic
var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "total number of http requests",
	},
	[]string{"method", "path", "status"},
)

// 2. latency
var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

// 3. errors
var httpErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total HTTP errors",
	},
	[]string{"method", "path", "status"},
)

// 4. server load
var httpInFlight = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Current in-flight requests",
	},
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpErrorsTotal,
		httpInFlight,
	)
}

// metrics middleware
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// exclude path named "metrics"
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unknown"
		}

		start := time.Now()

		httpInFlight.Inc()
		defer httpInFlight.Dec()

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     200,
		}

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(rec.statusCode),
		).Inc()

		httpRequestDuration.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(rec.statusCode),
		).Observe(duration)

		if rec.statusCode >= 400 {
			httpErrorsTotal.WithLabelValues(
				r.Method,
				routePattern,
				strconv.Itoa(rec.statusCode),
			).Inc()
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
