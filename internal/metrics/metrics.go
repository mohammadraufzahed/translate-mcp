package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Metrics struct {
	RequestsTotal      *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	CacheHitsTotal     *prometheus.CounterVec
	ProviderCallsTotal *prometheus.CounterVec
	ErrorsTotal        *prometheus.CounterVec
	InputTokensTotal   *prometheus.CounterVec
	OutputTokensTotal  *prometheus.CounterVec
	registry           *prometheus.Registry
}

func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{registry: reg}
	m.RequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_requests_total",
		Help: "Total MCP tool requests",
	}, []string{"tool"})
	m.RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_request_duration_seconds",
		Help:    "MCP tool request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool"})
	m.CacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "translate_cache_hits_total",
		Help: "Translation cache hits by tier",
	}, []string{"tier"})
	m.ProviderCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "translate_provider_calls_total",
		Help: "Provider calls by provider",
	}, []string{"provider"})
	m.ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "translate_errors_total",
		Help: "Translation errors by provider",
	}, []string{"provider"})
	m.InputTokensTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "translate_input_tokens_total",
		Help: "Estimated input tokens",
	}, []string{"provider"})
	m.OutputTokensTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "translate_output_tokens_total",
		Help: "Estimated output tokens",
	}, []string{"provider"})
	reg.MustRegister(m.RequestsTotal, m.RequestDuration, m.CacheHitsTotal, m.ProviderCallsTotal, m.ErrorsTotal, m.InputTokensTotal, m.OutputTokensTotal)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
