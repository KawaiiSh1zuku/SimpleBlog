BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE posts
    ADD COLUMN cover_img TEXT;

CREATE INDEX posts_published_search_trgm_idx
    ON posts USING GIN (
        (LOWER(title || ' ' || COALESCE(summary, '') || ' ' || content)) gin_trgm_ops
    )
    WHERE status = 'published';

COMMIT;
