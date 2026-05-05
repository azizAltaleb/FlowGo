package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by service, method, route and status code.",
	}, []string{"service", "method", "route", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "route"})
)

// Handler returns the Prometheus HTTP metrics handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// MuxMiddleware returns a gorilla/mux compatible middleware that records
// http_requests_total and http_request_duration_seconds for the given service.
func MuxMiddleware(service string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routeLabel(r)
			start := time.Now()
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.status)
			httpRequestsTotal.WithLabelValues(service, r.Method, route, status).Inc()
			httpRequestDuration.WithLabelValues(service, r.Method, route).Observe(duration)
		})
	}
}

func routeLabel(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if tpl, err := route.GetPathTemplate(); err == nil {
			return tpl
		}
	}
	path := r.URL.Path
	if path == "" {
		return "/"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 3 {
		return "/" + strings.Join(parts[:3], "/")
	}
	return path
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
