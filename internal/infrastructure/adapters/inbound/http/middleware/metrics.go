package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// httpRequestsTotal counts the total number of HTTP requests handled,
	// partitioned by method, path, and status code.
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// httpRequestDuration measures the latency of HTTP requests in seconds,
	// partitioned by method, path, and status code.
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets, // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"method", "path", "status"},
	)

	registerMetricsOnce sync.Once
)

func registerMetrics() {
	registerMetricsOnce.Do(func() {
		_ = prometheus.Register(httpRequestsTotal)
		_ = prometheus.Register(httpRequestDuration)
	})
}

// MetricsMiddleware records Prometheus metrics for each HTTP request.
//
// It tracks two metrics:
//   - http_requests_total (counter): total requests by method, path, status
//   - http_request_duration_seconds (histogram): latency by method, path, status
//
// This middleware should be registered AFTER RequestIDMiddleware and BEFORE
// the business-logic handlers so it captures all requests.
func MetricsMiddleware() gin.HandlerFunc {
	registerMetrics()

	return func(c *gin.Context) {
		start := time.Now()

		// Execute the handler chain
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		path := c.FullPath() // e.g. "/api/v1/users/:id" — avoids high cardinality

		// If the route was not matched (404), use a fixed label
		if path == "" {
			path = "unmatched"
		}

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	}
}
