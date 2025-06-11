package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"road-detector-go/internal/model"
	"road-detector-go/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RouteService сервис для работы с маршрутами
type RouteService struct {
	routeRepo repository.RouteRepository
	logger    *logrus.Logger
	staticDir string
}

// NewRouteService создает новый сервис для работы с маршрутами
func NewRouteService(routeRepo repository.RouteRepository, logger *logrus.Logger, staticDir string) *RouteService {
	return &RouteService{
		routeRepo: routeRepo,
		logger:    logger,
		staticDir: staticDir,
	}
}

// SaveRoute сохраняет маршрут в базе данных
func (s *RouteService) SaveRoute(routeID, videoFilename string, videoData io.Reader, analysisResult *AnalysisResult) error {
	s.logger.Infof("Сохраняем маршрут %s в базе данных", routeID)

	// Создаем уникальное имя файла для видео
	videoPath := ""
	if videoData != nil && videoFilename != "" {
		var err error
		videoPath, err = s.saveVideoFile(routeID, videoFilename, videoData)
		if err != nil {
			s.logger.Errorf("Ошибка сохранения видео файла: %v", err)
			return fmt.Errorf("failed to save video file: %w", err)
		}
	}

	// Преобразуем результат анализа в модель базы данных
	route := &model.Route{
		ID:                  routeID,
		Name:                fmt.Sprintf("Route %s", routeID[:8]),
		StartLat:            analysisResult.StartPoint.Lat,
		StartLon:            analysisResult.StartPoint.Lon,
		EndLat:              analysisResult.EndPoint.Lat,
		EndLon:              analysisResult.EndPoint.Lon,
		SegmentLengthM:      int32(analysisResult.SegmentLength),
		VideoFilename:       videoFilename,
		VideoPath:           videoPath,
		TotalFrames:         int32(analysisResult.OverallStats.TotalFrames),
		TotalDistanceMeters: analysisResult.OverallStats.TotalDistanceMeters,
		TotalSegments:       int32(analysisResult.OverallStats.TotalSegments),
		SegmentsWithData:    int32(analysisResult.OverallStats.SegmentsWithData),
		AverageCoverage:     analysisResult.OverallStats.AverageCoverage,
		CreatedAt:           time.Now(),
	}

	// Преобразуем сегменты
	for _, seg := range analysisResult.Segments {
		segment := model.Segment{
			RouteID:            routeID,
			SegmentID:          int32(seg.SegmentID),
			FramesCount:        int32(seg.FramesCount),
			CoveragePercentage: seg.CoveragePercentage,
			HasData:            seg.HasData,
			StartLat:           seg.StartCoordinate.Lat,
			StartLon:           seg.StartCoordinate.Lon,
			EndLat:             seg.EndCoordinate.Lat,
			EndLon:             seg.EndCoordinate.Lon,
		}
		route.Segments = append(route.Segments, segment)
	}

	// Сохраняем в базе данных
	s.logger.Infof("Сохраняем маршрут в БД. Количество сегментов: %d", len(route.Segments))
	err := s.routeRepo.Create(route)
	if err != nil {
		s.logger.Errorf("Ошибка сохранения маршрута в БД: %v", err)
		// Удаляем видео файл если что-то пошло не так
		if videoPath != "" {
			s.logger.Infof("Удаляем видео файл %s из-за ошибки сохранения в БД", videoPath)
			// os.Remove(videoPath)
		}
		return fmt.Errorf("failed to save route to database: %w", err)
	}

	s.logger.Infof("Маршрут %s успешно сохранен в БД с %d сегментами", routeID, len(route.Segments))
	return nil
}

