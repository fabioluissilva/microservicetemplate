package commonconfig

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/spf13/viper"
)

const (
	MESSAGEADAPTERNAME = "messageadapterphotoss3"
)

type Config interface {
	GetVersion() string
	GetLogLevel() string
	GetServiceName() string
	GetApiKey() string
}

type BaseConfig struct {
	Version     string `mapstructure:"VERSION"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	ServiceName string `mapstructure:"SERVICE_NAME"`
	ApiKey      string `mapstructure:"API_KEY" sensitive:"true"`
}

func (c *BaseConfig) GetVersion() string {
	return c.Version
}

func (c *BaseConfig) GetLogLevel() string {
	return c.LogLevel
}

func (c *BaseConfig) GetServiceName() string {
	return c.ServiceName
}

func (c *BaseConfig) GetApiKey() string {
	return c.ApiKey
}

var (
	conf Config
	once sync.Once
)

func setConfig(c Config) {
	conf = c
}

func GetConfig() Config {
	return conf
}

func ReadReleaseNotes() (string, error) {
	releaseNotesPath := "releasenotes.txt"
	commonlogger.GetLogger().Debug(fmt.Sprintf("Reading Release Notes from: %s", releaseNotesPath))
	content, err := os.ReadFile(releaseNotesPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func Initialize(target Config) {
	once.Do(func() {
		// defaults
		viper.SetConfigFile(".env")
		viper.AddConfigPath(".")
		viper.AddConfigPath("..")
		viper.SetConfigType("toml")
		viper.SetDefault("SERVICE_NAME", "servicetemplate")
		viper.SetDefault("LOG_LEVEL", "DEBUG")
		viper.SetDefault("VERSION", "0.0.0")

		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		viper.AutomaticEnv()

		err := viper.ReadInConfig()
		if err != nil {
			// don't use logger here yet!
			fmt.Fprintf(os.Stderr, "[commonconfig] Error loading config file: %s\n", err.Error())
			os.Exit(1)
		}

		err = viper.Unmarshal(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[commonconfig] Error parsing config: %s\n", err.Error())
			os.Exit(1)
		}

		setConfig(target)
		commonlogger.SetLogLevel(conf.GetLogLevel())
		if conf.GetApiKey() == "" {
			commonlogger.GetLogger().Error("API_KEY is required")
			os.Exit(1)
		}
		commonlogger.Debug(fmt.Sprintf("Successfully Loaded configuration for service: %s", conf.GetServiceName()))
	})
}
