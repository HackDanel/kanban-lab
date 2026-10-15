// Package metrics exposes Prometheus metrics for HTTP traffic, the database
// pool and real-time connections.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics owns a registry and the HTTP instruments.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

// New registers the HTTP, runtime and pool metrics. pool may be nil.
func New(pool *pgxpool.Pool, wsClients func() float64) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total", Help: "HTTP requests by route, method and status.",
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by route and method.",
			Buckets: []float64{.005, .01, .025, .05, .1, .2, .3, .5, 1, 2.5, 5},
		}, []string{"route", "method"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_requests_in_flight", Help: "HTTP requests being served.",
		}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight,
		collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	if pool != nil {
		reg.MustRegister(
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "db_pool_acquired_conns", Help: "Connections in use."},
				func() float64 { return float64(pool.Stat().AcquiredConns()) }),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "db_pool_idle_conns", Help: "Idle connections."},
				func() float64 { return float64(pool.Stat().IdleConns()) }),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "db_pool_max_conns", Help: "Pool size limit."},
				func() float64 { return float64(pool.Stat().MaxConns()) }),
			prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "db_pool_empty_acquire_total", Help: "Acquires that had to wait for a connection."},
				func() float64 { return float64(pool.Stat().EmptyAcquireCount()) }),
		)
	}
	if wsClients != nil {
		reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "websocket_clients", Help: "Connected WebSocket clients.",
		}, wsClients))
	}
	return m
}

// Handler serves the metrics in the Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Middleware records every request under its route pattern, so that
// /cards/{cardId} is one time series rather than one per card.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.Inc()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			m.inFlight.Dec()
			route := "unmatched"
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				route = rc.RoutePattern()
			}
			m.requests.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
			m.duration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
		}()
		next.ServeHTTP(rec, r)
	})
}
