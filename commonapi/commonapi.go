package commonapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/fabioluissilva/microservicetemplate/utilities"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Middleware to validate the API key for incoming requests

// Middleware to check if the X-API-KEY is present and valid according to the configuration
// If the API key is invalid, it returns a 401 Unauthorized response.
func withAPIKey(fn http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey != commonconfig.GetConfig().GetApiKey() {
			commonmetrics.UnauthorizedRequests.Inc()
			http.Error(w, "Invalid API Key", http.StatusUnauthorized)
			return
		}
		fn(w, r)
	}
}

func writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func testHttpMethod(r *http.Request, w *http.ResponseWriter, method string, handler string) bool {
	if r.Method != method {
		commonmetrics.NumberOfErrors.Inc()
		http.Error(*w, fmt.Sprintf(`{"error": "Only %s method is allowed"}`, method), http.StatusMethodNotAllowed)
		commonlogger.Error(fmt.Sprintf("%s: Only %s method is allowed", handler, method), "service", commonconfig.GetConfig().GetServiceName())
		return false
	}
	return true
}

func releaseNotesHandler(w http.ResponseWriter, r *http.Request) {
	if !testHttpMethod(r, &w, http.MethodGet, "releaseNotesHandler") {
		return
	}
	notes, err := utilities.ReadReleaseNotes()
	if err != nil {
		commonmetrics.NumberOfErrors.Inc()
		http.Error(w, `{"error": "Failed to read release notes"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(notes))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if !testHttpMethod(r, &w, http.MethodGet, "pingHandler") {
		return
	}
	message := r.URL.Query().Get("message")
	if message == "" {
		message = "No message provided"
	}
	commonlogger.Debug(fmt.Sprintf("Ping request received: %s", message), "service", commonconfig.GetConfig().GetServiceName())

	response := map[string]string{
		"service":   commonconfig.GetConfig().GetServiceName(),
		"version":   commonconfig.GetConfig().GetVersion(),
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "ok",
		"message":   message,
	}
	commonmetrics.NumberOfPings.Inc()
	writeJSONResponse(w, response)
}

func configHandler(cfg commonconfig.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		commonlogger.Debug("[API] Config request received")
		if r.Method != http.MethodGet {
			commonmetrics.NumberOfErrors.Inc()
			http.Error(w, `{"error": "Only GET method is allowed"}`, http.StatusMethodNotAllowed)
			commonlogger.Error("Only GET method is allowed")
		}
		w.Header().Set("Content-Type", "application/json")
		commonmetrics.NumberOfConfigRequests.Inc()
		maskedJson, err := utilities.ToMaskedJSON(&cfg)
		if err != nil {
			commonmetrics.NumberOfErrors.Inc()
			http.Error(w, `{"error": "Failed to generate config JSON"}`, http.StatusInternalServerError)
			commonlogger.Error("Failed to generate config JSON", "error", err.Error())
			return
		}
		w.Write([]byte(maskedJson))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Only GET method is allowed"}`, http.StatusMethodNotAllowed)
		commonlogger.Error("Only GET method is allowed", "package", "api", "service", commonconfig.GetConfig().GetServiceName())
		return
	}
	writeJSONResponse(w, map[string]string{"status": "ok"})
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Only GET method is allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	writeJSONResponse(w, map[string]string{"status": "alive"})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Only GET method is allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	// TODO: Add readiness check
	writeJSONResponse(w, map[string]string{"status": "ready"})
}

func RegisterRoutes(cfg commonconfig.Config) error {
	commonlogger.Debug("[API] Registering routes")
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/releasenotes", releaseNotesHandler)
	http.HandleFunc("/config", withAPIKey(configHandler(cfg)))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)
	commonlogger.Debug("Routes registered successfully")
	return nil
}

func StartAPI(cfg commonconfig.Config) (chan struct{}, error) {
	done := make(chan struct{})
	commonlogger.Info("Starting Prometheus Metrics Listener on " + strconv.Itoa(commonconfig.GetConfig().GetMetricsPort()))

	// Create servers
	metricsServer := &http.Server{
		Addr:    ":" + strconv.Itoa(commonconfig.GetConfig().GetMetricsPort()),
		Handler: nil,
	}
	apiServer := &http.Server{
		Addr:    ":" + strconv.Itoa(commonconfig.GetConfig().GetPort()),
		Handler: nil,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)

	// Start metrics server in goroutine
	go func() {
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			commonlogger.Error("Metrics server error: " + err.Error())
		}
	}()

	// Register routes
	if err := RegisterRoutes(cfg); err != nil {
		commonmetrics.NumberOfErrors.Inc()
		commonlogger.Error("Error registering routes: " + err.Error())
		return nil, err
	}

	// Start API server in goroutine
	go func() {
		commonlogger.Info("[API] Starting API on port " + strconv.Itoa(commonconfig.GetConfig().GetPort()))
		if err := apiServer.ListenAndServe(); err != http.ErrServerClosed {
			commonlogger.Error("API server error: " + err.Error())
		}
	}()

	// Handle shutdown in a separate goroutine
	go func() {
		sig := <-sigChan
		commonlogger.Info(fmt.Sprintf("Received signal: %v", sig))

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown both servers
		if err := metricsServer.Shutdown(ctx); err != nil {
			commonlogger.Error("Metrics server shutdown error: " + err.Error())
		}
		if err := apiServer.Shutdown(ctx); err != nil {
			commonlogger.Error("API server shutdown error: " + err.Error())
		}

		close(done)
	}()

	return done, nil
}
