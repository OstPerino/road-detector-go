package handler

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"road-detector-go/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RouteHandler обрабатывает HTTP запросы для работы с маршрутами
type RouteHandler struct {
	analyzerService *service.AnalyzerService
	routeService    *service.RouteService
	logger          *logrus.Logger
}

// NewRouteHandler создает новый экземпляр RouteHandler
func NewRouteHandler(analyzerService *service.AnalyzerService, routeService *service.RouteService, logger *logrus.Logger) *RouteHandler {
	return &RouteHandler{
		analyzerService: analyzerService,
		routeService:    routeService,
		logger:          logger,
	}
}

// RegisterRoutes регистрирует маршруты API
func (h *RouteHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/analyze", h.AnalyzeRoadMarking)
		api.GET("/routes", h.ListRoutes)
		api.GET("/routes/:id", h.GetRoute)
		api.DELETE("/routes/:id", h.DeleteRoute)
		api.GET("/routes/area", h.GetRoutesByArea)
		api.GET("/health", h.CheckHealth)
		api.GET("/routes/:id/video", h.GetRouteVideo)
	}
}

// AnalyzeRoadMarking обрабатывает запрос на анализ дорожной разметки
func (h *RouteHandler) AnalyzeRoadMarking(c *gin.Context) {
	h.logger.Info("Получен запрос на анализ дорожной разметки")

	// Парсим multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		h.logger.Errorf("Ошибка парсинга multipart form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка парсинга формы"})
		return
	}

	// Получаем параметры координат (поддерживаем разные форматы)
	startLatStr := getFormValue(c, []string{"start_lat", "startLat"})
	startLonStr := getFormValue(c, []string{"start_lon", "startLon"})
	endLatStr := getFormValue(c, []string{"end_lat", "endLat"})
	endLonStr := getFormValue(c, []string{"end_lon", "endLon"})
	segmentLengthStr := getFormValue(c, []string{"segment_length", "segment_length_m", "segmentLength"})
	routeID := getFormValue(c, []string{"route_id", "routeId"}) // Опциональный параметр

	// Проверяем обязательные параметры
	if startLatStr == "" || startLonStr == "" || endLatStr == "" || endLonStr == "" || segmentLengthStr == "" {
		h.logger.Error("Отсутствуют обязательные параметры")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Отсутствуют обязательные параметры: start_lat (или startLat), start_lon (или startLon), end_lat (или endLat), end_lon (или endLon), segment_length (или segment_length_m, segmentLength)",
		})
		return
	}

	// Парсим координаты
	startLat, err := strconv.ParseFloat(startLatStr, 64)
	if err != nil {
		h.logger.Errorf("Ошибка парсинга start_lat: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат start_lat"})
		return
	}

	startLon, err := strconv.ParseFloat(startLonStr, 64)
	if err != nil {
		h.logger.Errorf("Ошибка парсинга start_lon: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат start_lon"})
		return
	}

	endLat, err := strconv.ParseFloat(endLatStr, 64)
	if err != nil {
		h.logger.Errorf("Ошибка парсинга end_lat: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат end_lat"})
		return
	}

	endLon, err := strconv.ParseFloat(endLonStr, 64)
	if err != nil {
		h.logger.Errorf("Ошибка парсинга end_lon: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат end_lon"})
		return
	}

	segmentLength, err := strconv.ParseFloat(segmentLengthStr, 64)
	if err != nil {
		h.logger.Errorf("Ошибка парсинга segment_length: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат segment_length"})
		return
	}

	// Получаем видео файл
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		h.logger.Errorf("Ошибка получения видео файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Видео файл обязателен"})
		return
	}
	defer file.Close()

	// Читаем весь видео файл в буфер для повторного использования
	videoData, err := io.ReadAll(file)
	if err != nil {
		h.logger.Errorf("Ошибка чтения видео файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка чтения видео файла"})
		return
	}
	h.logger.Infof("Прочитано %d байт видео данных из файла %s", len(videoData), header.Filename)

	// Создаем reader из буфера для передачи в сервис анализа
	videoReader := bytes.NewReader(videoData)

	// Вызываем сервис анализа
	result, err := h.analyzerService.AnalyzeRoadMarking(
		startLat, startLon, endLat, endLon,
		segmentLength, videoReader, header.Filename, routeID,
	)
	if err != nil {
		h.logger.Errorf("Ошибка анализа: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка анализа дорожной разметки"})
		return
	}

	h.logger.Info("Анализ дорожной разметки завершен успешно")
	c.JSON(http.StatusOK, result)
}

