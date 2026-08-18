BEGIN;

CREATE TABLE friend_links (
    id          UUID PRIMARY KEY,
    name        VARCHAR(80)   NOT NULL,
    description VARCHAR(500),
    url         VARCHAR(2048) NOT NULL,
    favicon_url VARCHAR(2048),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT friend_links_name_not_blank CHECK (BTRIM(name) <> ''),
    CONSTRAINT friend_links_url_not_blank CHECK (BTRIM(url) <> ''),
    CONSTRAINT friend_links_name_plain CHECK (name !~ '[<>]'),
    CONSTRAINT friend_links_description_plain CHECK (description IS NULL OR description !~ '[<>]')
);

CREATE INDEX friend_links_name_idx ON friend_links (LOWER(name), id);
CREATE INDEX comments_visible_created_idx
    ON comments (created_at DESC, id DESC)
    WHERE status = 'visible';

ALTER TABLE comments
    ADD CONSTRAINT comments_nickname_plain CHECK (nickname !~ '[<>]') NOT VALID,
    ADD CONSTRAINT comments_content_plain CHECK (content !~ '[<>]') NOT VALID;

COMMIT;
