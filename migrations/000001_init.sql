BEGIN;

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL,
    password_hash TEXT         NOT NULL,
    nickname      VARCHAR(80)  NOT NULL,
    avatar_url    TEXT,
    role          VARCHAR(16)  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    CONSTRAINT users_username_not_blank CHECK (BTRIM(username) <> ''),
    CONSTRAINT users_nickname_not_blank CHECK (BTRIM(nickname) <> ''),
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'contributor'))
);

CREATE UNIQUE INDEX users_username_active_uq
    ON users (LOWER(username))
    WHERE deleted_at IS NULL;

CREATE TABLE categories (
    id          UUID PRIMARY KEY,
    name        VARCHAR(80)  NOT NULL,
    slug        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT categories_name_not_blank CHECK (BTRIM(name) <> ''),
    CONSTRAINT categories_slug_not_blank CHECK (BTRIM(slug) <> '')
);

CREATE UNIQUE INDEX categories_name_uq ON categories (LOWER(name));
CREATE UNIQUE INDEX categories_slug_uq ON categories (LOWER(slug));

CREATE TABLE posts (
    id           UUID PRIMARY KEY,
    author_id    UUID         NOT NULL,
    category_id  UUID         NOT NULL,
    slug         VARCHAR(160) NOT NULL,
    title        VARCHAR(200) NOT NULL,
    summary      TEXT,
    content      TEXT         NOT NULL,
    tags         TEXT[]       NOT NULL DEFAULT '{}'::TEXT[],
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT posts_author_fk
        FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT posts_category_fk
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT posts_slug_not_blank CHECK (BTRIM(slug) <> ''),
    CONSTRAINT posts_title_not_blank CHECK (BTRIM(title) <> ''),
    CONSTRAINT posts_content_not_blank CHECK (BTRIM(content) <> ''),
    CONSTRAINT posts_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT posts_published_at_check
        CHECK (status <> 'published' OR published_at IS NOT NULL),
    CONSTRAINT posts_tags_count_check CHECK (CARDINALITY(tags) <= 10)
);

CREATE UNIQUE INDEX posts_slug_uq ON posts (LOWER(slug));
CREATE INDEX posts_public_idx
    ON posts (published_at DESC, id DESC)
    WHERE status = 'published';
CREATE INDEX posts_author_idx ON posts (author_id, updated_at DESC);
CREATE INDEX posts_category_public_idx
    ON posts (category_id, published_at DESC)
    WHERE status = 'published';
CREATE INDEX posts_tags_gin ON posts USING GIN (tags);

CREATE TABLE comments (
    id           UUID PRIMARY KEY,
    post_id      UUID         NOT NULL,
    parent_id    UUID,
    nickname     VARCHAR(80)  NOT NULL,
    email        VARCHAR(254) NOT NULL,
    content      TEXT         NOT NULL,
    status       VARCHAR(16)  NOT NULL DEFAULT 'visible',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    moderated_at TIMESTAMPTZ,

    CONSTRAINT comments_post_fk
        FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT comments_id_post_uq UNIQUE (id, post_id),
    CONSTRAINT comments_parent_fk
        FOREIGN KEY (parent_id, post_id)
        REFERENCES comments(id, post_id)
        ON DELETE NO ACTION,
    CONSTRAINT comments_parent_not_self CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT comments_nickname_not_blank CHECK (BTRIM(nickname) <> ''),
    CONSTRAINT comments_status_check
        CHECK (status IN ('visible', 'hidden', 'deleted'))
);

CREATE INDEX comments_post_parent_idx
    ON comments (post_id, parent_id, created_at, id);
CREATE INDEX comments_admin_idx
    ON comments (status, created_at DESC);

COMMIT;
