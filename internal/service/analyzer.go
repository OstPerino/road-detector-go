package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"archive/zip"

	"github.com/sirupsen/logrus"
)

// AnalyzerService сервис для анализа дорожной разметки
type AnalyzerService struct {
	pythonServiceURL string
	logger           *logrus.Logger
	client           *http.Client
	routeService     *RouteService
}

// NewAnalyzerService создает новый сервис анализатора
func NewAnalyzerService(pythonServiceURL string, logger *logrus.Logger, routeService *RouteService) *AnalyzerService {
	return &AnalyzerService{
		pythonServiceURL: pythonServiceURL,
		logger:           logger,
		client: &http.Client{
			Timeout: 300 * time.Second, // Увеличиваем таймаут для обработки видео
		},
		routeService: routeService,
	}
}

// AnalyzeRoadMarking анализирует дорожное покрытие
func (s *AnalyzerService) AnalyzeRoadMarking(
	startLat, startLon, endLat, endLon, segmentLength float64,
	videoFile io.Reader,
	videoFilename string,
	routeID string, // Добавлен параметр routeID
) (*AnalysisResult, error) {
	s.logger.Infof("Начинаем анализ дорожного покрытия для маршрута %s", routeID)
	s.logger.Infof("Координаты: start(%.6f, %.6f), end(%.6f, %.6f), длина сегмента: %.2f",
		startLat, startLon, endLat, endLon, segmentLength)

	// Генерируем ID маршрута если не передан
	if routeID == "" {
		routeID = s.routeService.GenerateRouteID()
		s.logger.Infof("Сгенерирован новый ID маршрута: %s", routeID)
	}

	// Создаем multipart форму для отправки файла и данных
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавляем координаты в форму - используем названия как ожидает Python сервис /analyze-road-marking
	writer.WriteField("lat1", fmt.Sprintf("%.6f", startLat))
	writer.WriteField("lon1", fmt.Sprintf("%.6f", startLon))
	writer.WriteField("lat2", fmt.Sprintf("%.6f", endLat))
	writer.WriteField("lon2", fmt.Sprintf("%.6f", endLon))
	writer.WriteField("segment_length_m", fmt.Sprintf("%.0f", segmentLength))

	// Читаем видео файл в буфер для дальнейшего использования
	var videoData []byte
	if videoFile != nil {
		var err error
		videoData, err = io.ReadAll(videoFile)
		if err != nil {
			s.logger.Errorf("Ошибка чтения видео файла: %v", err)
			return nil, fmt.Errorf("failed to read video file: %w", err)
		}

		// Добавляем видео файл в форму
		part, err := writer.CreateFormFile("video", videoFilename)
		if err != nil {
			s.logger.Errorf("Ошибка создания form file: %v", err)
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		// Записываем в форму
		_, err = part.Write(videoData)
		if err != nil {
			s.logger.Errorf("Ошибка записи видео данных: %v", err)
			return nil, fmt.Errorf("failed to write video data: %w", err)
		}
	}

	writer.Close()

	// Отправляем запрос к Python сервису используя endpoint который возвращает ZIP
	url := fmt.Sprintf("%s/analyze-road-marking", s.pythonServiceURL)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		s.logger.Errorf("Ошибка создания HTTP запроса: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	s.logger.Infof("Отправляем запрос к Python сервису: %s", url)
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorf("Ошибка отправки запроса: %v", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.logger.Errorf("Python сервис вернул ошибку %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("python service returned error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Читаем ZIP архив
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Errorf("Ошибка чтения ZIP архива: %v", err)
		return nil, fmt.Errorf("failed to read ZIP archive: %w", err)
	}

	s.logger.Infof("Получен ZIP архив размером %d байт", len(zipData))

	// Обрабатываем ZIP архив
	result, annotatedVideoData, err := s.processZipArchive(zipData, startLat, startLon, endLat, endLon, segmentLength)
	if err != nil {
		s.logger.Errorf("Ошибка обработки ZIP архива: %v", err)
		return nil, fmt.Errorf("failed to process ZIP archive: %w", err)
	}

	// Сохраняем аннотированное видео
	if annotatedVideoData != nil && len(annotatedVideoData) > 0 {
		annotatedVideoPath := fmt.Sprintf("static/annotated_%s_%s", routeID, videoFilename)
		err = s.saveAnnotatedVideo(annotatedVideoPath, annotatedVideoData)
		if err != nil {
			s.logger.Errorf("Ошибка сохранения аннотированного видео: %v", err)
		} else {
			s.logger.Infof("Аннотированное видео сохранено: %s", annotatedVideoPath)
		}
	}

	s.logger.Infof("Анализ завершен. Найдено %d сегментов, средний покрытие: %.2f%%",
		result.OverallStats.TotalSegments, result.OverallStats.AverageCoverage)

	// Сохраняем результат в базе данных
	if s.routeService != nil && len(videoData) > 0 {
		s.logger.Infof("Начинаем сохранение маршрута в БД. Размер видео: %d байт", len(videoData))
		videoReader := bytes.NewReader(videoData)
		err = s.routeService.SaveRoute(routeID, videoFilename, videoReader, result)
		if err != nil {
			s.logger.Errorf("Ошибка сохранения маршрута в БД: %v", err)
			// Не возвращаем ошибку, так как анализ прошел успешно
			s.logger.Warnf("Анализ выполнен, но данные не сохранены в БД")
		} else {
			s.logger.Infof("Маршрут %s успешно сохранен в базе данных", routeID)
		}
	} else {
		if s.routeService == nil {
			s.logger.Warn("RouteService не инициализирован - сохранение в БД пропущено")
		}
		if len(videoData) == 0 {
			s.logger.Warn("Видео данных нет - сохранение в БД пропущено")
		}
	}

	return result, nil
}

// CheckHealth проверяет состояние сервиса
func (s *AnalyzerService) CheckHealth() error {
	s.logger.Info("Проверяем состояние Python сервиса")

	url := fmt.Sprintf("%s/health", s.pythonServiceURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorf("Python сервис недоступен: %v", err)
		return fmt.Errorf("python service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Errorf("Python сервис вернул статус %d", resp.StatusCode)
		return fmt.Errorf("python service returned status %d", resp.StatusCode)
	}

	s.logger.Info("Python сервис работает нормально")
	return nil
}

// calculateDistance вычисляет расстояние между двумя точками в метрах
func (s *AnalyzerService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Формула Haversine для точного вычисления расстояния
	const earthRadius = 6371000 // метры

	// Конвертируем градусы в радианы
	lat1Rad := lat1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	deltaLat := (lat2 - lat1) * (math.Pi / 180)
	deltaLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// generateSegments генерирует промежуточные точки между начальной и конечной координатами
func (s *AnalyzerService) generateSegments(startLat, startLon, endLat, endLon, segmentLength float64) []Coordinates {
	distance := s.calculateDistance(startLat, startLon, endLat, endLon)
	numSegments := int(distance/segmentLength) + 1

	if numSegments < 2 {
		numSegments = 2
	}

	segments := make([]Coordinates, numSegments)

	for i := 0; i < numSegments; i++ {
		ratio := float64(i) / float64(numSegments-1)
		lat := startLat + (endLat-startLat)*ratio
		lon := startLon + (endLon-startLon)*ratio

		segments[i] = Coordinates{
			Lat: lat,
			Lon: lon,
		}
	}

	s.logger.Infof("Сгенерировано %d сегментов для расстояния %.2f м", numSegments, distance)
	return segments
}

// parseCoordinate парсит строку координаты в float64
func parseCoordinate(coord string) (float64, error) {
	coord = strings.TrimSpace(coord)
	return strconv.ParseFloat(coord, 64)
}

// processZipArchive обрабатывает ZIP архив с результатами анализа и аннотированным видео
func (s *AnalyzerService) processZipArchive(zipData []byte, startLat, startLon, endLat, endLon, segmentLength float64) (*AnalysisResult, []byte, error) {
	// Создаем reader для ZIP архива
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ZIP reader: %w", err)
	}

	var analysisData []byte
	var videoData []byte

	// Обрабатываем файлы в архиве
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file %s: %w", file.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}

		if file.Name == "analysis_results.json" {
			analysisData = data
			s.logger.Infof("Найден JSON файл с результатами: %d байт", len(data))
		} else if strings.HasPrefix(file.Name, "annotated_") && strings.HasSuffix(file.Name, ".mp4") {
			videoData = data
			s.logger.Infof("Найдено аннотированное видео: %s, размер: %d байт", file.Name, len(data))
		}
	}

	if analysisData == nil {
		return nil, nil, fmt.Errorf("analysis_results.json not found in ZIP archive")
	}

	// Парсим результаты анализа
	var pythonResults struct {
		Status       string `json:"status"`
		OverallStats struct {
			TotalFrames         int     `json:"total_frames"`
			TotalDistanceMeters float64 `json:"total_distance_meters"`
			SegmentLengthMeters int     `json:"segment_length_meters"`
			TotalSegments       int     `json:"total_segments"`
			SegmentsWithData    int     `json:"segments_with_data"`
			AverageCoverage     float64 `json:"average_coverage"`
		} `json:"overall_stats"`
		Segments []struct {
			SegmentID          int     `json:"segment_id"`
			FramesCount        int     `json:"frames_count"`
			CoveragePercentage float64 `json:"coverage_percentage"`
			HasData            bool    `json:"has_data"`
		} `json:"segments"`
		Coordinates struct {
			Start struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"start"`
			End struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"end"`
		} `json:"coordinates"`
	}

	if err := json.Unmarshal(analysisData, &pythonResults); err != nil {
		return nil, nil, fmt.Errorf("failed to parse analysis results: %w", err)
	}

	s.logger.Infof("Обработано кадров: %d, сегментов: %d",
		pythonResults.OverallStats.TotalFrames, pythonResults.OverallStats.TotalSegments)

	// Преобразуем результаты в наш формат
	segments := make([]SegmentInfo, len(pythonResults.Segments))
	for i, seg := range pythonResults.Segments {
		// Интерполируем координаты сегмента
		progress := float64(i) / float64(len(pythonResults.Segments))
		if len(pythonResults.Segments) == 1 {
			progress = 0.5
		}

		startSegLat := startLat + (endLat-startLat)*progress
		startSegLon := startLon + (endLon-startLon)*progress

		endProgress := float64(i+1) / float64(len(pythonResults.Segments))
		if i == len(pythonResults.Segments)-1 {
			endProgress = 1.0
		}

		endSegLat := startLat + (endLat-startLat)*endProgress
		endSegLon := startLon + (endLon-startLon)*endProgress

		segments[i] = SegmentInfo{
			SegmentID:          seg.SegmentID,
			FramesCount:        seg.FramesCount,
			CoveragePercentage: seg.CoveragePercentage,
			HasData:            seg.HasData,
			StartCoordinate: Coordinates{
				Lat: startSegLat,
				Lon: startSegLon,
			},
			EndCoordinate: Coordinates{
				Lat: endSegLat,
				Lon: endSegLon,
			},
		}
	}

	// Создаем финальный результат
	result := &AnalysisResult{
		StartPoint: Coordinates{
			Lat: startLat,
			Lon: startLon,
		},
		EndPoint: Coordinates{
			Lat: endLat,
			Lon: endLon,
		},
		SegmentLength: segmentLength,
		Segments:      segments,
		OverallStats: OverallStats{
			TotalFrames:         pythonResults.OverallStats.TotalFrames,
			TotalDistanceMeters: pythonResults.OverallStats.TotalDistanceMeters,
			SegmentLengthMeters: segmentLength,
			TotalSegments:       pythonResults.OverallStats.TotalSegments,
			SegmentsWithData:    pythonResults.OverallStats.SegmentsWithData,
			AverageCoverage:     pythonResults.OverallStats.AverageCoverage,
		},
	}

	return result, videoData, nil
}

// saveAnnotatedVideo сохраняет аннотированное видео на диск
func (s *AnalyzerService) saveAnnotatedVideo(filePath string, videoData []byte) error {
	// Создаем директорию если не существует
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Записываем файл
	err := os.WriteFile(filePath, videoData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write video file %s: %w", filePath, err)
	}

	s.logger.Infof("Аннотированное видео сохранено: %s (%d байт)", filePath, len(videoData))
	return nil
}
