BEGIN;

DROP INDEX IF EXISTS posts_author_idx;
DROP INDEX IF EXISTS posts_tags_gin;
DROP INDEX IF EXISTS posts_category_public_idx;
DROP INDEX IF EXISTS comments_post_parent_idx;
DROP INDEX IF EXISTS comments_admin_idx;

CREATE INDEX posts_updated_idx
    ON posts (updated_at DESC, id DESC);
CREATE INDEX posts_author_updated_idx
    ON posts (author_id, updated_at DESC, id DESC);
CREATE INDEX posts_published_tags_gin
    ON posts USING GIN (tags)
    WHERE status = 'published';
CREATE INDEX posts_category_public_idx
    ON posts (category_id, published_at DESC, id DESC)
    WHERE status = 'published';

CREATE INDEX comments_post_created_idx
    ON comments (post_id, created_at, id);
CREATE INDEX comments_created_idx
    ON comments (created_at DESC, id DESC);

COMMIT;
