# SimpleBlog HTTP API 文档

## 概述

- **Base URL**: `/`
- **认证方式**: JWT Bearer Token（部分接口需要）
- **响应格式**: JSON（RSS 接口返回 `application/rss+xml`）
- **字符编码**: UTF-8

## 统一响应格式

所有 API（除 RSS 外）返回统一的 JSON 结构：

```json
{
  "code": 200,
  "msg": "success",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 业务状态码，与 HTTP 状态码语义相同 |
| msg | string | 提示信息 |
| data | object/array | 业务数据，可能为空 |

### 常见错误码

| code | 说明 |
|------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证/认证失效 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如用户名重复） |
| 500 | 服务器内部错误 |

---

## 1. 服务接口 `/service`

### 1.1 健康检查

**GET** `/service/ping`

检查服务是否正常运行。

**请求**: 无

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": "Hello, 127.0.0.1"
}
```

### 1.2 测试 Turnstile 验证

**GET** `/service/test_turnstile`

测试 Cloudflare Turnstile 人机验证。

**请求头**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| X-Turnstile-Response | string | 是 | Turnstile 验证 token |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": "Turnstile verification passed!"
}
```

---

## 2. 认证接口 `/auth`

### 2.1 登录

**POST** `/auth/login`

用户登录获取 JWT Token。

**限流**: 每个 IP 每分钟最多 10 次请求。

**请求体**:
```json
{
  "username": "admin",
  "password": "your_password"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-01-02T00:00:00Z"
  }
}
```

### 2.2 获取当前用户信息

**GET** `/auth/me`

获取当前登录用户的详细信息。

**认证**: 需要 JWT Token

**请求头**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Authorization | string | 是 | `Bearer <token>` |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "nickname": "管理员",
    "avatar_url": "https://example.com/avatar.jpg",
    "role": "admin",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

---

## 3. 管理接口 `/admin`

所有 `/admin` 接口需要 JWT 认证。

**请求头**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Authorization | string | 是 | `Bearer <token>` |

### 3.1 用户管理（需要 admin 角色）

#### 3.1.1 获取用户列表

**GET** `/admin/users`

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "username": "admin",
      "nickname": "管理员",
      "avatar_url": "https://...",
      "role": "admin",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 3.1.2 创建用户

**POST** `/admin/users`

**请求体**:
```json
{
  "username": "newuser",
  "password": "password123",
  "nickname": "新用户",
  "avatar_url": "https://example.com/avatar.jpg",
  "role": "contributor"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名（唯一） |
| password | string | 是 | 密码 |
| nickname | string | 否 | 昵称 |
| avatar_url | string | 否 | 头像 URL |
| role | string | 是 | 角色：`admin` 或 `contributor` |

**响应**: 返回创建的用户信息

#### 3.1.3 获取单个用户

**GET** `/admin/users/{userID}`

**路径参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| userID | uuid | 用户 ID |

**响应**: 返回用户信息

#### 3.1.4 更新用户

**PATCH** `/admin/users/{userID}`

**请求体**（所有字段可选）:
```json
{
  "username": "newusername",
  "nickname": "新昵称",
  "avatar_url": "https://...",
  "role": "admin"
}
```

**响应**: 返回更新后的用户信息

#### 3.1.5 删除用户

**DELETE** `/admin/users/{userID}`

**响应**:
```json
{
  "code": 200,
  "msg": "user deleted",
  "data": null
}
```

#### 3.1.6 更新用户密码

**PUT** `/admin/users/{userID}/password`

**请求体**:
```json
{
  "password": "new_password"
}
```

**响应**:
```json
{
  "code": 200,
  "msg": "password updated",
  "data": null
}
```

### 3.2 分类管理（需要 admin 角色）

#### 3.2.1 获取分类列表

**GET** `/admin/categories`

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "name": "技术",
      "slug": "tech",
      "description": "技术相关文章",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 3.2.2 创建分类

**POST** `/admin/categories`

**请求体**:
```json
{
  "name": "技术",
  "slug": "tech",
  "description": "技术相关文章"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 分类名称 |
| slug | string | 是 | URL 别名（唯一） |
| description | string | 否 | 分类描述 |

#### 3.2.3 获取单个分类

**GET** `/admin/categories/{categoryID}`

#### 3.2.4 更新分类

**PATCH** `/admin/categories/{categoryID}`

**请求体**（所有字段可选）:
```json
{
  "name": "新名称",
  "slug": "new-slug",
  "description": "新描述"
}
```

#### 3.2.5 删除分类

**DELETE** `/admin/categories/{categoryID}`

### 3.3 友链管理（需要 admin 角色）

#### 3.3.1 获取友链列表

**GET** `/admin/links`

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "name": "友站名称",
      "description": "友站描述",
      "url": "https://friend.example.com",
      "favicon_url": "https://friend.example.com/favicon.ico",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 3.3.2 创建友链

**POST** `/admin/links`

**请求体**:
```json
{
  "name": "友站名称",
  "description": "友站描述",
  "url": "https://friend.example.com",
  "favicon_url": "https://friend.example.com/favicon.ico"
}
```

#### 3.3.3 获取单个友链

**GET** `/admin/links/{linkID}`

#### 3.3.4 更新友链

**PATCH** `/admin/links/{linkID}`

**请求体**（所有字段可选）:
```json
{
  "name": "新名称",
  "description": "新描述",
  "url": "https://new.url",
  "favicon_url": "https://new.url/favicon.ico"
}
```

#### 3.3.5 删除友链

**DELETE** `/admin/links/{linkID}`

### 3.4 文章管理（认证用户即可）

#### 3.4.1 获取文章列表

**GET** `/admin/posts`

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 每页数量，默认值由服务端决定 |
| cursor | string | 否 | 分页游标，来自上次响应的 `next_cursor` |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "author_id": "uuid",
        "category_id": "uuid",
        "slug": "my-first-post",
        "title": "我的第一篇文章",
        "summary": "文章摘要...",
        "tags": ["Go", "Web"],
        "status": "published",
        "published_at": "2024-01-01T00:00:00Z",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ],
    "next_cursor": "eyJpZCI6IjEyMy..."
  }
}
```

#### 3.4.2 创建文章

**POST** `/admin/posts`

**请求体**:
```json
{
  "category_id": "uuid",
  "slug": "my-first-post",
  "title": "我的第一篇文章",
  "summary": "文章摘要",
  "content": "文章正文内容...",
  "tags": ["Go", "Web"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category_id | uuid | 是 | 分类 ID |
| slug | string | 是 | URL 别名（唯一） |
| title | string | 是 | 标题 |
| summary | string | 否 | 摘要 |
| content | string | 否 | 正文 |
| tags | string[] | 否 | 标签数组 |

#### 3.4.3 获取单篇文章

**GET** `/admin/posts/{postID}`

**响应**: 返回完整文章信息（包含 content 字段）

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "uuid",
    "author_id": "uuid",
    "category_id": "uuid",
    "slug": "my-first-post",
    "title": "我的第一篇文章",
    "summary": "文章摘要",
    "content": "完整正文内容...",
    "tags": ["Go", "Web"],
    "status": "draft",
    "published_at": null,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 3.4.4 更新文章

**PATCH** `/admin/posts/{postID}`

**请求体**（所有字段可选）:
```json
{
  "category_id": "uuid",
  "slug": "new-slug",
  "title": "新标题",
  "summary": "新摘要",
  "content": "新内容",
  "tags": ["NewTag"]
}
```

#### 3.4.5 删除文章

**DELETE** `/admin/posts/{postID}`

#### 3.4.6 发布文章

**POST** `/admin/posts/{postID}/publish`

将文章状态从 `draft` 改为 `published`，并设置 `published_at`。

**响应**: 返回更新后的文章信息

#### 3.4.7 撤回文章

**POST** `/admin/posts/{postID}/withdraw`

将文章状态从 `published` 改为 `draft`。

**响应**: 返回更新后的文章信息

### 3.5 评论管理（认证用户即可）

#### 3.5.1 获取评论列表

**GET** `/admin/comments`

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 每页数量 |
| cursor | string | 否 | 分页游标 |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "post_id": "uuid",
        "post_slug": "my-first-post",
        "post_title": "我的第一篇文章",
        "parent_id": null,
        "nickname": "访客",
        "email": "visitor@example.com",
        "content": "评论内容",
        "status": "visible",
        "created_at": "2024-01-01T00:00:00Z",
        "moderated_at": null
      }
    ],
    "next_cursor": "..."
  }
}
```

#### 3.5.2 隐藏评论

**POST** `/admin/comments/{commentID}/hide`

将评论状态设为 `hidden`。

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "uuid",
    "status": "hidden",
    "moderated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 3.5.3 显示评论

**POST** `/admin/comments/{commentID}/show`

将评论状态设为 `visible`。

#### 3.5.4 删除评论

**DELETE** `/admin/comments/{commentID}`

将评论状态设为 `deleted`（软删除）。

### 3.6 上传管理（认证用户即可）

#### 3.6.1 获取预签名上传 URL

**POST** `/admin/uploads/presign`

获取 S3 直传所需的预签名 URL 和参数。

**请求体**:
```json
{
  "filename": "image.jpg",
  "content_type": "image/jpeg",
  "size": 102400
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| filename | string | 是 | 文件名 |
| content_type | string | 是 | MIME 类型 |
| size | int64 | 是 | 文件大小（字节） |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "method": "POST",
    "object_key": "uploads/uuid-image.jpg",
    "upload_url": "https://s3.example.com/bucket",
    "fields": {
      "key": "uploads/uuid-image.jpg",
      "policy": "base64...",
      "x-amz-credential": "...",
      "x-amz-algorithm": "AWS4-HMAC-SHA256",
      "x-amz-date": "20240101T000000Z",
      "x-amz-signature": "..."
    },
    "public_url": "https://cdn.example.com/uploads/uuid-image.jpg",
    "expires_at": "2024-01-01T00:10:00Z"
  }
}
```

**前端上传流程**:
1. 调用此接口获取预签名信息
2. 使用返回的 `method` 和 `upload_url`，将 `fields` 作为表单字段，文件作为 `file` 字段，发送 multipart/form-data 请求到 S3
3. 上传成功后使用 `public_url` 作为文件访问地址

---

## 4. RSS 接口 `/rss`

RSS 接口返回 `application/rss+xml` 格式。

### 4.1 文章 RSS

**GET** `/rss/article`

获取最新发布文章的 RSS 订阅。

**响应**: RSS 2.0 XML 格式

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>站点名称 - 文章</title>
    <link>https://example.com</link>
    <description>站点描述</description>
    <lastBuildDate>Mon, 01 Jan 2024 00:00:00 +0000</lastBuildDate>
    <item>
      <title>文章标题</title>
      <link>https://example.com/posts/my-first-post</link>
      <description>文章摘要</description>
      <guid isPermaLink="false">urn:uuid:article-uuid</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>
```

### 4.2 评论 RSS

**GET** `/rss/comment`

获取最新评论的 RSS 订阅。

**响应**: RSS 2.0 XML 格式

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>站点名称 - 评论</title>
    <link>https://example.com</link>
    <description>站点描述</description>
    <item>
      <title>访客 评论了 我的第一篇文章</title>
      <link>https://example.com/posts/my-first-post#comment-uuid</link>
      <description>评论内容</description>
      <guid isPermaLink="false">urn:uuid:comment-uuid</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>
```

---

## 5. 公开接口 `/`

公开接口无需认证。

### 5.1 获取文章列表

**GET** `/posts`

获取已发布的文章列表，支持分页和筛选。

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 每页数量 |
| cursor | string | 否 | 分页游标 |
| category | string | 否 | 按分类 slug 筛选 |
| tag | string | 否 | 按标签筛选 |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "items": [
      {
        "id": "uuid",
        "slug": "my-first-post",
        "title": "我的第一篇文章",
        "summary": "文章摘要",
        "tags": ["Go", "Web"],
        "published_at": "2024-01-01T00:00:00Z",
        "author": {
          "id": "uuid",
          "nickname": "作者昵称",
          "avatar_url": "https://..."
        },
        "category": {
          "id": "uuid",
          "name": "技术",
          "slug": "tech"
        }
      }
    ],
    "next_cursor": "..."
  }
}
```

### 5.2 获取文章详情

**GET** `/posts/{slug}`

**路径参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| slug | string | 文章 URL 别名 |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "uuid",
    "slug": "my-first-post",
    "title": "我的第一篇文章",
    "summary": "文章摘要",
    "content": "完整正文内容...",
    "tags": ["Go", "Web"],
    "published_at": "2024-01-01T00:00:00Z",
    "author": {
      "id": "uuid",
      "nickname": "作者昵称",
      "avatar_url": "https://..."
    },
    "category": {
      "id": "uuid",
      "name": "技术",
      "slug": "tech"
    }
  }
}
```

