package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"road-detector-go/internal/client"
	"road-detector-go/internal/config"
	"road-detector-go/internal/geo"
	"road-detector-go/internal/handler"
	"road-detector-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Настраиваем логгер
	logger := setupLogger(cfg.Logging.Level)
	logger.Info("Запускаем сервис анализа дорожной разметки")

	// Инициализируем компоненты
	geoCalc := geo.NewCalculator()
	
	pythonClient := client.NewPythonAPIClient(
		cfg.PythonAPI.BaseURL,
		time.Duration(cfg.PythonAPI.Timeout)*time.Second,
		logger,
	)
	
	analyzerService := service.NewAnalyzerService(pythonClient, geoCalc, logger)
	analyzerHandler := handler.NewAnalyzerHandler(analyzerService, logger)

	// Настраиваем роутер
	router := setupRouter(analyzerHandler, cfg.Logging.Level == "debug")

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		logger.Infof("Сервер запущен на порту %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ждем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Получен сигнал завершения, останавливаем сервер...")

	// Даем серверу 30 секунд на graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Ошибка при остановке сервера: %v", err)
	} else {
		logger.Info("Сервер успешно остановлен")
	}
}

// setupLogger настраивает логгер
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	
	// Устанавливаем уровень логирования
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Устанавливаем формат JSON для продакшена
	if level != "debug" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	return logger
}

// setupRouter настраивает маршруты
func setupRouter(analyzerHandler *handler.AnalyzerHandler, debug bool) *gin.Engine {
	// Устанавливаем режим Gin
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/analyze", analyzerHandler.AnalyzeRoadMarking)
		api.GET("/health", analyzerHandler.HealthCheck)
	}

	// Health check на корневом пути
	router.GET("/health", analyzerHandler.HealthCheck)

	return router
}

// corsMiddleware настраивает CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
} 