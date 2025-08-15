package main

import (
	"fmt"

	"jmartins.com/microservicetemplate/commonconfig"
	"jmartins.com/microservicetemplate/commonlogger"
	"jmartins.com/microservicetemplate/utilities"
)

type ServiceConfig struct {
	commonconfig.BaseConfig `mapstructure:",squash"`
	S3Bucket                string `mapstructure:"S3_BUCKET"`
	S3Region                string `mapstructure:"S3_REGION"`
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
	commonlogger.Debug("Main Started")
	maskedConfig, _ := utilities.ToMaskedJSON(&config)
	fmt.Println(maskedConfig)
}
