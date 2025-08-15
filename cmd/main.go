package main

import (
	"fmt"

	"github.com/fabioluissilva/microservicetemplate/commonapi"
	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/commonmetrics"
	"github.com/fabioluissilva/microservicetemplate/utilities"
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

func main() {
	var config ServiceConfig
	commonconfig.Initialize(&config)
	commonmetrics.InitializeMetrics()
	commonlogger.Info("Main Started")
	maskedConfig, _ := utilities.ToMaskedJSON(&config)
	releasenotes, _ := utilities.ReadReleaseNotes()
	fmt.Println(releasenotes)
	fmt.Println(maskedConfig)
	// Start the API server
	done, err := commonapi.StartAPI(&config)
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
