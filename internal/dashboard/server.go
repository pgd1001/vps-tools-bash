package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pgd1001/vps-tools/internal/metrics"
)

// Dashboard provides a web-based monitoring dashboard
type Dashboard struct {
	server      *http.Server
	router      *mux.Router
	collector   *metrics.Collector
	port        int
	enableAuth  bool
	username    string
	password    string
}

// Config holds dashboard configuration
type Config struct {
	Port       int    `yaml:"port"`
	EnableAuth bool   `yaml:"enable_auth"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
}

// NewDashboard creates a new dashboard instance
func NewDashboard(config Config, collector *metrics.Collector) *Dashboard {
	d := &Dashboard{
		collector:  collector,
		port:       config.Port,
		enableAuth:  config.EnableAuth,
		username:    config.Username,
		password:    config.Password,
		router:      mux.NewRouter(),
	}

	d.setupRoutes()
	d.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", d.port),
		Handler:      d.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return d
}

// setupRoutes sets up the dashboard routes
func (d *Dashboard) setupRoutes() {
	// API routes
	api := d.router.PathPrefix("/api/v1").Subrouter()
	
	// Metrics endpoint (Prometheus format)
	api.Handle("/metrics", promhttp.Handler()).Methods("GET")
	
	// Dashboard data endpoints
	api.HandleFunc("/dashboard", d.authMiddleware(d.getDashboardData)).Methods("GET")
	api.HandleFunc("/servers", d.authMiddleware(d.getServersData)).Methods("GET")
	api.HandleFunc("/health", d.authMiddleware(d.getHealthData)).Methods("GET")
	api.HandleFunc("/jobs", d.authMiddleware(d.getJobsData)).Methods("GET")
	api.HandleFunc("/metrics/summary", d.authMiddleware(d.getMetricsSummary)).Methods("GET")

	// Static files and web interface
	d.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/")))).Methods("GET")
	d.router.HandleFunc("/", d.authMiddleware(d.serveIndex)).Methods("GET")
	d.router.HandleFunc("/dashboard", d.authMiddleware(d.serveDashboard)).Methods("GET")

	// Health check endpoint
	d.router.HandleFunc("/health", d.healthCheck).Methods("GET")
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	fmt.Printf("Starting dashboard server on port %d\n", d.port)
	return d.server.ListenAndServe()
}

// Stop stops the dashboard server
func (d *Dashboard) Stop(ctx context.Context) error {
	return d.server.Shutdown(ctx)
}

// authMiddleware adds authentication if enabled
func (d *Dashboard) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	if !d.enableAuth {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != d.username || password != d.password {
			w.Header().Set("WWW-Authenticate", `Basic realm="VPS Tools Dashboard"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// API Handlers

func (d *Dashboard) getDashboardData(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"title":       "VPS Tools Dashboard",
		"timestamp":   time.Now().Unix(),
		"version":     "1.0.0",
		"serverCount": d.getServerCount(),
		"jobCount":    d.getJobCount(),
		"healthStatus": d.getOverallHealthStatus(),
	}

	d.writeJSONResponse(w, data)
}

func (d *Dashboard) getServersData(w http.ResponseWriter, r *http.Request) {
	servers := []map[string]interface{}{
		{
			"id":       "server-1",
			"name":     "web-server-01",
			"host":     "192.168.1.100",
			"status":   "online",
			"cpu":      45.2,
			"memory":   67.8,
			"disk":     23.4,
			"uptime":   "15d 3h 42m",
			"lastSeen": time.Now().Add(-5 * time.Minute).Unix(),
		},
		{
			"id":       "server-2",
			"name":     "db-server-01",
			"host":     "192.168.1.101",
			"status":   "online",
			"cpu":      78.5,
			"memory":   82.1,
			"disk":     45.7,
			"uptime":   "30d 12h 15m",
			"lastSeen": time.Now().Add(-2 * time.Minute).Unix(),
		},
		{
			"id":       "server-3",
			"name":     "cache-server-01",
			"host":     "192.168.1.102",
			"status":   "offline",
			"cpu":      0,
			"memory":   0,
			"disk":     0,
			"uptime":   "0d 0h 0m",
			"lastSeen": time.Now().Add(-1 * time.Hour).Unix(),
		},
	}

	d.writeJSONResponse(w, servers)
}

