-- Создание таблицы маршрутов
CREATE TABLE IF NOT EXISTS routes (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    start_lat DOUBLE PRECISION NOT NULL,
    start_lon DOUBLE PRECISION NOT NULL,
    end_lat DOUBLE PRECISION NOT NULL,
    end_lon DOUBLE PRECISION NOT NULL,
    segment_length_m INTEGER NOT NULL,
    video_filename VARCHAR(255),
    video_path VARCHAR(500),
    total_frames INTEGER NOT NULL DEFAULT 0,
    total_distance_meters DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_segments INTEGER NOT NULL DEFAULT 0,
    segments_with_data INTEGER NOT NULL DEFAULT 0,
    average_coverage DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Создание индексов
CREATE INDEX IF NOT EXISTS idx_routes_created_at ON routes(created_at);
CREATE INDEX IF NOT EXISTS idx_routes_deleted_at ON routes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_routes_coordinates ON routes(start_lat, start_lon, end_lat, end_lon); 