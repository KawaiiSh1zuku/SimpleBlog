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

- login 接收 username 和 password，成功后签发 2 小时有效的 JWT。
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

- 分页参数。
- category slug 筛选。
- tag 筛选。
- 按 published_at 倒序。

公开端只能返回 published 文章。正文详情返回 Markdown + LaTeX 原文，不返回服务端渲染 HTML。

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

- contributor 列表只返回自己的文章。
- contributor 只能操作自己的文章。
- admin 可以操作全部文章。
- author_id 和 status 不接受普通 PATCH 直接修改。
- 发布和撤回使用显式动作路由。

## 8. 分类后台

仅 admin：

```text
GET    /admin/categories
POST   /admin/categories
GET    /admin/categories/{id}
PATCH  /admin/categories/{id}
DELETE /admin/categories/{id}
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

首版上传限制：

- MIME：`image/jpeg`、`image/png`、`image/webp`、`image/gif`。
- 单文件最大 10 MiB。
- 服务端生成 object key，不接受客户端提供任意对象路径。

## 11. 服务端点

```text
GET /service/ping
```

后续部署需要时可以拆分存活与就绪检查，但第一版保留单个轻量端点即可。
