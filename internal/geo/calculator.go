package geo

import (
	"math"
	"road-detector-go/pkg/models"
)

// Calculator для географических вычислений
type Calculator struct{}

// NewCalculator создает новый калькулятор
func NewCalculator() *Calculator {
	return &Calculator{}
}

// DistanceMeters вычисляет расстояние между двумя точками в метрах
// Использует формулу гаверсинуса
func (c *Calculator) DistanceMeters(point1, point2 models.Coordinates) float64 {
	const earthRadiusKm = 6371.0

	// Преобразуем градусы в радианы
	lat1Rad := point1.Lat * math.Pi / 180
	lon1Rad := point1.Lon * math.Pi / 180
	lat2Rad := point2.Lat * math.Pi / 180
	lon2Rad := point2.Lon * math.Pi / 180

	// Разности координат
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Формула гаверсинуса
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	chord := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Расстояние в метрах
	return earthRadiusKm * chord * 1000
}

// InterpolateCoordinates создает интерполированные координаты между двумя точками
func (c *Calculator) InterpolateCoordinates(start, end models.Coordinates, numPoints int) []models.Coordinates {
	if numPoints <= 0 {
		return []models.Coordinates{}
	}

	if numPoints == 1 {
		return []models.Coordinates{start}
	}

	coords := make([]models.Coordinates, numPoints)
	
	for i := 0; i < numPoints; i++ {
		// Линейная интерполяция
		ratio := float64(i) / float64(numPoints-1)
		
		coords[i] = models.Coordinates{
			Lat: start.Lat + (end.Lat-start.Lat)*ratio,
			Lon: start.Lon + (end.Lon-start.Lon)*ratio,
		}
	}

	return coords
}

// CalculateSegments разбивает маршрут на сегменты заданной длины
func (c *Calculator) CalculateSegments(start, end models.Coordinates, segmentLengthM int, frameCoords []models.Coordinates, frameResults []int) []models.SegmentInfo {
	totalDistance := c.DistanceMeters(start, end)
	numSegments := int(math.Ceil(totalDistance / float64(segmentLengthM)))
	
	// Инициализируем сегменты
	segments := make([]models.SegmentInfo, numSegments)
	segmentFrames := make([][]int, numSegments)
	
	// Распределяем кадры по сегментам
	for i, coord := range frameCoords {
		distFromStart := c.DistanceMeters(start, coord)
		segmentIdx := int(distFromStart / float64(segmentLengthM))
		
		// Ограничиваем индекс сегмента
		if segmentIdx >= numSegments {
			segmentIdx = numSegments - 1
		}
		
		segmentFrames[segmentIdx] = append(segmentFrames[segmentIdx], frameResults[i])
	}
	
	// Вычисляем статистику для каждого сегмента
	for i := 0; i < numSegments; i++ {
		segments[i].SegmentID = int32(i + 1)
		
		if len(segmentFrames[i]) > 0 {
			// Считаем покрытие
			totalMarkings := 0
			for _, marking := range segmentFrames[i] {
				totalMarkings += marking
			}
			
			coverage := float64(totalMarkings) / float64(len(segmentFrames[i])) * 100
			
			segments[i].FramesCount = int32(len(segmentFrames[i]))
			segments[i].CoveragePercentage = math.Round(coverage*10) / 10 // Округляем до 1 знака
			segments[i].HasData = true
			
			// Вычисляем координаты сегмента
			segmentStart := float64(i) * float64(segmentLengthM)
			segmentEnd := math.Min(float64(i+1)*float64(segmentLengthM), totalDistance)
			
			startRatio := segmentStart / totalDistance
			endRatio := segmentEnd / totalDistance
			
			segments[i].StartCoordinate = models.Coordinates{
				Lat: start.Lat + (end.Lat-start.Lat)*startRatio,
				Lon: start.Lon + (end.Lon-start.Lon)*startRatio,
			}
			
			segments[i].EndCoordinate = models.Coordinates{
				Lat: start.Lat + (end.Lat-start.Lat)*endRatio,
				Lon: start.Lon + (end.Lon-start.Lon)*endRatio,
			}
		} else {
			segments[i].FramesCount = 0
			segments[i].CoveragePercentage = 0
			segments[i].HasData = false
		}
	}
	
	return segments
}

// CalculateOverallStats вычисляет общую статистику
func (c *Calculator) CalculateOverallStats(segments []models.SegmentInfo, totalFrames int, totalDistance float64, segmentLength int) models.OverallStats {
	segmentsWithData := int32(0)
	var validCoverages []float64
	
	for _, segment := range segments {
		if segment.HasData {
			segmentsWithData++
			validCoverages = append(validCoverages, segment.CoveragePercentage)
		}
	}
	
	var averageCoverage float64
	if len(validCoverages) > 0 {
		sum := 0.0
		for _, coverage := range validCoverages {
			sum += coverage
		}
		averageCoverage = math.Round((sum/float64(len(validCoverages)))*10) / 10
	}
	
	return models.OverallStats{
		TotalFrames:         int32(totalFrames),
		TotalDistanceMeters: math.Round(totalDistance*100) / 100, // Округляем до 2 знаков
		SegmentLengthMeters: int32(segmentLength),
		TotalSegments:       int32(len(segments)),
		SegmentsWithData:    segmentsWithData,
		AverageCoverage:     averageCoverage,
	}
} 