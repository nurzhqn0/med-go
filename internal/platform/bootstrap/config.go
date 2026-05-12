package bootstrap

import (
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DoctorAddress          string
	AppointmentAddress     string
	DoctorServiceTarget    string
	DatabaseURL            string
	DoctorDatabaseURL      string
	AppointmentDatabaseURL string
	NATSURL                string
	RedisURL               string
	CacheTTL               time.Duration
	RateLimitRPM           int
	GatewayURL             string
	GatewayPort            string
	WorkerPoolSize         int
	JobMaxRetries          int
	JobBackoffs            []time.Duration
}

func LoadConfig() Config {
	databaseURL := GetEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/med_go?sslmode=disable")

	return Config{
		DoctorAddress:          serviceAddress("DOCTOR_SERVICE_ADDR", ":8081"),
		AppointmentAddress:     serviceAddress("APPOINTMENT_SERVICE_ADDR", ":8082"),
		DoctorServiceTarget:    GetEnv("DOCTOR_SERVICE_GRPC_TARGET", "127.0.0.1:8081"),
		DatabaseURL:            databaseURL,
		DoctorDatabaseURL:      GetEnv("DOCTOR_DATABASE_URL", databaseURL),
		AppointmentDatabaseURL: GetEnv("APPOINTMENT_DATABASE_URL", databaseURL),
		NATSURL:                GetEnv("NATS_URL", "nats://localhost:4222"),
		RedisURL:               GetEnv("REDIS_URL", "redis://localhost:6379"),
		CacheTTL:               time.Duration(getEnvInt("CACHE_TTL_SECONDS", 60)) * time.Second,
		RateLimitRPM:           getEnvInt("RATE_LIMIT_RPM", 100),
		GatewayURL:             GetEnv("GATEWAY_URL", "http://localhost:8080"),
		GatewayPort:            GetEnv("GATEWAY_PORT", "8080"),
		WorkerPoolSize:         getEnvInt("WORKER_POOL_SIZE", 3),
		JobMaxRetries:          getEnvInt("JOB_MAX_RETRIES", 3),
		JobBackoffs:            getEnvDurations("JOB_BACKOFF_SECONDS", []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}),
	}
}

func serviceAddress(primaryKey, fallback string) string {
	if value := GetEnv(primaryKey, ""); value != "" {
		return value
	}

	if port := GetEnv("GRPC_PORT", ""); port != "" {
		return ":" + strings.TrimPrefix(port, ":")
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := GetEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func getEnvDurations(key string, fallback []time.Duration) []time.Duration {
	value := GetEnv(key, "")
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	durations := make([]time.Duration, 0, len(parts))
	for _, part := range parts {
		seconds, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || seconds <= 0 {
			return fallback
		}
		durations = append(durations, time.Duration(seconds)*time.Second)
	}
	if len(durations) == 0 {
		return fallback
	}

	return durations
}
