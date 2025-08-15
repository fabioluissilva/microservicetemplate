package commonconfig

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/spf13/viper"
)

type Config interface {
	GetVersion() string
	GetLogLevel() string
	GetServiceName() string
	GetApiKey() string
	GetMetricsPort() int
	GetPort() int
	GetHeartBeatDebug() bool
	GetHeartBeatCron() string
}

type BaseConfig struct {
	Version        string `mapstructure:"VERSION"`
	LogLevel       string `mapstructure:"LOG_LEVEL"`
	ServiceName    string `mapstructure:"SERVICE_NAME"`
	ApiKey         string `mapstructure:"API_KEY" sensitive:"true"`
	MetricsPort    int    `mapstructure:"METRICS_PORT"`
	Port           int    `mapstructure:"PORT"`
	HeartBeatDebug bool   `mapstructure:"HEARTBEAT_DEBUG"`
	HeartBeatCron  string `mapstructure:"HEARTBEAT_CRON"`
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
func (c *BaseConfig) GetMetricsPort() int {
	return c.MetricsPort
}
func (c *BaseConfig) GetPort() int {
	return c.Port
}
func (c *BaseConfig) GetHeartBeatDebug() bool {
	return c.HeartBeatDebug
}

func (c *BaseConfig) GetHeartBeatCron() string {
	return c.HeartBeatCron
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

func Initialize(target Config) {
	once.Do(func() {
		// defaults
		viper.SetConfigFile(".env")
		viper.AddConfigPath(".")
		viper.AddConfigPath("..")
		viper.SetConfigType("toml")
		viper.SetDefault("VERSION", "0.0.0")
		viper.SetDefault("SERVICE_NAME", "servicetemplate")
		viper.SetDefault("LOG_LEVEL", "INFO")
		viper.SetDefault("METRICS_PORT", 9091)
		viper.SetDefault("PORT", 8001)
		viper.SetDefault("HEARTBEAT_DEBUG", false)
		viper.SetDefault("HEARTBEAT_CRON", "*/1 * * * *")
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
			commonlogger.GetLogger().Error("API_KEY is required", "service", conf.GetServiceName())
			os.Exit(1)
		}
		commonlogger.Debug("Successfully Loaded configuration", "service", conf.GetServiceName())
	})
}
