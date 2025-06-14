-- Удаляем уникальный индекс на route_id и segment_id
DROP INDEX IF EXISTS idx_segments_route_segment_unique;

-- Удаляем избыточные индексы, если они есть
DROP INDEX IF EXISTS idx_segments_route_id;
DROP INDEX IF EXISTS idx_segments_segment_id; 