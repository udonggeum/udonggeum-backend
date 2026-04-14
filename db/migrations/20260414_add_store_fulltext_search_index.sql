-- Migration: Add full-text search GIN index on stores table
-- Date: 2026-04-14
-- Description: PostgreSQL tsvector 기반 Full-Text Search 인덱스 추가
--              공백 단위 토크나이징으로 "봉명동" → "명동" 오매칭 방지

CREATE INDEX IF NOT EXISTS idx_stores_fts
ON stores USING GIN (
    to_tsvector(
        'simple',
        coalesce(name, '') || ' ' ||
        coalesce(region, '') || ' ' ||
        coalesce(district, '') || ' ' ||
        coalesce(dong, '') || ' ' ||
        coalesce(address, '')
    )
);
