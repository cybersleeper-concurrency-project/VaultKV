package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort              string   `mapstructure:"SERVER_PORT"`
	ServerTimeout           string   `mapstructure:"SERVER_TIMEOUT"`
	LogLevel                string   `mapstructure:"LOG_LEVEL"`
	ClusterNodes            []string `mapstructure:"CLUSTER_NODES"`
	ClusterMaxConcurrency   int      `mapstructure:"CLUSTER_MAX_CONCURRENCY"`
	ClusterReplicas         int      `mapstructure:"CLUSTER_REPLICAS"`
	HttpMaxIdleConns        int      `mapstructure:"HTTP_MAX_IDLE_CONNS"`
	HttpMaxIdleConnsPerHost int      `mapstructure:"HTTP_MAX_IDLE_CONNS_PER_HOST"`
	HttpMaxConnsPerHost     int      `mapstructure:"HTTP_MAX_CONNS_PER_HOST"`
	HttpIdleConnTimeout     string   `mapstructure:"HTTP_IDLE_CONN_TIMEOUT"`
}

type Cluster struct {
}
type HttpClient struct {
}

func LoadConfig(prefix string) (*Config, error) {
	_ = godotenv.Load(".env")

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
