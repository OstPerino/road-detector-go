package model

import (
	"time"

	"gorm.io/gorm"
)

// Route представляет маршрут в базе данных
type Route struct {
	ID             string  `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name           string  `gorm:"type:varchar(255);not null" json:"name"`
	Description    string  `gorm:"type:text" json:"description"`
	StartLat       float64 `gorm:"not null" json:"start_lat"`
	StartLon       float64 `gorm:"not null" json:"start_lon"`
	EndLat         float64 `gorm:"not null" json:"end_lat"`
	EndLon         float64 `gorm:"not null" json:"end_lon"`
	SegmentLengthM int     `gorm:"not null" json:"segment_length_m"`
	VideoFilename  string  `gorm:"type:varchar(255)" json:"video_filename"`
	VideoPath      string  `gorm:"type:varchar(500)" json:"video_path"`

	// Общая статистика
	TotalFrames         int     `gorm:"not null;default:0" json:"total_frames"`
	TotalDistanceMeters float64 `gorm:"not null;default:0" json:"total_distance_meters"`
	TotalSegments       int     `gorm:"not null;default:0" json:"total_segments"`
	SegmentsWithData    int     `gorm:"not null;default:0" json:"segments_with_data"`
	AverageCoverage     float64 `gorm:"not null;default:0" json:"average_coverage"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Связь с сегментами
	Segments []Segment `gorm:"foreignKey:RouteID;constraint:OnDelete:CASCADE" json:"segments"`
}

// Segment представляет сегмент маршрута в базе данных
type Segment struct {
	ID                 uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	RouteID            string  `gorm:"type:varchar(36);not null;index" json:"route_id"`
	SegmentID          int32   `gorm:"not null" json:"segment_id"`
	FramesCount        int32   `gorm:"not null" json:"frames_count"`
	CoveragePercentage float64 `gorm:"not null" json:"coverage_percentage"`
	HasData            bool    `gorm:"not null" json:"has_data"`
	StartLat           float64 `gorm:"not null" json:"start_lat"`
	StartLon           float64 `gorm:"not null" json:"start_lon"`
	EndLat             float64 `gorm:"not null" json:"end_lat"`
	EndLon             float64 `gorm:"not null" json:"end_lon"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Обратная связь с маршрутом
	Route Route `gorm:"foreignKey:RouteID;references:ID" json:"-"`
}

// TableName указывает имя таблицы для Route
func (Route) TableName() string {
	return "routes"
}

// TableName указывает имя таблицы для Segment
func (Segment) TableName() string {
	return "segments"
}
