# SimpleBlog 数据库设计

## 1. 基线说明

当前 PostgreSQL 中没有业务表，因此首份 schema 直接创建 `users`，不设计 `admins -> users` 数据迁移。UUID 由 Go 应用生成，避免依赖数据库扩展。

通用约定：

- 时间使用 `TIMESTAMPTZ`。
- `updated_at` 由 repository 的 UPDATE 语句显式设置为 `NOW()`，不使用触发器。
- role/status 使用 `VARCHAR + CHECK`，不使用 PostgreSQL enum。
- 邮箱、密码哈希等敏感字段不能进入日志和公开 DTO。

## 2. 实体关系

```text
users      1 ---- n posts
categories 1 ---- n posts
posts      1 ---- n comments
comments   1 ---- n comments (parent_id)
```

标签是 `posts.tags TEXT[]`，不是独立实体。

## 3. 初始 Schema

```sql
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
```

## 4. 数据规则

### 4.1 用户删除

用户删除是软删除：设置 `deleted_at`，不物理删除用户。认证查询必须包含：

```sql
WHERE deleted_at IS NULL
```

这样可以保留文章作者关系。删除或降级 admin 前，service 必须在事务中保证系统仍至少存在一个未删除 admin。

### 4.2 标签

标签由 PostService 规范化后整体写入：

- trim 空格。
- 删除空标签。
- 英文转为小写。
- 去重并保留第一次出现的顺序。
- 最多 10 个，每项建议不超过 30 个 Unicode 字符。

查询标签：

```sql
SELECT DISTINCT UNNEST(tags) AS tag
FROM posts
WHERE status = 'published'
ORDER BY tag;
```

按标签筛选使用数组包含操作，并由 GIN 索引支持。

### 4.3 文章状态

- draft 可以没有 `published_at`。
- 首次发布设置 `published_at = NOW()`。
- 撤回只修改 status，保留 published_at。
- 再发布默认不刷新 published_at。

公开 repository 的 SQL 必须直接包含 `status = 'published'`，不能仅依靠 handler 过滤。

公开列表以 `(published_at, id)` 作为降序稳定游标：

```sql
WHERE status = 'published'
  AND (published_at, id) < ($cursor_published_at, $cursor_id)
ORDER BY published_at DESC, id DESC
LIMIT $limit_plus_one;
```

首次请求省略游标条件。查询多取一条用于判断是否存在下一页，不执行额外 COUNT。

### 4.4 多级评论

`parent_id` 指向直属父评论。复合外键 `(parent_id, post_id)` 保证父评论与回复属于同一文章。

CommentService 还必须保证：

- 只能对已发布文章发表评论。
- parent_id 创建后不可更改。
- 不设置最大回复深度；前端展示策略不改变数据库中的真实 parent_id。
- 回复已删除或已隐藏评论是否允许，由当前设计统一设为允许；回复仍挂在原节点下。
- 返回公开评论树时，email 永远不进入 DTO。

隐藏：

```sql
UPDATE comments
SET status = 'hidden', moderated_at = NOW()
WHERE id = $1;
```

恢复仅适用于 hidden：

```sql
UPDATE comments
SET status = 'visible', moderated_at = NOW()
WHERE id = $1 AND status = 'hidden';
```

删除使用不可恢复墓碑，保留 parent_id 和 id。业务代码不得对单条评论执行物理 DELETE：

```sql
UPDATE comments
SET status = 'deleted',
    nickname = '该评论已被删除',
    email = '',
    content = '',
    moderated_at = NOW()
WHERE id = $1;
```

公开端对 hidden/deleted 返回占位节点，同时继续返回其可见后代。deleted 节点的固定文案为“该评论已被删除”。只有删除整篇文章时，`posts -> comments` 外键才物理级联删除整棵评论树。

父级自引用使用 `ON DELETE NO ACTION`，用于阻止误删仍有回复的父评论；删除文章时，同一条级联语句会删除该文章的全部评论，不留下悬空引用。

## 5. 不建立的表

- tags：标签属于 posts 属性。
- post_tags：不需要多对多关系。
- uploads/media：第一版仅签发 S3 预签名地址，不做媒体管理。
- sessions/refresh_tokens：第一版使用 24 小时 JWT，不维护服务端会话或 refresh token。
