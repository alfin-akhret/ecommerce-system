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
	EmailDefaultSender   string
	RabbitMQHost         string
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
		EmailDefaultSender:   getEnv("EMAIL_DEFAULT_SENDER", "noreply@testcommerce.com"),
		RabbitMQHost:         getEnv("RABBITMQ_HOST", "amqp://guest:guest@rabbitmq:5672"),
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)

	if val == "" {
		return fallback
	}

	return val
}
