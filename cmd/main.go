package main

import (
	"fmt"

	"github.com/fabioluissilva/microservicetemplate/commonconfig"
	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/fabioluissilva/microservicetemplate/utilities"
)

type ServiceConfig struct {
	commonconfig.BaseConfig `mapstructure:",squash"`
	Test                    string `mapstructure:"TEST"`
}

func (c ServiceConfig) GetApiKey() string {
	return c.ApiKey
}
func (c ServiceConfig) GetVersion() string {
	return c.Version
}
func (c ServiceConfig) GetLogLevel() string {
	return c.LogLevel
}
func (c ServiceConfig) GetServiceName() string {
	return c.ServiceName
}

func main() {
	var config ServiceConfig
	commonconfig.Initialize(&config)
	commonlogger.Info("Main Started")
	maskedConfig, _ := utilities.ToMaskedJSON(&config)
	releasenotes, _ := utilities.ReadReleaseNotes()
	fmt.Println(releasenotes)
	fmt.Println(maskedConfig)
}