### 5.3 获取文章评论

**GET** `/posts/{slug}/comments`

获取指定文章的评论树形结构。

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "parent_id": null,
      "nickname": "访客",
      "avatar_url": "https://gravatar.com/avatar/...",
      "content": "评论内容",
      "status": "visible",
      "created_at": "2024-01-01T00:00:00Z",
      "replies": [
        {
          "id": "uuid",
          "parent_id": "parent-uuid",
          "nickname": "回复者",
          "avatar_url": "https://gravatar.com/avatar/...",
          "content": "回复内容",
          "status": "visible",
          "created_at": "2024-01-01T00:00:00Z",
          "replies": []
        }
      ]
    }
  ]
}
```

**说明**:
- 评论以树形结构返回，顶级评论 `parent_id` 为 `null`
- `replies` 包含子评论数组
- 隐藏/删除的评论会显示特殊文本，内容为空

### 5.4 发表评论

**POST** `/posts/{slug}/comments`

需要 Turnstile 人机验证。

**请求头**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| X-Turnstile-Response | string | 是 | Turnstile 验证 token |

**请求体**:
```json
{
  "parent_id": "uuid-or-null",
  "nickname": "访客昵称",
  "email": "visitor@example.com",
  "content": "评论内容"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_id | uuid | 否 | 父评论 ID，为 null 表示顶级评论 |
| nickname | string | 是 | 昵称 |
| email | string | 是 | 邮箱（用于显示 Gravatar 头像） |
| content | string | 是 | 评论内容 |

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "uuid",
    "parent_id": null,
    "nickname": "访客昵称",
    "avatar_url": "https://gravatar.com/avatar/...",
    "content": "评论内容",
    "status": "visible",
    "created_at": "2024-01-01T00:00:00Z",
    "replies": []
  }
}
```

### 5.5 获取分类列表

**GET** `/categories`

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "name": "技术",
      "slug": "tech",
      "description": "技术相关文章"
    }
  ]
}
```

