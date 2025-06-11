-- Создание таблицы сегментов
CREATE TABLE IF NOT EXISTS segments (
    id SERIAL PRIMARY KEY,
    route_id VARCHAR(36) NOT NULL,
    segment_id INTEGER NOT NULL,
    frames_count INTEGER NOT NULL DEFAULT 0,
    coverage_percentage DOUBLE PRECISION NOT NULL DEFAULT 0,
    has_data BOOLEAN NOT NULL DEFAULT FALSE,
    start_lat DOUBLE PRECISION NOT NULL,
    start_lon DOUBLE PRECISION NOT NULL,
    end_lat DOUBLE PRECISION NOT NULL,
    end_lon DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_segments_route FOREIGN KEY (route_id) REFERENCES routes(id) ON DELETE CASCADE
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_segments_route_id ON segments(route_id);
CREATE INDEX IF NOT EXISTS idx_segments_segment_id ON segments(segment_id);
CREATE INDEX IF NOT EXISTS idx_segments_deleted_at ON segments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_segments_coordinates ON segments(start_lat, start_lon, end_lat, end_lon);

-- Создание составного индекса для уникальности сегментов в пределах маршрута
CREATE UNIQUE INDEX IF NOT EXISTS idx_segments_route_segment_unique ON segments(route_id, segment_id) WHERE deleted_at IS NULL; 