package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector collects and manages application metrics
type Collector struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize   *prometheus.HistogramVec

	// SSH metrics
	sshConnectionsTotal    *prometheus.CounterVec
	sshConnectionDuration *prometheus.HistogramVec
	sshConnectionErrors   *prometheus.CounterVec

	// Health check metrics
	healthChecksTotal    *prometheus.CounterVec
	healthCheckDuration *prometheus.HistogramVec
	healthCheckStatus   *prometheus.GaugeVec

	// Job metrics
	jobsTotal     *prometheus.CounterVec
	jobDuration   *prometheus.HistogramVec
	jobStatus     *prometheus.GaugeVec
	jobQueueSize  prometheus.Gauge

	// Server metrics
	serverCount    prometheus.Gauge
	serversByTag  *prometheus.GaugeVec
	serverStatus   *prometheus.GaugeVec

	// System metrics
	cpuUsage    *prometheus.GaugeVec
	memoryUsage *prometheus.GaugeVec
	diskUsage   *prometheus.GaugeVec
	networkIO   *prometheus.GaugeVec

	// Plugin metrics
	pluginExecutionsTotal *prometheus.CounterVec
	pluginDuration       *prometheus.HistogramVec
	pluginErrors        *prometheus.CounterVec

	// Custom metrics
	customMetrics map[string]prometheus.Metric
	mu            sync.RWMutex
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint"},
		),

		sshConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_ssh_connections_total",
				Help: "Total number of SSH connections",
			},
			[]string{"server", "status"},
		),
		sshConnectionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_ssh_connection_duration_seconds",
				Help:    "SSH connection duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"server"},
		),
		sshConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_ssh_connection_errors_total",
				Help: "Total number of SSH connection errors",
			},
			[]string{"server", "error_type"},
		),

		healthChecksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_health_checks_total",
				Help: "Total number of health checks",
			},
			[]string{"server", "check_type"},
		),
		healthCheckDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_health_check_duration_seconds",
				Help:    "Health check duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"server", "check_type"},
		),
		healthCheckStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_health_check_status",
				Help: "Health check status (1=healthy, 0=unhealthy)",
			},
			[]string{"server", "check_type"},
		),

		jobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_jobs_total",
				Help: "Total number of jobs executed",
			},
			[]string{"server", "job_type", "status"},
		),
		jobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_job_duration_seconds",
				Help:    "Job execution duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"server", "job_type"},
		),
		jobStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_job_status",
				Help: "Current job status (1=running, 0=stopped)",
			},
			[]string{"server", "job_id"},
		),
		jobQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "vps_tools_job_queue_size",
				Help: "Current job queue size",
			},
		),

		serverCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "vps_tools_server_count",
				Help: "Total number of configured servers",
			},
		),
		serversByTag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_servers_by_tag",
				Help: "Number of servers by tag",
			},
			[]string{"tag"},
		),
		serverStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_server_status",
				Help: "Server status (1=online, 0=offline)",
			},
			[]string{"server"},
		),

		cpuUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
			[]string{"server"},
		),
		memoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_memory_usage_percent",
				Help: "Memory usage percentage",
			},
			[]string{"server"},
		),
		diskUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_disk_usage_percent",
				Help: "Disk usage percentage",
			},
			[]string{"server", "mount_point"},
		),
		networkIO: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "vps_tools_network_io_bytes_per_second",
				Help: "Network I/O in bytes per second",
			},
			[]string{"server", "direction"},
		),

		pluginExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_plugin_executions_total",
				Help: "Total number of plugin executions",
			},
			[]string{"plugin", "action", "status"},
		),
		pluginDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vps_tools_plugin_duration_seconds",
				Help:    "Plugin execution duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"plugin", "action"},
		),
		pluginErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vps_tools_plugin_errors_total",
				Help: "Total number of plugin errors",
			},
			[]string{"plugin", "error_type"},
		),

		customMetrics: make(map[string]prometheus.Metric),
	}
}

// HTTP Metrics

