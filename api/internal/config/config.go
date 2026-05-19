package config

import (
	"os"
)

type Config struct {
	Port                 string
	DBUrl                string
	RedisAddr            string
	OTelServiceName      string
	OTelExporterEndpoint string
	SMTPHost             string
	SMTPPort             string
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		DBUrl:                getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ecommerce"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		OTelServiceName:      getEnv("OTEL_SERVICE_NAME", "ecommerce-api"),
		OTelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SMTPHost:             getEnv("SMTP_SERVER", "mailpit"),
		SMTPPort:             getEnv("SMTP_SERVER", "1025"),
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)

	if val == "" {
		return fallback
	}

	return val
}
