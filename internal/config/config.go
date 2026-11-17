package config

import (
	"errors"

	"github.com/Rambudhi/money-transfer/internal/pkg/log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig
	PostgreSQL PostgreSQLConfig
	RabbitMQ   RabbitMQConfig
}

type AppConfig struct {
	Name    string
	Version string
	Mode    string
	Address string
	Port    string
}

type PostgreSQLConfig struct {
	Username           string
	Password           string
	DB                 string
	Host               string
	Schema             string
	Port               int
	MinIdleConnections int
	MaxOpenConnections int
	MaxLifetime        int
	LogMode            bool
}

type RabbitMQConfig struct {
	User     string
	Password string
	Host     string
	Port     int
	URL      string
}

func LoadConfig() (*Config, error) {
	// Load .env into environment variables.
	// Ignore error if .env does not exist.
	_ = godotenv.Load()

	v := viper.New()
	v.AutomaticEnv()

	// Application defaults
	v.SetDefault("APP_ADDRESS", "0.0.0.0")
	v.SetDefault("APP_PORT", "8090")
	v.SetDefault("TZ", "Asia/Jakarta")

	// PostgreSQL defaults
	v.SetDefault("POSTGRESQL_HOST", "localhost")
	v.SetDefault("POSTGRESQL_PORT", 5432)

	// RabbitMQ defaults
	v.SetDefault("RABBITMQ_HOST", "localhost")
	v.SetDefault("RABBITMQ_PORT", 5672)

	cfg := &Config{
		App: AppConfig{
			Name:    v.GetString("APP_NAME"),
			Version: v.GetString("APP_VERSION"),
			Mode:    v.GetString("APP_MODE"),
			Address: v.GetString("APP_ADDRESS"),
			Port:    v.GetString("APP_PORT"),
		},

		PostgreSQL: PostgreSQLConfig{
			Username:           v.GetString("POSTGRESQL_USERNAME"),
			Password:           v.GetString("POSTGRESQL_PASSWORD"),
			DB:                 v.GetString("POSTGRESQL_DB"),
			Host:               v.GetString("POSTGRESQL_HOST"),
			Schema:             v.GetString("POSTGRESQL_SCHEMA"),
			Port:               v.GetInt("POSTGRESQL_PORT"),
			MinIdleConnections: v.GetInt("POSTGRESQL_MIN_IDLE_CONNECTIONS"),
			MaxOpenConnections: v.GetInt("POSTGRESQL_MAX_OPEN_CONNECTIONS"),
			MaxLifetime:        v.GetInt("POSTGRESQL_MAX_LIFETIME"),
			LogMode:            v.GetBool("POSTGRESQL_LOG_MODE"),
		},

		RabbitMQ: RabbitMQConfig{
			User:     v.GetString("RABBITMQ_USER"),
			Password: v.GetString("RABBITMQ_PASSWORD"),
			Host:     v.GetString("RABBITMQ_HOST"),
			Port:     v.GetInt("RABBITMQ_PORT"),
			URL:      v.GetString("RABBITMQ_URL"),
		},
	}

	if cfg.App.Name == "" {
		log.Error("APP_NAME is required")
		return nil, errors.New("APP_NAME is required")
	}

	if cfg.PostgreSQL.DB == "" {
		log.Error("POSTGRESQL_DB is required")
		return nil, errors.New("POSTGRESQL_DB is required")
	}

	return cfg, nil
}