// RecordHTTPRequest records an HTTP request
func (c *Collector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration, size int64) {
	c.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	c.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	c.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// SSH Metrics

// RecordSSHConnection records an SSH connection
func (c *Collector) RecordSSHConnection(server, status string, duration time.Duration) {
	c.sshConnectionsTotal.WithLabelValues(server, status).Inc()
	c.sshConnectionDuration.WithLabelValues(server).Observe(duration.Seconds())
}

// RecordSSHError records an SSH connection error
func (c *Collector) RecordSSHError(server, errorType string) {
	c.sshConnectionErrors.WithLabelValues(server, errorType).Inc()
}

// Health Check Metrics

// RecordHealthCheck records a health check
func (c *Collector) RecordHealthCheck(server, checkType string, duration time.Duration, status float64) {
	c.healthChecksTotal.WithLabelValues(server, checkType).Inc()
	c.healthCheckDuration.WithLabelValues(server, checkType).Observe(duration.Seconds())
	c.healthCheckStatus.WithLabelValues(server, checkType).Set(status)
}

// Job Metrics

// RecordJob records a job execution
func (c *Collector) RecordJob(server, jobType, status string, duration time.Duration) {
	c.jobsTotal.WithLabelValues(server, jobType, status).Inc()
	c.jobDuration.WithLabelValues(server, jobType).Observe(duration.Seconds())
}

// SetJobStatus sets the current job status
func (c *Collector) SetJobStatus(server, jobID string, status float64) {
	c.jobStatus.WithLabelValues(server, jobID).Set(status)
}

// SetJobQueueSize sets the current job queue size
func (c *Collector) SetJobQueueSize(size float64) {
	c.jobQueueSize.Set(size)
}

// Server Metrics

// SetServerCount sets the total number of servers
func (c *Collector) SetServerCount(count float64) {
	c.serverCount.Set(count)
}

// SetServersByTag sets the number of servers by tag
func (c *Collector) SetServersByTag(tag string, count float64) {
	c.serversByTag.WithLabelValues(tag).Set(count)
}

// SetServerStatus sets the server status
func (c *Collector) SetServerStatus(server string, status float64) {
	c.serverStatus.WithLabelValues(server).Set(status)
}

// System Metrics

// SetCPUUsage sets the CPU usage for a server
func (c *Collector) SetCPUUsage(server string, usage float64) {
	c.cpuUsage.WithLabelValues(server).Set(usage)
}

// SetMemoryUsage sets the memory usage for a server
func (c *Collector) SetMemoryUsage(server string, usage float64) {
	c.memoryUsage.WithLabelValues(server).Set(usage)
}

// SetDiskUsage sets the disk usage for a server
func (c *Collector) SetDiskUsage(server, mountPoint string, usage float64) {
	c.diskUsage.WithLabelValues(server, mountPoint).Set(usage)
}

// SetNetworkIO sets the network I/O for a server
func (c *Collector) SetNetworkIO(server, direction string, bytesPerSecond float64) {
	c.networkIO.WithLabelValues(server, direction).Set(bytesPerSecond)
}

// Plugin Metrics

// RecordPluginExecution records a plugin execution
func (c *Collector) RecordPluginExecution(plugin, action, status string, duration time.Duration) {
	c.pluginExecutionsTotal.WithLabelValues(plugin, action, status).Inc()
	c.pluginDuration.WithLabelValues(plugin, action).Observe(duration.Seconds())
}

// RecordPluginError records a plugin error
func (c *Collector) RecordPluginError(plugin, errorType string) {
	c.pluginErrors.WithLabelValues(plugin, errorType).Inc()
}

// Custom Metrics

// RegisterCustomMetric registers a custom metric
func (c *Collector) RegisterCustomMetric(name string, metric prometheus.Metric) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.customMetrics[name]; exists {
		return fmt.Errorf("custom metric %s already registered", name)
	}

	c.customMetrics[name] = metric
	return nil
}

// UnregisterCustomMetric unregisters a custom metric
func (c *Collector) UnregisterCustomMetric(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.customMetrics, name)
}

// GetCustomMetric gets a custom metric by name
func (c *Collector) GetCustomMetric(name string) (prometheus.Metric, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metric, exists := c.customMetrics[name]
	if !exists {
		return nil, fmt.Errorf("custom metric %s not found", name)
	}

	return metric, nil
}

// ListCustomMetrics lists all custom metrics
func (c *Collector) ListCustomMetrics() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var names []string
	for name := range c.customMetrics {
		names = append(names, name)
	}

	return names
}

// MetricsCollector interface for custom metric collectors
type MetricsCollector interface {
	Collect(ctx context.Context) ([]MetricValue, error)
	Name() string
	Description() string
}

// MetricValue represents a single metric value
type MetricValue struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// RegisterMetricsCollector registers a custom metrics collector
func (c *Collector) RegisterMetricsCollector(collector MetricsCollector) error {
	// Create a custom gauge metric for the collector
	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: collector.Name(),
			Help: collector.Description(),
		},
		[]string{}, // Labels will be added dynamically
	)

	// Start collection goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			values, err := collector.Collect(context.Background())
			if err != nil {
				continue
			}

			for _, value := range values {
				// Convert labels to label values
				var labelValues []string
				for _, labelValue := range value.Labels {
					labelValues = append(labelValues, labelValue)
				}

				gauge.WithLabelValues(labelValues...).Set(value.Value)
			}
		}
	}()

	return nil
}