package models

// Coordinates представляет географические координаты
type Coordinates struct {
	Lat float64 `json:"lat"` // Широта
	Lon float64 `json:"lon"` // Долгота
}

// AnalyzeRequest представляет запрос на анализ дорожной разметки
type AnalyzeRequest struct {
	VideoData     []byte      `json:"-"`              // Данные видео файла (не сериализуем в JSON)
	VideoFilename string      `json:"video_filename"` // Имя видео файла
	StartPoint    Coordinates `json:"start_point"`    // Начальная точка маршрута
	EndPoint      Coordinates `json:"end_point"`      // Конечная точка маршрута
	SegmentLength int         `json:"segment_length"` // Длина сегмента в метрах
}

// SegmentInfo содержит информацию о сегменте дороги
type SegmentInfo struct {
	SegmentID          int32       `json:"segment_id"`          // ID сегмента
	FramesCount        int32       `json:"frames_count"`        // Количество кадров в сегменте
	CoveragePercentage float64     `json:"coverage_percentage"` // Процент покрытия разметкой
	StartCoordinate    Coordinates `json:"start_coordinate"`    // Начальные координаты сегмента
	EndCoordinate      Coordinates `json:"end_coordinate"`      // Конечные координаты сегмента
	HasData            bool        `json:"has_data"`            // Наличие данных в сегменте
}

// OverallStats содержит общую статистику анализа
type OverallStats struct {
	TotalFrames         int32   `json:"total_frames"`          // Общее количество кадров
	TotalDistanceMeters float64 `json:"total_distance_meters"` // Общее расстояние в метрах
	SegmentLengthMeters int32   `json:"segment_length_meters"` // Длина сегмента в метрах
	TotalSegments       int32   `json:"total_segments"`        // Общее количество сегментов
	SegmentsWithData    int32   `json:"segments_with_data"`    // Количество сегментов с данными
	AverageCoverage     float64 `json:"average_coverage"`      // Среднее покрытие по всем сегментам
}

// AnalyzeResponse представляет ответ анализа дорожной разметки
type AnalyzeResponse struct {
	Status       string       `json:"status"`        // Статус выполнения (success/error)
	Message      string       `json:"message"`       // Сообщение о результате
	OverallStats OverallStats `json:"overall_stats"` // Общая статистика
	Segments     []SegmentInfo `json:"segments"`     // Информация о сегментах
}

// PythonAPIResponse определяет структуру ответа от Python FastAPI сервиса
type PythonAPIResponse struct {
	Status       string `json:"status"`        // Статус выполнения
	Message      string `json:"message"`       // Сообщение
	FrameResults []int  `json:"frame_results"` // Результаты анализа по кадрам (1 - есть разметка, 0 - нет)
}

// HealthResponse представляет ответ проверки здоровья сервиса
type HealthResponse struct {
	Status      string `json:"status"`       // Статус сервиса (healthy/unhealthy)
	ModelLoaded bool   `json:"model_loaded"` // Загружена ли модель нейронной сети
	Version     string `json:"version"`      // Версия сервиса
} 