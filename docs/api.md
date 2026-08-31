# SimpleBlog API 设计

配套文档：[权限模型](permissions.md)、[配置说明](configuration.md)、[运行与部署](operations.md)。当前路由没有 `/api` 前缀。

## 1. 响应契约

除 RSS 外，所有由应用程序处理并返回的业务结果统一使用 HTTP 200。真实结果由 JSON `code` 字段表示：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

`data` 字段始终存在：列表为空时返回 `[]`，无业务数据或错误响应返回 `null`，不会省略字段。

失败示例：

```json
{
  "code": 403,
  "msg": "permission denied",
  "data": null
}
```

建议业务 code：

| code | 含义 |
|---:|---|
| 200 | 成功 |
| 400 | 请求格式或参数错误 |
| 401 | 未登录、JWT 无效或登录失败 |
| 403 | 已登录但没有权限 |
| 404 | 资源不存在 |
| 409 | 唯一键、状态或关联关系冲突 |
| 413 | 请求体超过 2 MiB 上限 |
| 422 | 请求格式正确但业务校验失败 |
| 429 | 登录请求超过单 IP 频率限制 |
| 500 | 未预期的服务端错误 |
| 503 | 数据库暂时不可用、死锁或事务需要重试 |

前端必须始终读取 `code`，不能只根据 HTTP 状态判断。以下路径也必须使用相同 JSON 格式和 HTTP 200：

- JWT/角色中间件拒绝。
- Turnstile 拒绝。
- Chi NotFound 与 MethodNotAllowed。
- panic 恢复。
- 参数解析和 body 大小限制。

反向代理、网络中断和进程崩溃不属于应用可控制的业务响应，仍可能产生真实非 200 或无响应。日志和监控必须按 body code/结构化日志统计错误，不能只统计 HTTP 状态。

浏览器 CORS 预检 `OPTIONS` 是传输层例外，成功时返回 HTTP 204 且无 JSON body。允许来源由 `CORS_ALLOWED_ORIGINS` 配置；业务请求仍遵循上述响应协议。

`GET /rss/article` 和 `GET /rss/comment` 是订阅器兼容所需的显式例外：成功返回 RSS 2.0 XML 与 `application/rss+xml; charset=utf-8`，内部失败返回真实 HTTP 500，不使用 JSON envelope。

有请求体的接口按 JSON 解析，客户端应发送 `Content-Type: application/json`；未知字段会被拒绝。全局请求体上限为 2 MiB，文章正文另限 1 MiB。成功或失败都可能省略空 `data`。

## 2. 路由分组

```text
/auth           登录和当前用户
/               公开内容
/admin          登录后的后台；contributor 也使用此路径
/rss            RSS 2.0 订阅
/service        健康检查等服务端点
```

`admin` 表示后台管理界面，不等同于所有路由都只允许 admin 角色。

## 3. 认证

```text
POST /auth/login
GET  /auth/me
```

登录请求：

```json
{"username": "admin", "password": "your-password"}
```

成功时 `data` 为：

```json
{"token": "jwt", "expires_at": "2026-08-19T12:00:00Z"}
```

- login 接收 username 和 password，成功后签发 24 小时有效的 JWT。
- login 按客户端 IP 限制为每分钟 10 次；超限仍返回 HTTP 200，响应体 code 为 429。
- me 返回当前用户的 id、username、nickname、avatar_url 和 role。
- JWT 使用 `Authorization: Bearer <token>`。
- `/auth/me` 返回 `id`、`username`、`nickname`、`avatar_url`、`role`、`created_at` 和 `updated_at`，不返回密码哈希。

## 4. 公开文章与分类

```text
GET /posts
GET /posts/{slug}
GET /search
GET /categories
GET /tags
```

文章列表支持：

- `limit`：默认 20，范围 1 到 100。
- `cursor`：服务端返回的 Base64URL 不透明游标，客户端不得解析或修改。
- `category`：按 category slug 精确筛选。
- `tag`：按规范化后的单个 tag 精确筛选。
- 按 published_at 倒序。

列表响应：

```json
{
  "items": [
    {
      "id": "uuid",
      "slug": "first-post",
      "title": "First Post",
      "summary": "summary",
      "cover_img": "https://cdn.example.com/cover.webp",
      "tags": ["go"],
      "published_at": "2026-08-18T12:00:00Z",
      "author": {"id": "uuid", "nickname": "Writer", "avatar_url": ""},
      "category": {"id": "uuid", "name": "Tech", "slug": "tech"}
    }
  ],
  "next_cursor": "opaque-cursor"
}
```

