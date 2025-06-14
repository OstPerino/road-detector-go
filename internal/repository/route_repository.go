package repository

import (
	"fmt"

	"road-detector-go/internal/model"

	"gorm.io/gorm"
)

// RouteRepository интерфейс для работы с маршрутами
type RouteRepository interface {
	Create(route *model.Route) error
	GetByID(id string) (*model.Route, error)
	GetByArea(northEast, southWest Coordinates) ([]*model.Route, error)
	List(page, pageSize int) ([]*model.Route, int64, error)
	Delete(id string) error
	Update(route *model.Route) error
}

// Coordinates представляет координаты точки
type Coordinates struct {
	Lat float64
	Lon float64
}

// routeRepository реализация RouteRepository
type routeRepository struct {
	db *gorm.DB
}

// NewRouteRepository создает новый instance RouteRepository
func NewRouteRepository(db *gorm.DB) RouteRepository {
	return &routeRepository{
		db: db,
	}
}

// Create создает новый маршрут в базе данных
func (r *routeRepository) Create(route *model.Route) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Сначала создаем маршрут
	if err := tx.Create(route).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create route: %w", err)
	}

	// Затем создаем сегменты
	for i := range route.Segments {
		// Логируем данные сегмента перед созданием
		fmt.Printf("Создаем сегмент %d: RouteID=%s, SegmentID=%d, ID=%d\n",
			i, route.Segments[i].RouteID, route.Segments[i].SegmentID, route.Segments[i].ID)

		route.Segments[i].ID = 0 // Обнуляем ID для auto-increment
		route.Segments[i].RouteID = route.ID
		// Не обнуляем segment_id, он может быть любым

		if err := tx.Create(&route.Segments[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create segment %d: %w", i, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID получает маршрут по ID
func (r *routeRepository) GetByID(id string) (*model.Route, error) {
	var route model.Route
	err := r.db.Preload("Segments").Where("id = ?", id).First(&route).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("route with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get route: %w", err)
	}
	return &route, nil
}

// GetByArea получает маршруты в заданной области
func (r *routeRepository) GetByArea(northEast, southWest Coordinates) ([]*model.Route, error) {
	var routes []*model.Route

	// Находим маршруты, у которых есть сегменты в заданной области
	err := r.db.Preload("Segments").
		Joins("JOIN segments ON segments.route_id = routes.id").
		Where("(segments.start_lat BETWEEN ? AND ? AND segments.start_lon BETWEEN ? AND ?) OR "+
			"(segments.end_lat BETWEEN ? AND ? AND segments.end_lon BETWEEN ? AND ?)",
			southWest.Lat, northEast.Lat, southWest.Lon, northEast.Lon,
			southWest.Lat, northEast.Lat, southWest.Lon, northEast.Lon).
		Distinct("routes.id").
		Find(&routes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get routes by area: %w", err)
	}

	return routes, nil
}

// List получает список маршрутов с пагинацией
func (r *routeRepository) List(page, pageSize int) ([]*model.Route, int64, error) {
	var routes []*model.Route
	var total int64

	// Подсчитываем общее количество
	if err := r.db.Model(&model.Route{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count routes: %w", err)
	}

	// Получаем маршруты с пагинацией
	offset := (page - 1) * pageSize
	err := r.db.Preload("Segments").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&routes).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list routes: %w", err)
	}

	return routes, total, nil
}

// Delete удаляет маршрут по ID
func (r *routeRepository) Delete(id string) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Сначала удаляем сегменты
	if err := tx.Where("route_id = ?", id).Delete(&model.Segment{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete segments: %w", err)
	}

	// Затем удаляем маршрут
	result := tx.Where("id = ?", id).Delete(&model.Route{})
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete route: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("route with id %s not found", id)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Update обновляет маршрут
func (r *routeRepository) Update(route *model.Route) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Обновляем маршрут
	if err := tx.Save(route).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update route: %w", err)
	}

	// Удаляем старые сегменты
	if err := tx.Where("route_id = ?", route.ID).Delete(&model.Segment{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete old segments: %w", err)
	}

	// Создаем новые сегменты
	for i := range route.Segments {
		route.Segments[i].ID = 0 // Обнуляем ID для auto-increment
		route.Segments[i].RouteID = route.ID
		if err := tx.Create(&route.Segments[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create segment %d: %w", i, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
