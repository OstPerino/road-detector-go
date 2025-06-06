package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"road-detector-go/internal/service"
	"road-detector-go/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AnalyzerHandler обработчик для анализа дорожной разметки
type AnalyzerHandler struct {
	analyzerService *service.AnalyzerService
	logger          *logrus.Logger
}

// NewAnalyzerHandler создает новый обработчик
func NewAnalyzerHandler(analyzerService *service.AnalyzerService, logger *logrus.Logger) *AnalyzerHandler {
	return &AnalyzerHandler{
		analyzerService: analyzerService,
		logger:          logger,
	}
}

// AnalyzeRoadMarking обрабатывает запрос на анализ дорожной разметки
// @Summary Анализ дорожной разметки
// @Description Анализирует видео с дорожной разметкой и возвращает статистику по сегментам
// @Tags analysis
// @Accept multipart/form-data
// @Produce json
// @Param video formData file true "Видео файл для анализа"
// @Param startLat formData number true "Широта начальной точки" minimum(-90) maximum(90)
// @Param startLon formData number true "Долгота начальной точки" minimum(-180) maximum(180)
// @Param endLat formData number true "Широта конечной точки" minimum(-90) maximum(90)
// @Param endLon formData number true "Долгота конечной точки" minimum(-180) maximum(180)
// @Param segmentLength formData integer false "Длина сегмента в метрах" default(100) minimum(50) maximum(1000)
// @Success 200 {object} models.AnalyzeResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /analyze [post]
func (h *AnalyzerHandler) AnalyzeRoadMarking(c *gin.Context) {
	h.logger.Info("Получен запрос на анализ дорожной разметки")

	// Парсим multipart form
	err := c.Request.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		h.logger.Errorf("Ошибка парсинга multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ошибка парсинга формы",
		})
		return
	}

	// Получаем видео файл
	videoFile, header, err := c.Request.FormFile("video")
	if err != nil {
		h.logger.Errorf("Ошибка получения видео файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Видео файл обязателен",
		})
		return
	}
	defer videoFile.Close()

	// Читаем содержимое видео файла
	videoData, err := io.ReadAll(videoFile)
	if err != nil {
		h.logger.Errorf("Ошибка чтения видео файла: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка чтения видео файла",
		})
		return
	}

	// Парсим координаты
	startLat, err := parseFloat(c.PostForm("startLat"), "startLat")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startLon, err := parseFloat(c.PostForm("startLon"), "startLon")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endLat, err := parseFloat(c.PostForm("endLat"), "endLat")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endLon, err := parseFloat(c.PostForm("endLon"), "endLon")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Парсим длину сегмента (опциональный параметр)
	segmentLength := 100 // значение по умолчанию
	if segmentLengthStr := c.PostForm("segmentLength"); segmentLengthStr != "" {
		segmentLength, err = strconv.Atoi(segmentLengthStr)
		if err != nil || segmentLength < 50 || segmentLength > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "segmentLength должен быть числом от 50 до 1000",
			})
			return
		}
	}

	// Валидация координат
	if err := validateCoordinates(startLat, startLon, endLat, endLon); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Создаем запрос для сервиса
	request := models.AnalyzeRequest{
		VideoData:     videoData,
		VideoFilename: header.Filename,
		StartPoint: models.Coordinates{
			Lat: startLat,
			Lon: startLon,
		},
		EndPoint: models.Coordinates{
			Lat: endLat,
			Lon: endLon,
		},
		SegmentLength: segmentLength,
	}

	// Вызываем сервис
	response, err := h.analyzerService.AnalyzeRoadMarking(request)
	if err != nil {
		h.logger.Errorf("Ошибка сервиса анализа: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Внутренняя ошибка сервера",
		})
		return
	}

	h.logger.Info("Анализ успешно завершен")
	c.JSON(http.StatusOK, response)
}

// HealthCheck проверяет состояние сервиса
// @Summary Проверка состояния сервиса
// @Description Возвращает информацию о состоянии сервиса и его зависимостей
// @Tags health
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Failure 500 {object} gin.H
// @Router /health [get]
func (h *AnalyzerHandler) HealthCheck(c *gin.Context) {
	h.logger.Debug("Получен запрос проверки здоровья")

	health, err := h.analyzerService.CheckHealth()
	if err != nil {
		h.logger.Errorf("Ошибка проверки здоровья: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка проверки состояния сервиса",
		})
		return
	}

	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// parseFloat парсит строку в float64
func parseFloat(value, fieldName string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("%s обязателен", fieldName)
	}

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s должен быть числом", fieldName)
	}

	return result, nil
}

// validateCoordinates валидирует координаты
func validateCoordinates(startLat, startLon, endLat, endLon float64) error {
	if startLat < -90 || startLat > 90 {
		return fmt.Errorf("startLat должен быть в диапазоне от -90 до 90")
	}
	if startLon < -180 || startLon > 180 {
		return fmt.Errorf("startLon должен быть в диапазоне от -180 до 180")
	}
	if endLat < -90 || endLat > 90 {
		return fmt.Errorf("endLat должен быть в диапазоне от -90 до 90")
	}
	if endLon < -180 || endLon > 180 {
		return fmt.Errorf("endLon должен быть в диапазоне от -180 до 180")
	}

	return nil
} 