### 5.6 获取标签列表

**GET** `/tags`

获取所有已使用标签。

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": ["Go", "Web", "Docker", "Kubernetes"]
}
```

### 5.7 获取友链列表

**GET** `/links`

**响应**:
```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "uuid",
      "name": "友站名称",
      "description": "友站描述",
      "url": "https://friend.example.com",
      "favicon_url": "https://friend.example.com/favicon.ico",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

## 附录

### A. 数据类型说明

#### Role（用户角色）
| 值 | 说明 |
|------|------|
| admin | 管理员，拥有所有权限 |
| contributor | 投稿者，可管理文章和评论 |

#### PostStatus（文章状态）
| 值 | 说明 |
|------|------|
| draft | 草稿，不对外公开 |
| published | 已发布，对外公开 |

#### CommentStatus（评论状态）
| 值 | 说明 |
|------|------|
| visible | 可见 |
| hidden | 已隐藏 |
| deleted | 已删除 |

### B. 认证说明

1. 调用 `POST /auth/login` 获取 JWT Token
2. 在需要认证的接口请求头中添加：`Authorization: Bearer <token>`
3. Token 过期后需重新登录

### C. 分页说明

使用游标分页（Cursor-based Pagination）：
- 首次请求不带 `cursor` 参数
- 响应中的 `next_cursor` 用于获取下一页
- `next_cursor` 为空表示已到最后一页
- 下一页请求携带 `cursor=<next_cursor>`

### D. Turnstile 验证说明

需要 Turnstile 验证的接口：
- `POST /posts/{slug}/comments` - 发表评论

前端集成方式：
1. 在页面加载时引入 Turnstile 脚本
2. 渲染 Turnstile widget
3. 用户完成验证后获取 token
4. 将 token 放入请求头 `X-Turnstile-Response`
