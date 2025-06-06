package service

import (
	"fmt"
	"time"

	"road-detector-go/internal/client"
	"road-detector-go/internal/geo"
	"road-detector-go/pkg/models"
	"github.com/sirupsen/logrus"
)

// AnalyzerService сервис для анализа дорожной разметки
type AnalyzerService struct {
	pythonClient *client.PythonAPIClient
	geoCalc      *geo.Calculator
	logger       *logrus.Logger
}

// NewAnalyzerService создает новый сервис анализатора
func NewAnalyzerService(pythonClient *client.PythonAPIClient, geoCalc *geo.Calculator, logger *logrus.Logger) *AnalyzerService {
	return &AnalyzerService{
		pythonClient: pythonClient,
		geoCalc:      geoCalc,
		logger:       logger,
	}
}

// AnalyzeRoadMarking анализирует дорожную разметку
func (s *AnalyzerService) AnalyzeRoadMarking(request models.AnalyzeRequest) (*models.AnalyzeResponse, error) {
	s.logger.Infof("Начинаем анализ дорожной разметки для видео %s", request.VideoFilename)
	
	startTime := time.Now()
	
	// 1. Отправляем запрос в Python API для получения данных нейронной сети
	s.logger.Info("Отправляем видео в Python API для анализа нейронной сетью")
	pythonResp, err := s.pythonClient.AnalyzeVideo(request)
	if err != nil {
		s.logger.Errorf("Ошибка при обращении к Python API: %v", err)
		return &models.AnalyzeResponse{
			Status:  "error",
			Message: fmt.Sprintf("Ошибка анализа нейронной сетью: %v", err),
		}, nil
	}

	if pythonResp.Status != "success" {
		s.logger.Errorf("Python API вернул ошибку: %s", pythonResp.Message)
		return &models.AnalyzeResponse{
			Status:  "error",
			Message: fmt.Sprintf("Ошибка от Python API: %s", pythonResp.Message),
		}, nil
	}

	s.logger.Infof("Получили результаты нейронной сети: %d кадров", len(pythonResp.FrameResults))

	// 2. Выполняем географические вычисления
	return s.processGeographicAnalysis(request, pythonResp, startTime)
}

// processGeographicAnalysis выполняет географический анализ результатов
func (s *AnalyzerService) processGeographicAnalysis(request models.AnalyzeRequest, pythonResp *models.PythonAPIResponse, startTime time.Time) (*models.AnalyzeResponse, error) {
	// Интерполируем координаты для всех кадров
	numFrames := len(pythonResp.FrameResults)
	frameCoords := s.geoCalc.InterpolateCoordinates(request.StartPoint, request.EndPoint, numFrames)
	
	s.logger.Infof("Интерполировали координаты для %d кадров", numFrames)

	// Вычисляем общее расстояние
	totalDistance := s.geoCalc.DistanceMeters(request.StartPoint, request.EndPoint)
	s.logger.Infof("Общее расстояние маршрута: %.2f м", totalDistance)

	// Разбиваем на сегменты и вычисляем покрытие
	segments := s.geoCalc.CalculateSegments(
		request.StartPoint,
		request.EndPoint,
		request.SegmentLength,
		frameCoords,
		pythonResp.FrameResults,
	)

	s.logger.Infof("Разбили маршрут на %d сегментов по %d м", len(segments), request.SegmentLength)

	// Вычисляем общую статистику
	overallStats := s.geoCalc.CalculateOverallStats(segments, numFrames, totalDistance, request.SegmentLength)

	processingTime := time.Since(startTime)
	s.logger.Infof("Анализ завершен за %v. Среднее покрытие: %.1f%%", processingTime, overallStats.AverageCoverage)

	return &models.AnalyzeResponse{
		Status:       "success",
		Message:      "Анализ дорожной разметки успешно завершен",
		OverallStats: overallStats,
		Segments:     segments,
	}, nil
}

// CheckHealth проверяет состояние сервиса и его зависимостей
func (s *AnalyzerService) CheckHealth() (*models.HealthResponse, error) {
	s.logger.Debug("Проверяем состояние сервиса анализатора")

	// Проверяем состояние Python API
	pythonHealth, err := s.pythonClient.CheckHealth()
	if err != nil {
		s.logger.Errorf("Python API недоступен: %v", err)
		return &models.HealthResponse{
			Status:      "unhealthy",
			ModelLoaded: false,
			Version:     "1.0.0",
		}, nil
	}

	// Если Python API здоров, возвращаем его статус
	return pythonHealth, nil
} 