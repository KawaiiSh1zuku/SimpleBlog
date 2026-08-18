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
/api/auth       登录和当前用户
/api            公开内容
/api/admin      登录后的后台；contributor 也使用此路径
/service        健康检查等服务端点
```

`admin` 表示后台管理界面，不等同于所有路由都只允许 admin 角色。

## 3. 认证

```text
POST /api/auth/login
GET  /api/auth/me
```

- login 接收 username、password 和 Turnstile token。
- me 返回当前用户的 id、username、nickname、avatar_url 和 role。
- JWT 使用 `Authorization: Bearer <token>`。

## 4. 公开文章与分类

```text
GET /api/posts
GET /api/posts/{slug}
GET /api/categories
GET /api/tags
```

文章列表支持：

- 分页参数。
- category slug 筛选。
- tag 筛选。
- 按 published_at 倒序。

公开端只能返回 published 文章。正文详情返回 Markdown + LaTeX 原文，不返回服务端渲染 HTML。

## 5. 公开评论

```text
GET  /api/posts/{slug}/comments
POST /api/posts/{slug}/comments
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

hidden/deleted 节点返回占位内容，但保留 replies。deleted 节点固定显示“该评论已被删除”。为避免任意深度 JSON 造成前端布局问题，前端可以限制视觉缩进层数，但不能改变真实 parent_id 关系。

## 6. 用户管理

仅 admin：

```text
GET    /api/admin/users
POST   /api/admin/users
GET    /api/admin/users/{id}
PATCH  /api/admin/users/{id}
DELETE /api/admin/users/{id}
PUT    /api/admin/users/{id}/password
```

DELETE 执行软删除。修改密码使用独立路由，避免普通资料更新误改凭据。

## 7. 文章后台

```text
GET    /api/admin/posts
POST   /api/admin/posts
GET    /api/admin/posts/{id}
PATCH  /api/admin/posts/{id}
DELETE /api/admin/posts/{id}
POST   /api/admin/posts/{id}/publish
POST   /api/admin/posts/{id}/withdraw
```

- contributor 列表只返回自己的文章。
- contributor 只能操作自己的文章。
- admin 可以操作全部文章。
- author_id 和 status 不接受普通 PATCH 直接修改。
- 发布和撤回使用显式动作路由。

## 8. 分类后台

仅 admin：

```text
GET    /api/admin/categories
POST   /api/admin/categories
GET    /api/admin/categories/{id}
PATCH  /api/admin/categories/{id}
DELETE /api/admin/categories/{id}
```

存在关联文章的分类不能删除，返回业务 code 409。

标签不存在后台 CRUD；标签随文章创建或更新请求整体提交。

## 9. 评论后台

```text
GET    /api/admin/comments
POST   /api/admin/comments/{id}/hide
POST   /api/admin/comments/{id}/show
DELETE /api/admin/comments/{id}
```

- contributor 只管理自己文章下的评论。
- admin 管理全部评论。
- hide 可恢复。
- DELETE 写入 deleted 墓碑并清除个人信息，不物理删除回复树；公开端显示“该评论已被删除”。

## 10. S3 图片直传

```text
POST /api/admin/uploads/presign
```

请求只提供文件元信息，例如原始文件名、Content-Type 和大小。响应提供：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "object_key": "posts/user-id/uuid.webp",
    "upload_url": "https://s3.example/...",
    "headers": {
      "Content-Type": "image/webp"
    },
    "public_url": "https://cdn.example/posts/user-id/uuid.webp"
  }
}
```

前端直接向 upload_url 上传，文件内容不经过 Blog API。

## 11. 服务端点

```text
GET /service/ping
```

后续部署需要时可以拆分存活与就绪检查，但第一版保留单个轻量端点即可。