func (d *Dashboard) getHealthData(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"overall": "warning",
		"checks": []map[string]interface{}{
			{
				"name":      "web-server-01",
				"status":    "healthy",
				"timestamp": time.Now().Add(-5 * time.Minute).Unix(),
				"metrics": map[string]interface{}{
					"cpu":    45.2,
					"memory": 67.8,
					"disk":   23.4,
					"load":   1.2,
				},
			},
			{
				"name":      "db-server-01",
				"status":    "warning",
				"timestamp": time.Now().Add(-2 * time.Minute).Unix(),
				"metrics": map[string]interface{}{
					"cpu":    78.5,
					"memory": 82.1,
					"disk":   45.7,
					"load":   2.8,
				},
			},
			{
				"name":      "cache-server-01",
				"status":    "critical",
				"timestamp": time.Now().Add(-1 * time.Hour).Unix(),
				"metrics": map[string]interface{}{
					"cpu":    0,
					"memory": 0,
					"disk":   0,
					"load":   0,
				},
			},
		},
	}

	d.writeJSONResponse(w, health)
}

func (d *Dashboard) getJobsData(w http.ResponseWriter, r *http.Request) {
	jobs := map[string]interface{}{
		"running": []map[string]interface{}{
			{
				"id":       "job-123",
				"name":     "backup-web-server",
				"server":   "web-server-01",
				"type":     "backup",
				"status":   "running",
				"progress": 65,
				"started":  time.Now().Add(-10 * time.Minute).Unix(),
				"eta":      time.Now().Add(5 * time.Minute).Unix(),
			},
		},
		"completed": []map[string]interface{}{
			{
				"id":       "job-122",
				"name":     "security-scan",
				"server":   "db-server-01",
				"type":     "security",
				"status":   "completed",
				"duration": "2m 15s",
				"started":  time.Now().Add(-30 * time.Minute).Unix(),
				"finished": time.Now().Add(-28 * time.Minute).Unix(),
			},
		},
		"failed": []map[string]interface{}{
			{
				"id":       "job-121",
				"name":     "deploy-update",
				"server":   "cache-server-01",
				"type":     "deploy",
				"status":   "failed",
				"error":    "Connection timeout",
				"started":  time.Now().Add(-2 * time.Hour).Unix(),
				"finished": time.Now().Add(-1 * time.Hour).Unix(),
			},
		},
	}

	d.writeJSONResponse(w, jobs)
}

func (d *Dashboard) getMetricsSummary(w http.ResponseWriter, r *http.Request) {
	summary := map[string]interface{}{
		"uptime":        "15d 3h 42m",
		"totalRequests":  15420,
		"avgResponseTime": "125ms",
		"errorRate":     0.02,
		"activeConnections": 42,
		"throughput": map[string]interface{}{
			"requestsPerSecond": 12.5,
			"bytesPerSecond":    1048576, // 1MB/s
		},
		"resources": map[string]interface{}{
			"cpuUsage":    15.2,
			"memoryUsage": 45.8,
			"diskUsage":   23.4,
		},
	}

	d.writeJSONResponse(w, summary)
}

// Web Interface Handlers

func (d *Dashboard) serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func (d *Dashboard) serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard.html")
}

func (d *Dashboard) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// Helper Methods

func (d *Dashboard) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (d *Dashboard) getServerCount() int {
	// This would typically query the database or internal state
	return 3
}

func (d *Dashboard) getJobCount() int {
	// This would typically query the database or internal state
	return 5
}

func (d *Dashboard) getOverallHealthStatus() string {
	// This would typically calculate based on all server health checks
	return "warning"
}