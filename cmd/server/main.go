package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"road-detector-go/internal/database"
	"road-detector-go/internal/handler"
	"road-detector-go/internal/repository"
	"road-detector-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Инициализируем логгер
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.Info("Запуск Road Detector API Server")

	// Получаем конфигурацию из переменных окружения
	config := getConfig()

	// Инициализируем базу данных
	logger.Info("Подключение к базе данных...")
	if err := database.Connect(); err != nil {
		logger.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	// Выполняем миграции
	logger.Info("Выполнение миграций базы данных...")
	if err := database.Migrate(); err != nil {
		logger.Fatalf("Ошибка выполнения миграций: %v", err)
	}

	// Проверяем здоровье базы данных
	if err := database.HealthCheck(); err != nil {
		logger.Fatalf("База данных недоступна: %v", err)
	}

	logger.Info("База данных успешно подключена и готова к работе")

	// Создаем папку для статических файлов
	staticDir := filepath.Join(".", "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		logger.Fatalf("Ошибка создания папки для статических файлов: %v", err)
	}

	// Инициализируем репозитории
	routeRepo := repository.NewRouteRepository(database.DB)

	// Инициализируем сервисы
	routeService := service.NewRouteService(routeRepo, logger, staticDir)
	analyzerService := service.NewAnalyzerService(config.PythonServiceURL, logger, routeService)

	// Инициализируем обработчики
	routeHandler := handler.NewRouteHandler(analyzerService, routeService, logger)

	// Настраиваем Gin router
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Добавляем middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Обслуживание статических файлов
	router.Static("/static", staticDir)

	// Регистрируем маршруты
	routeHandler.RegisterRoutes(router)

	// Добавляем базовый маршрут для проверки
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Road Detector API Server",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// Запускаем сервер
	serverAddr := fmt.Sprintf(":%s", config.Port)
	logger.Infof("Сервер запущен на порту %s", config.Port)
	logger.Infof("API доступно по адресу: http://localhost:%s/api/v1", config.Port)

	if err := router.Run(serverAddr); err != nil {
		logger.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// Config содержит конфигурацию приложения
type Config struct {
	Port             string
	PythonServiceURL string
	Environment      string
}

// getConfig получает конфигурацию из переменных окружения
func getConfig() *Config {
	return &Config{
		Port:             getEnv("SERVER_PORT", "8080"),
		PythonServiceURL: getEnv("PYTHON_API_BASE_URL", "http://localhost:8000"),
		Environment:      getEnv("ENVIRONMENT", "development"),
	}
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// corsMiddleware добавляет заголовки CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
