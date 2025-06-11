package service

import (
	"time"
)

// Coordinates представляет географические координаты
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// SegmentInfo информация о сегменте дороги
type SegmentInfo struct {
	SegmentID          int         `json:"segment_id"`
	FramesCount        int         `json:"frames_count"`
	CoveragePercentage float64     `json:"coverage_percentage"`
	HasData            bool        `json:"has_data"`
	StartCoordinate    Coordinates `json:"start_coordinate"`
	EndCoordinate      Coordinates `json:"end_coordinate"`
}

// OverallStats общая статистика анализа
type OverallStats struct {
	TotalFrames         int     `json:"total_frames"`
	TotalDistanceMeters float64 `json:"total_distance_meters"`
	SegmentLengthMeters float64 `json:"segment_length_meters"`
	TotalSegments       int     `json:"total_segments"`
	SegmentsWithData    int     `json:"segments_with_data"`
	AverageCoverage     float64 `json:"average_coverage"`
}

// AnalysisResult результат анализа дороги
type AnalysisResult struct {
	StartPoint    Coordinates   `json:"start_point"`
	EndPoint      Coordinates   `json:"end_point"`
	SegmentLength float64       `json:"segment_length"`
	Segments      []SegmentInfo `json:"segments"`
	OverallStats  OverallStats  `json:"overall_stats"`
}

// RouteResponse ответ с информацией о маршруте
type RouteResponse struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	StartPoint    Coordinates   `json:"start_point"`
	EndPoint      Coordinates   `json:"end_point"`
	SegmentLength float64       `json:"segment_length"`
	Segments      []SegmentInfo `json:"segments"`
	OverallStats  OverallStats  `json:"overall_stats"`
	CreatedAt     time.Time     `json:"created_at"`
	VideoFilename string        `json:"video_filename,omitempty"`
	VideoPath     string        `json:"video_path,omitempty"`
}

// SaveRouteRequest запрос на сохранение маршрута
type SaveRouteRequest struct {
	RouteID       string          `json:"route_id"`
	VideoFilename string          `json:"video_filename,omitempty"`
	AnalysisData  *AnalysisResult `json:"analysis_data"`
}

// GetSegmentsByAreaRequest запрос на получение сегментов по области
type GetSegmentsByAreaRequest struct {
	NorthEast Coordinates `json:"north_east"`
	SouthWest Coordinates `json:"south_west"`
}

// GetSegmentsByAreaResponse ответ со списком сегментов в области
type GetSegmentsByAreaResponse struct {
	Routes []RouteResponse `json:"routes"`
	Total  int             `json:"total"`
}

// ListRoutesResponse ответ со списком маршрутов
type ListRoutesResponse struct {
	Routes []RouteResponse `json:"routes"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Size   int             `json:"size"`
}
