# SimpleBlog API 设计

## 1. 响应契约

所有由应用程序处理并返回的业务结果统一使用 HTTP 200。真实结果由 JSON `code` 字段表示：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

失败示例：

```json
{
  "code": 403,
  "msg": "permission denied"
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
| 422 | 请求格式正确但业务校验失败 |
| 500 | 未预期的服务端错误 |

前端必须始终读取 `code`，不能只根据 HTTP 状态判断。以下路径也必须使用相同 JSON 格式和 HTTP 200：

- JWT/角色中间件拒绝。
- Turnstile 拒绝。
- Chi NotFound 与 MethodNotAllowed。
- panic 恢复。
- 参数解析和 body 大小限制。

反向代理、网络中断和进程崩溃不属于应用可控制的业务响应，仍可能产生真实非 200 或无响应。日志和监控必须按 body code/结构化日志统计错误，不能只统计 HTTP 状态。

## 2. 路由分组

```text
/auth           登录和当前用户
/               公开内容
/admin          登录后的后台；contributor 也使用此路径
/service        健康检查等服务端点
```

`admin` 表示后台管理界面，不等同于所有路由都只允许 admin 角色。

## 3. 认证

```text
POST /auth/login
GET  /auth/me
```

- login 接收 username 和 password，成功后签发 24 小时有效的 JWT。
- me 返回当前用户的 id、username、nickname、avatar_url 和 role。
- JWT 使用 `Authorization: Bearer <token>`。

## 4. 公开文章与分类

```text
GET /posts
GET /posts/{slug}
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
- Turnstile 请求头固定为 `X-Turnstile-Token`；缺失或验证失败分别返回业务 code 401/403。
- nickname 最大 80 个 Unicode 字符，email 必须是单一有效邮箱地址，parent_id 必须属于同一篇文章。
- 允许回复 hidden/deleted 节点；回复仍保留真实 parent_id。

评论响应推荐返回树结构：

```json
{
  "id": "uuid",
  "parent_id": null,
  "nickname": "visitor",
  "content": "comment content",
  "status": "visible",
  "created_at": "2026-08-18T12:00:00Z",
  "replies": []
}
```

hidden/deleted 节点返回占位内容，但保留 replies。deleted 节点固定显示“该评论已被删除”。后端不限制回复深度；前端采用类似 B 站的不随层级增加缩进的展示，但不能改变真实 parent_id 关系。

hidden 节点公开显示“评论已隐藏”且不返回原正文；deleted 节点不可恢复。评论树由后端按 parent_id 组装，任意层级回复都保留在 `replies` 中。

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
- tags 会被 trim、转小写、去空、去重并保序，最多 10 个，每项最多 30 个 Unicode 字符。
- content 原样保存 Markdown 与 LaTeX 文本，不在服务端渲染。

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
- 后台列表包含 email 供站点管理，但公开 DTO 和公开查询均不读取或返回 email。
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
    "object_key": "posts/user-id/uuid.webp",
    "upload_url": "https://s3.example/...",
    "fields": {
      "key": "posts/user-id/uuid.webp",
      "Content-Type": "image/webp",
      "policy": "base64-policy",
      "x-amz-signature": "signature"
    },
    "public_url": "https://cdn.example/posts/user-id/uuid.webp",
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
- policy 同时约束精确 Content-Type 和请求声明的精确字节数，签名 10 分钟后过期。
- filename 仅作元数据校验，不参与 object key；服务端按 MIME 决定扩展名。
- admin 与 contributor 均可申请签名，未登录请求返回业务 code 401。

## 11. 服务端点

```text
GET /service/ping
```

后续部署需要时可以拆分存活与就绪检查，但第一版保留单个轻量端点即可。
