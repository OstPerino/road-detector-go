package config

import (
	"os"
	"strconv"
)

// Config структура конфигурации приложения
type Config struct {
	Server struct {
		Port int
		Host string
	}
	PythonAPI struct {
		BaseURL string
		Timeout int // в секундах
	}
	Logging struct {
		Level string
	}
}

// LoadConfig загружает конфигурацию из переменных окружения
func LoadConfig() *Config {
	cfg := &Config{}

	// Конфигурация сервера
	cfg.Server.Port = getEnvInt("SERVER_PORT", 8080)
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")

	// Конфигурация Python API
	cfg.PythonAPI.BaseURL = getEnv("PYTHON_API_BASE_URL", "http://localhost:8000")
	cfg.PythonAPI.Timeout = getEnvInt("PYTHON_API_TIMEOUT_SECONDS", 300) // 5 минут по умолчанию

	// Конфигурация логирования
	cfg.Logging.Level = getEnv("LOG_LEVEL", "info")

	return cfg
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает int значение переменной окружения или возвращает значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
} 