`next_cursor` 仅在存在下一页时返回。游标按 `(published_at, id)` 降序定位，翻页请求必须继续携带相同 category/tag 筛选条件。

全文字段搜索使用独立接口：

```text
GET /search?q=postgresql&limit=20&cursor=opaque-cursor
```

- `q` 必填，trim 后为 1 到 100 个 Unicode 字符，仅接受单行纯文本。
- 搜索范围为已发布文章的 title、summary 和 Markdown/LaTeX content，大小写不敏感。
- 返回结构、`limit` 范围和游标规则与 `GET /posts` 相同，结果仍按 `(published_at, id)` 倒序，而不是相关度排序。
- `/posts?tag=...` 保留为规范化 tag 的精确筛选；全文搜索不替代标签筛选。

公开端只能返回 published 文章。列表不返回 content；详情返回 Markdown + LaTeX 原文，不返回服务端渲染 HTML。作者响应只包含 id、nickname、avatar_url，不包含登录用户名。

`GET /categories` 返回全部分类的公开字段；`GET /tags` 只聚合已发布文章使用的标签，草稿标签不会出现。

## 5. 公开评论

```text
GET  /posts/{slug}/comments
POST /posts/{slug}/comments
```

创建评论请求包含：

```json
{
  "parent_id": null,
  "nickname": "visitor",
  "email": "visitor@example.com",
  "content": "comment content"
}
```

- parent_id 为空表示顶级评论。
- parent_id 非空表示回复指定评论。
- Turnstile token 推荐放入统一请求头，不写入评论 DTO。
- 只能评论已发布文章。
- 公开响应绝不包含 email。
- 服务端可读取 email，仅用于对 `trim + lowercase` 后的值计算 Gravatar MD5；响应返回 `avatar_url`，镜像基址由 `GRAVATAR_BASE_URL` 配置。
- Turnstile 请求头固定为 `X-Turnstile-Response`；缺失或验证失败分别返回业务 code 401/403。
- nickname 最大 80 个 Unicode 字符，email 必须是单一有效邮箱地址，parent_id 必须属于同一篇文章。
- nickname 与 content 是纯文本：服务端统一换行并 trim 后保存，拒绝 `<`、`>` 和危险控制字符；content 允许换行和制表符。
- 允许回复 hidden/deleted 节点；回复仍保留真实 parent_id。

`GET /posts/{slug}/comments` 的 `data` 返回根评论数组，每个节点递归包含 `replies`：

```json
[
  {
    "id": "uuid",
    "parent_id": null,
    "nickname": "visitor",
    "avatar_url": "https://www.gravatar.com/avatar/hash?d=identicon",
    "content": "comment content",
    "status": "visible",
    "created_at": "2026-08-18T12:00:00Z",
    "replies": []
  }
]
```

创建成功时 `data` 是新评论的单个节点，`replies` 为空数组。

hidden/deleted 节点返回占位内容、空 `avatar_url`，但保留 replies。deleted 节点固定显示“该评论已被删除”。后端不限制回复深度；前端采用类似 B 站的不随层级增加缩进的展示，但不能改变真实 parent_id 关系。

hidden 节点公开显示“评论已隐藏”且不返回原正文；deleted 节点不可恢复。评论树由后端按 parent_id 组装，任意层级回复都保留在 `replies` 中。

前端必须把 nickname/content 作为文本节点渲染，不得传给 `innerHTML`。后端会对历史旧数据再次中和 HTML 尖括号，但无法保护前端自行构造的不安全 DOM 属性或脚本字符串。

## 6. 用户管理

仅 admin：

```text
GET    /admin/users
POST   /admin/users
GET    /admin/users/{id}
PATCH  /admin/users/{id}
DELETE /admin/users/{id}
PUT    /admin/users/{id}/password
```

DELETE 执行软删除。修改密码使用独立路由，避免普通资料更新误改凭据。
管理员不能删除自己；删除或降级最后一个有效 admin 返回业务 code 409。用户名、昵称与角色更新使用 PATCH，未提供字段保持不变。

