package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fabioluissilva/microservicetemplate/commonapi"
	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/fabioluissilva/microservicetemplate/commonmqengine"
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

func customPingHandlerWithoutAPIKey(w http.ResponseWriter, r *http.Request) {

	response := map[string]string{"message": "Custom Ping Handler Without API Key is working!"}
	w.Header().Set("Content-Type", "application/json")
	commonlogger.Info("Custom Ping Without API KEY Handler called")
	json.NewEncoder(w).Encode(response)
}

func customPingHandlerWithAPIKey(w http.ResponseWriter, r *http.Request) {

	response := map[string]string{"message": "Custom Ping Handler WITH API Key is working!"}
	w.Header().Set("Content-Type", "application/json")
	commonlogger.Info("Custom Ping with API KEY Handler called")
	json.NewEncoder(w).Encode(response)
}

func customScheduledJob() {
	commonlogger.Debug("Custom Scheduled Job executed")
	// You can add more logic here, like sending metrics or logging
}

func main() {
	var config ServiceConfig
	commonconfig.Initialize(&config)
	commonlogger.SetServiceName(config.GetServiceName())
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
	// set RabbitMQ configuration
	mqcfg := commonmqengine.NewMQConfiguration(
		commonmqengine.WithCredentials("proxmox", "proxmox"),
		commonmqengine.WithHost("rabbitmq.thothnet.local"),
		commonmqengine.WithPort(5672),
		commonmqengine.WithVHost("/"),
		commonmqengine.WithQueues(
			commonmqengine.NewQueue("orders",
				commonmqengine.WithExchange(""),
				commonmqengine.WithRoutingKey(""),
				commonmqengine.WithDurable(true),
			),
			commonmqengine.NewQueue("audit",
				commonmqengine.WithAutoDelete(true),
				commonmqengine.WithDurable(true),
			),
		),
	)
	commonmqengine.InitMQEngine(context.Background(), *mqcfg)

	// Start the API server with a ping custom handler. Note that this is a separate route from the default ping handler.
	// If you want to override the existing one, just add the same route with a different handler.
	// commonapi exports a WithAPIKey middleware that can be used to protect routes.
	overrides := commonapi.RouteMap{
		"/ping2": customPingHandlerWithoutAPIKey,
		"/ping3": commonapi.WithAPIKey(customPingHandlerWithAPIKey),
	}
	done, err := commonapi.StartAPI(&config, overrides)

	if err != nil {
		commonlogger.Error("Error starting API: ", "error", err.Error())
		return
	}
	// starts the scheduler

	commonlogger.Info("Successfully started the service: ")
	// Wait for shutdown to complete
	<-done
	commonlogger.Info("Service shutdown complete")

}
