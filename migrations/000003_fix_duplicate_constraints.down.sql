-- Восстанавливаем внешний ключ
ALTER TABLE segments ADD CONSTRAINT fk_segments_route 
    FOREIGN KEY (route_id) REFERENCES routes(id) ON DELETE CASCADE;

-- Восстанавливаем индексы
CREATE INDEX IF NOT EXISTS idx_segments_route_id ON segments(route_id);
CREATE INDEX IF NOT EXISTS idx_segments_segment_id ON segments(segment_id); 