创建用户请求字段为 `username`、`password`、`nickname`、`avatar_url` 和 `role`。PATCH 只接受 `username`、`nickname`、`avatar_url`、`role`；修改密码请求固定为：

```json
{"password": "new-strong-password"}
```

## 7. 文章后台

```text
GET    /admin/posts
POST   /admin/posts
GET    /admin/posts/{id}
PATCH  /admin/posts/{id}
DELETE /admin/posts/{id}
POST   /admin/posts/{id}/publish
POST   /admin/posts/{id}/withdraw
```

创建请求：

```json
{
  "category_id": "uuid",
  "slug": "first-post",
  "title": "First Post",
  "summary": "optional summary",
  "cover_img": "https://cdn.example.com/cover.webp",
  "content": "# Markdown\n\n$$E = mc^2$$",
  "tags": ["go", "latex"]
}
```

- contributor 列表只返回自己的文章。
- contributor 只能操作自己的文章。
- admin 可以操作全部文章。
- author_id 和 status 不接受普通 PATCH 直接修改。
- 发布和撤回使用显式动作路由。
- 新文章固定为 draft；首次发布写入 published_at，撤回和再次发布保留原发布时间。
- slug 由客户端提供，服务端转为小写，只允许字母、数字和单连字符，最大 160 字符。
- cover_img 可省略或为空；非空时必须是无 userinfo 的绝对 HTTP(S) URL，最大 2048 个 Unicode 字符。
- 创建时 cover_img 为空，服务端会按正文顺序选择第一张合法绝对 HTTP(S) 图片作为头图。自动识别常用 Markdown 内联图片 `![alt](url)` 和 HTML `<img src="url">`；代码块、行内代码、HTML 注释、相对地址及非 HTTP(S) 地址会被忽略。
- PATCH 保存后的 cover_img 为空时，服务端从本次保存后的正文自动回填；因此前端显式传空字符串且正文有图时也会生成头图。正文没有合法图片时，空字符串会清除并存为 NULL。未传 cover_img 且已有头图时不会被正文变化覆盖。
- tags 会被 trim、转小写、去空、去重并保序，最多 10 个，每项最多 30 个 Unicode 字符。
- content 原样保存 Markdown 与 LaTeX 文本，不在服务端渲染。
- `GET /admin/posts` 支持 `limit`（默认 50，最大 100）和不透明 `cursor`，响应为 `{ "items": [...], "next_cursor": "..." }`。
- 后台文章列表不返回 content；编辑器通过 `GET /admin/posts/{id}` 获取完整正文。

## 8. 分类后台

仅 admin：

```text
GET    /admin/categories
POST   /admin/categories
GET    /admin/categories/{id}
PATCH  /admin/categories/{id}
DELETE /admin/categories/{id}
```

分类写入字段为 `name`、`slug` 和可选 `description`。slug 使用与文章相同的格式，最大 100 字符；name 最大 80 个 Unicode 字符。

```json
{"name": "Tech", "slug": "tech", "description": "Technology posts"}
```

存在关联文章的分类不能删除，返回业务 code 409。

标签不存在后台 CRUD；标签随文章创建或更新请求整体提交。

## 9. 评论后台

```text
GET    /admin/comments
POST   /admin/comments/{id}/hide
POST   /admin/comments/{id}/show
DELETE /admin/comments/{id}
```

- contributor 只管理自己文章下的评论。
- admin 管理全部评论。
- hide 可恢复。
- DELETE 写入 deleted 墓碑并清除个人信息，不物理删除回复树；公开端显示“该评论已被删除”。
- 后台列表包含 email 供站点管理；公开查询可读取 email 计算 Gravatar，但公开 DTO 和响应绝不返回 email。
- `GET /admin/comments` 支持 `limit`（默认 50，最大 100）和不透明 `cursor`，响应为 `{ "items": [...], "next_cursor": "..." }`。
- visible 只能 hide，hidden 可以 show，deleted 不能恢复或重复删除；非法状态转换返回业务 code 409。

## 10. S3 图片直传

```text
POST /admin/uploads/presign
```

