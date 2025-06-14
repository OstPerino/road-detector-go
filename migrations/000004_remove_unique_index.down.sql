-- Восстанавливаем уникальный индекс
CREATE UNIQUE INDEX idx_segments_route_segment_unique ON segments(route_id, segment_id) WHERE deleted_at IS NULL;

-- Восстанавливаем избыточные индексы
CREATE INDEX IF NOT EXISTS idx_segments_route_id ON segments(route_id);
CREATE INDEX IF NOT EXISTS idx_segments_segment_id ON segments(segment_id); 