// getFormValue получает значение из формы, пробуя разные варианты ключей
func getFormValue(c *gin.Context, keys []string) string {
	for _, key := range keys {
		if value := c.PostForm(key); value != "" {
			return value
		}
	}
	return ""
}

// ListRoutes возвращает список маршрутов с пагинацией
func (h *RouteHandler) ListRoutes(c *gin.Context) {
	h.logger.Info("Получен запрос на получение списка маршрутов")

	// Получаем параметры пагинации
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	// Получаем маршруты
	routes, total, err := h.routeService.ListRoutes(page, size)
	if err != nil {
		h.logger.Errorf("Ошибка получения списка маршрутов: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения списка маршрутов"})
		return
	}

	response := service.ListRoutesResponse{
		Routes: routes,
		Total:  total,
		Page:   page,
		Size:   size,
	}

	h.logger.Infof("Возвращено %d маршрутов из %d", len(routes), total)
	c.JSON(http.StatusOK, response)
}

// GetRoute возвращает маршрут по ID
func (h *RouteHandler) GetRoute(c *gin.Context) {
	routeID := c.Param("id")
	h.logger.Infof("Получен запрос на получение маршрута с ID: %s", routeID)

	route, err := h.routeService.GetRouteByID(routeID)
	if err != nil {
		h.logger.Errorf("Ошибка получения маршрута: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Маршрут не найден"})
		return
	}

	h.logger.Info("Маршрут найден и возвращен")
	c.JSON(http.StatusOK, route)
}

// DeleteRoute удаляет маршрут по ID
func (h *RouteHandler) DeleteRoute(c *gin.Context) {
	routeID := c.Param("id")
	h.logger.Infof("Получен запрос на удаление маршрута с ID: %s", routeID)

	err := h.routeService.DeleteRoute(routeID)
	if err != nil {
		h.logger.Errorf("Ошибка удаления маршрута: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления маршрута"})
		return
	}

	h.logger.Info("Маршрут успешно удален")
	c.JSON(http.StatusOK, gin.H{"message": "Маршрут успешно удален"})
}

// GetRoutesByArea возвращает маршруты в указанной области
func (h *RouteHandler) GetRoutesByArea(c *gin.Context) {
	h.logger.Info("Получен запрос на получение маршрутов по области")

	// Получаем параметры области
	neLat := c.Query("ne_lat")
	neLon := c.Query("ne_lon")
	swLat := c.Query("sw_lat")
	swLon := c.Query("sw_lon")

	if neLat == "" || neLon == "" || swLat == "" || swLon == "" {
		h.logger.Error("Отсутствуют параметры области")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Отсутствуют обязательные параметры: ne_lat, ne_lon, sw_lat, sw_lon",
		})
		return
	}

	// Парсим координаты
	neLatFloat, err := strconv.ParseFloat(neLat, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ne_lat"})
		return
	}

	neLonFloat, err := strconv.ParseFloat(neLon, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат ne_lon"})
		return
	}

	swLatFloat, err := strconv.ParseFloat(swLat, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат sw_lat"})
		return
	}

	swLonFloat, err := strconv.ParseFloat(swLon, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат sw_lon"})
		return
	}

	// Получаем маршруты в области
	routes, err := h.routeService.GetRoutesByArea(neLatFloat, neLonFloat, swLatFloat, swLonFloat)
	if err != nil {
		h.logger.Errorf("Ошибка получения маршрутов по области: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения маршрутов"})
		return
	}

	response := service.GetSegmentsByAreaResponse{
		Routes: routes,
		Total:  len(routes),
	}

	h.logger.Infof("Найдено %d маршрутов в указанной области", len(routes))
	c.JSON(http.StatusOK, response)
}

// CheckHealth проверяет состояние сервиса
func (h *RouteHandler) CheckHealth(c *gin.Context) {
	h.logger.Info("Получен запрос проверки здоровья сервиса")

	// Проверяем состояние анализатора
	err := h.analyzerService.CheckHealth()
	if err != nil {
		h.logger.Errorf("Сервис анализа недоступен: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Сервис анализа недоступен",
		})
		return
	}

	h.logger.Info("Сервис работает нормально")
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Сервис работает нормально",
	})
}

// GetRouteVideo возвращает видео для конкретного маршрута
func (h *RouteHandler) GetRouteVideo(c *gin.Context) {
	routeID := c.Param("id")
	if routeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "route ID is required"})
		return
	}

	route, err := h.routeService.GetRouteByID(routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	if route.VideoPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found for this route"})
		return
	}

	// Отправляем видео файл
	c.File(route.VideoPath)
}