请求只提供文件元信息，例如原始文件名、Content-Type 和大小。响应提供：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "method": "POST",
    "object_key": "blog/uploads/20260822/uuid.webp",
    "upload_url": "https://s3.example/...",
    "fields": {
      "key": "blog/uploads/20260822/uuid.webp",
      "Content-Type": "image/webp",
      "policy": "base64-policy",
      "x-amz-signature": "signature"
    },
    "public_url": "https://cdn.example/blog/uploads/20260822/uuid.webp",
    "expires_at": "2026-08-18T12:10:00Z"
  }
}
```

请求体固定为：

```json
{
  "filename": "diagram.webp",
  "content_type": "image/webp",
  "size": 123456
}
```

前端必须构造 `multipart/form-data`，先原样加入响应中的全部 `fields`，再以字段名 `file` 加入图片，向 `upload_url` 发起 POST。文件内容不经过 Blog API；上传成功后将 `public_url` 写入 Markdown。

首版上传限制：

- MIME：`image/jpeg`、`image/png`、`image/webp`、`image/gif`。
- 单文件最大 10 MiB。
- 服务端生成 object key，不接受客户端提供任意对象路径。
- object key 固定为 `blog/uploads/YYYYMMDD/{uuid}.{ext}`，日期使用服务进程本地时区。
- policy 同时约束精确 Content-Type 和请求声明的精确字节数，签名 10 分钟后过期。
- filename 仅作元数据校验，不参与 object key；服务端按 MIME 决定扩展名。
- admin 与 contributor 均可申请签名，未登录请求返回业务 code 401。

## 11. RSS 订阅

```text
GET /rss/article
GET /rss/comment
```

- 两个 feed 均返回最新 50 条，按时间倒序。
- 文章 feed 只包含 published 文章，description 使用纯文本 summary，不嵌入完整正文或可执行 HTML。
- 评论 feed 只包含 published 文章下的 visible 评论，不查询或输出邮箱。
- 文章链接为前端地址 `{SITE_BASE_URL}/articles/{slug}`；评论链接为对应文章地址追加 `#comment-{commentID}`。
- 所有 XML 由 `encoding/xml` 编码，条目内容不做字符串拼接。

## 12. 友情链接

公开读取为 `GET /links`。仅 admin 可使用：

```text
GET    /admin/links
POST   /admin/links
GET    /admin/links/{id}
PATCH  /admin/links/{id}
DELETE /admin/links/{id}
```

写入字段为 `name`、`description`、`url`、`favicon_url`。name 最大 80 字符，description 最大 500 字符；URL 最大 2048 字符，只接受不含 userinfo 的绝对 HTTP(S) URL，favicon_url 可为空。name/description 为纯文本。列表按名称大小写不敏感排序，再按 UUID 稳定排序。

## 13. 服务端点

```text
GET /service/ping
GET /service/test_turnstile
```

`/service/ping` 返回当前识别的客户端 IP，不访问数据库，是轻量存活端点而非数据库就绪检查。

`/service/test_turnstile` 需要 `X-Turnstile-Response` 请求头，仅用于检查 Turnstile siteverify 配置。缺失或验证失败遵循统一业务响应协议。

## 14. 输入与分页限制

| 项目 | 限制 |
|---|---|
| 全局请求体 | 最大 2 MiB |
| username | 3–64 个 Unicode 字符 |
| password | 10–72 字节 |
| 用户 nickname | 1–80 个 Unicode 字符 |
| 用户 avatar_url | 最大 2048 个 Unicode 字符 |
| 分类 name / slug / description | 1–80 / 最大 100 / 最大 2000 个 Unicode 字符 |
| 文章 title / slug | 1–200 / 最大 160 个 Unicode 字符 |
| 文章 summary / content | 最大 1000 个 Unicode 字符 / 最大 1 MiB（按 UTF-8 字节数） |
| 文章 tags | 最多 10 个，每项最多 30 个 Unicode 字符 |
| 评论 nickname / email / content | 最大 80 / 最大 254 / 最大 10000 个 Unicode 字符 |
| 友情链接 name / description / URL | 最大 80 / 最大 500 / 最大 2048 个 Unicode 字符 |
| 上传 filename / size | 最大 255 个 Unicode 字符 / 1 字节到 10 MiB |
| 公开文章分页 | 默认 20，`limit` 最大 100 |
| 后台文章、评论分页 | 默认 50，`limit` 最大 100 |
| 不透明 cursor | Base64URL 字符串，最大 512 字符，只能原样回传 |

slug 会被 trim 并转为小写，只允许字母、数字及分隔单词的单连字符。JSON 顶层对象之后不能附加第二个 JSON 值。
