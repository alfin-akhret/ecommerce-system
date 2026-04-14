package helper

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "total number of http requests",
	},
	[]string{"method", "endpoint", "status"},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

// metrics middleware
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// exclude endpoints named "metrics"
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     200,
		}

		next.ServeHTTP(rec, r)

		httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(rec.statusCode),
		).Inc()
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