// GetRouteByID получает маршрут по ID
func (s *RouteService) GetRouteByID(routeID string) (*RouteResponse, error) {
	s.logger.Infof("Получаем маршрут %s из базы данных", routeID)

	route, err := s.routeRepo.GetByID(routeID)
	if err != nil {
		s.logger.Errorf("Ошибка получения маршрута: %v", err)
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	return s.modelToResponse(route), nil
}

// GetRoutesByArea получает маршруты в заданной области
func (s *RouteService) GetRoutesByArea(neLat, neLon, swLat, swLon float64) ([]RouteResponse, error) {
	s.logger.Infof("Получаем маршруты в области: NE(%.6f, %.6f) SW(%.6f, %.6f)",
		neLat, neLon, swLat, swLon)

	// Преобразуем координаты
	ne := repository.Coordinates{Lat: neLat, Lon: neLon}
	sw := repository.Coordinates{Lat: swLat, Lon: swLon}

	routes, err := s.routeRepo.GetByArea(ne, sw)
	if err != nil {
		s.logger.Errorf("Ошибка получения маршрутов по области: %v", err)
		return nil, fmt.Errorf("failed to get routes by area: %w", err)
	}

	responses := make([]RouteResponse, len(routes))
	for i, route := range routes {
		responses[i] = *s.modelToResponse(route)
	}

	s.logger.Infof("Найдено %d маршрутов в области", len(responses))
	return responses, nil
}

// ListRoutes получает список всех маршрутов с пагинацией
func (s *RouteService) ListRoutes(page, pageSize int) ([]RouteResponse, int64, error) {
	s.logger.Infof("Получаем список маршрутов: страница %d, размер %d", page, pageSize)

	routes, total, err := s.routeRepo.List(page, pageSize)
	if err != nil {
		s.logger.Errorf("Ошибка получения списка маршрутов: %v", err)
		return nil, 0, fmt.Errorf("failed to list routes: %w", err)
	}

	responses := make([]RouteResponse, len(routes))
	for i, route := range routes {
		responses[i] = *s.modelToResponse(route)
	}

	s.logger.Infof("Получено %d маршрутов из %d общих", len(responses), total)
	return responses, total, nil
}

// DeleteRoute удаляет маршрут по ID
func (s *RouteService) DeleteRoute(routeID string) error {
	s.logger.Infof("Удаляем маршрут %s", routeID)

	// Сначала получаем маршрут для удаления видео файла
	route, err := s.routeRepo.GetByID(routeID)
	if err != nil {
		s.logger.Errorf("Ошибка получения маршрута для удаления: %v", err)
		return fmt.Errorf("failed to get route for deletion: %w", err)
	}

	// Удаляем из базы данных
	err = s.routeRepo.Delete(routeID)
	if err != nil {
		s.logger.Errorf("Ошибка удаления маршрута из БД: %v", err)
		return fmt.Errorf("failed to delete route from database: %w", err)
	}

	// Удаляем видео файл если он существует
	if route.VideoPath != "" {
		if err := os.Remove(route.VideoPath); err != nil {
			s.logger.Warnf("Не удалось удалить видео файл %s: %v", route.VideoPath, err)
		} else {
			s.logger.Infof("Видео файл %s успешно удален", route.VideoPath)
		}
	}

	s.logger.Infof("Маршрут %s успешно удален", routeID)
	return nil
}

// saveVideoFile сохраняет видео файл в статической папке
func (s *RouteService) saveVideoFile(routeID, originalFilename string, videoData io.Reader) (string, error) {
	s.logger.Infof("Начинаем сохранение видео файла. RouteID: %s, оригинальное имя: %s", routeID, originalFilename)

	// Создаем папку для маршрута
	routeDir := filepath.Join(s.staticDir, "videos", routeID)
	s.logger.Infof("Создаем директорию: %s", routeDir)
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		s.logger.Errorf("Ошибка создания директории %s: %v", routeDir, err)
		return "", fmt.Errorf("failed to create route directory: %w", err)
	}

	// Определяем расширение файла
	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".mp4" // По умолчанию
		s.logger.Warnf("Расширение файла не найдено, используем .mp4")
	}

	// Создаем путь к файлу
	filename := fmt.Sprintf("%s%s", routeID, ext)
	filePath := filepath.Join(routeDir, filename)
	s.logger.Infof("Путь к файлу: %s", filePath)

	// Создаем файл
	file, err := os.Create(filePath)
	if err != nil {
		s.logger.Errorf("Ошибка создания файла %s: %v", filePath, err)
		return "", fmt.Errorf("failed to create video file: %w", err)
	}
	defer file.Close()

	// Копируем данные
	bytesWritten, err := io.Copy(file, videoData)
	if err != nil {
		s.logger.Errorf("Ошибка записи данных в файл %s: %v", filePath, err)
		os.Remove(filePath) // Удаляем файл в случае ошибки
		return "", fmt.Errorf("failed to write video data: %w", err)
	}

	s.logger.Infof("Видео файл сохранен: %s (записано %d байт)", filePath, bytesWritten)
	return filePath, nil
}

// modelToResponse преобразует модель базы данных в ответ API
func (s *RouteService) modelToResponse(route *model.Route) *RouteResponse {
	response := &RouteResponse{
		ID:            route.ID,
		Name:          route.Name,
		StartPoint:    Coordinates{Lat: route.StartLat, Lon: route.StartLon},
		EndPoint:      Coordinates{Lat: route.EndLat, Lon: route.EndLon},
		SegmentLength: float64(route.SegmentLengthM),
		OverallStats: OverallStats{
			TotalFrames:         int(route.TotalFrames),
			TotalDistanceMeters: route.TotalDistanceMeters,
			SegmentLengthMeters: float64(route.SegmentLengthM),
			TotalSegments:       int(route.TotalSegments),
			SegmentsWithData:    int(route.SegmentsWithData),
			AverageCoverage:     route.AverageCoverage,
		},
		CreatedAt:     route.CreatedAt,
		VideoFilename: route.VideoFilename,
		VideoPath:     route.VideoPath,
	}

	// Преобразуем сегменты
	for _, seg := range route.Segments {
		segment := SegmentInfo{
			SegmentID:          int(seg.SegmentID),
			FramesCount:        int(seg.FramesCount),
			CoveragePercentage: seg.CoveragePercentage,
			HasData:            seg.HasData,
			StartCoordinate:    Coordinates{Lat: seg.StartLat, Lon: seg.StartLon},
			EndCoordinate:      Coordinates{Lat: seg.EndLat, Lon: seg.EndLon},
		}
		response.Segments = append(response.Segments, segment)
	}

	return response
}

// GenerateRouteID генерирует уникальный ID для маршрута
func (s *RouteService) GenerateRouteID() string {
	return uuid.New().String()
}
