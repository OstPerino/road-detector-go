-- Удаляем дублирующийся внешний ключ
ALTER TABLE segments DROP CONSTRAINT IF EXISTS fk_segments_route;

-- Удаляем избыточные индексы
DROP INDEX IF EXISTS idx_segments_route_id;
DROP INDEX IF EXISTS idx_segments_segment_id;

-- Оставляем только уникальный индекс для route_id + segment_id
-- (он уже существует как idx_segments_route_segment_unique) 