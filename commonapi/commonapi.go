package commonapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/fabioluissilva/microservicetemplate/utilities"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouteMap is a mapping of route paths to their handler functions
type RouteMap map[string]http.HandlerFunc

func defaultRoutes(cfg commonconfig.Config) RouteMap {

	return RouteMap{
		"/ping":         pingHandler,                    // no cfg needed
		"/config":       withAPIKey(configHandler(cfg)), // needs cfg
		"/releasenotes": releaseNotesHandler,
		"/metrics":      promhttp.Handler().ServeHTTP,
		"/health":       healthHandler,
		"/liveness":     livenessHandler,
		"/readiness":    readinessHandler,
	}
}

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

func RegisterRoutes(cfg commonconfig.Config, overrides RouteMap) error {
	defaults := defaultRoutes(cfg)

	// Log override actions
	for path := range overrides {
		if _, exists := defaults[path]; exists {
			commonlogger.Debug(fmt.Sprintf("Overriding default route: %s", path))
		} else {
			commonlogger.Debug(fmt.Sprintf("Adding custom route: %s", path))
		}
	}

	// Merge: overrides replace or add
	for path, handler := range overrides {
		defaults[path] = handler
	}

	// Register all routes
	for path, handler := range defaults {
		http.HandleFunc(path, handler)
	}

	commonlogger.Info("Routes registered:")
	for path := range defaults {
		commonlogger.Info("  " + path)
	}

	return nil
}

func StartAPI(cfg commonconfig.Config, overrides RouteMap) (chan struct{}, error) {
	done := make(chan struct{})
	commonlogger.Info("Starting Prometheus Metrics Listener on " + strconv.Itoa(cfg.GetMetricsPort()))

	// Create servers
	metricsServer := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.GetMetricsPort()),
		Handler: nil,
	}
	apiServer := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.GetPort()),
		Handler: nil,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)

	// Start metrics server
	go func() {
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			commonlogger.Error("Metrics server error: " + err.Error())
		}
	}()

	// ✅ Apply overrides if provided
	finalRoutes := defaultRoutes(cfg)
	for path, handler := range overrides {
		commonlogger.Debug(fmt.Sprintf("Overriding/adding route: %s", path))
		finalRoutes[path] = handler
	}

	// ✅ Register all routes
	for path, handler := range finalRoutes {
		handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
		commonlogger.Info("Registered route:", "path", path, "handler", handlerName)
		http.HandleFunc(path, handler)
	}

	// Start API server
	go func() {
		commonlogger.Info("[API] Starting API on port " + strconv.Itoa(cfg.GetPort()))
		if err := apiServer.ListenAndServe(); err != http.ErrServerClosed {
			commonlogger.Error("API server error: " + err.Error())
		}
	}()

	// Graceful shutdown
	go func() {
		sig := <-sigChan
		commonlogger.Info(fmt.Sprintf("Received signal: %v", sig))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

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
