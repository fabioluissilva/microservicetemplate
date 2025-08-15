package main

import (
	"encoding/json"
	"net/http"

	"github.com/fabioluissilva/microservicetemplate/commonapi"
	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/fabioluissilva/microservicetemplate/commonscheduler"
)

type ServiceConfig struct {
	commonconfig.BaseConfig `mapstructure:",squash"`
	Test                    string `mapstructure:"TEST"`
}

func (c *ServiceConfig) GetApiKey() string {
	return c.ApiKey
}
func (c *ServiceConfig) GetVersion() string {
	return c.Version
}
func (c *ServiceConfig) GetLogLevel() string {
	return c.LogLevel
}
func (c *ServiceConfig) GetServiceName() string {
	return c.ServiceName
}
func (c *ServiceConfig) GetMetricsPort() int {
	return c.MetricsPort
}
func (c *ServiceConfig) GetPort() int {
	return c.Port
}

func customPingHandler(w http.ResponseWriter, r *http.Request) {

	response := map[string]string{"message": "Custom Ping Handler is working!"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func customScheduledJob() {
	commonlogger.Info("Custom Scheduled Job executed", "service", commonconfig.GetConfig().GetServiceName())
	// You can add more logic here, like sending metrics or logging
}

func main() {
	var config ServiceConfig
	commonconfig.Initialize(&config)
	commonmetrics.InitializeMetrics()
	commonlogger.Info("Main Started")
	// Define a custom scheduled job
	scheduledJobs := []commonscheduler.CronJob{
		{
			Name:     "Custom Scheduled Job",
			CronExpr: "*/1 * * * *",
			Job:      customScheduledJob,
			Tags:     []string{"custom", "scheduled"},
		},
	}
	// you can pass nil if you don't have custom jobs
	commonscheduler.InitScheduler(scheduledJobs)

	// Start the API server with a ping custom handler. Note that this is a separate route from the default ping handler.
	// If you want to override the existing one, just add the same route with a different handler.
	overrides := commonapi.RouteMap{
		"/ping2": customPingHandler,
	}
	done, err := commonapi.StartAPI(&config, overrides)

	if err != nil {
		commonlogger.Error("Error starting API: ", "error", err.Error(), "service", commonconfig.GetConfig().GetServiceName())
		return
	}
	// starts the scheduler

	commonlogger.Info("Successfully started the service: ", "service", commonconfig.GetConfig().GetServiceName())
	// Wait for shutdown to complete
	<-done
	commonlogger.Info("Service shutdown complete", "service", commonconfig.GetConfig().GetServiceName())

}
