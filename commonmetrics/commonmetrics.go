package commonmetrics

import (
	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Helper functions for creating Prometheus metrics with service name prefix
func NewCounter(suffix, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: getServiceName() + suffix,
		Help: help,
	})
}

func NewGauge(suffix, help string) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name: getServiceName() + suffix,
		Help: help,
	})
}

func NewHistogram(suffix, help string, buckets []float64) prometheus.Histogram {
	return promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    getServiceName() + suffix,
		Help:    help,
		Buckets: buckets,
	})
}

// getServiceName ensures the configuration is loaded before accessing the service name
func getServiceName() string {
	return commonconfig.GetConfig().GetServiceName()
}

var (
	HeartbeatCount         prometheus.Counter
	HeartbeatMessage       prometheus.Gauge
	ServiceStartTime       prometheus.Gauge
	NumberOfErrors         prometheus.Counter
	NumberOfPings          prometheus.Counter
	UnauthorizedRequests   prometheus.Counter
	NumberOfConfigRequests prometheus.Counter
	NumberOfStatusRequests prometheus.Counter
)

// InitializeMetrics initializes all Prometheus metrics after configuration is loaded
func InitializeMetrics() {
	HeartbeatCount = NewCounter("_heartbeat_count", "The total number of executed heartbeats")
	HeartbeatMessage = NewGauge("_heartbeat_message", "The last heartbeat received")
	ServiceStartTime = NewGauge("_service_start_time", "The last time the service was started")
	NumberOfErrors = NewCounter("_error_count", "The total number of errors")
	NumberOfPings = NewCounter("_ping_count", "Number of pings requested")
	UnauthorizedRequests = NewCounter("_unauthorized_requests_count", "The total number of unauthorized requests")
	NumberOfConfigRequests = NewCounter("_config_requests_count", "The total number of configuration requests")
	NumberOfStatusRequests = NewCounter("_status_requests_count", "The total number of status requests")
	commonlogger.Debug("Metrics initialized successfully", "package", "metrics", "service", commonconfig.GetConfig().GetServiceName())
}
