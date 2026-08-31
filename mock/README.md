# SimpleBlog Mock Server

`mock` 是与当前 SimpleBlog HTTP API 对齐的内存服务，用于公开站点和管理后台联调。它不需要 PostgreSQL、S3、JWT 密钥或 Cloudflare Turnstile，进程重启后恢复固定种子。

## 启动

在仓库根目录运行：

```powershell
go run ./mock
```

默认地址为 `http://localhost:8090`。可通过 `MOCK_ADDR` 修改监听地址：

```powershell
$env:MOCK_ADDR = ':18090'
go run ./mock
```

种子里的绝对图片 URL 固定使用 `http://localhost:8090/mock-assets/...`。需要验证图片展示时请保留默认端口。

## 测试账号

登录仍需发送任意非空的 `X-Turnstile-Response`，例如 `mock`。

| 用户名 | 密码 | 角色 | 固定 Token |
|---|---|---|---|
| `admin` | `admin123456` | admin | `mock-admin-token` |
| `writer` | `writer123456` | contributor | `mock-contributor-token` |
| `editor` | `editor123456` | contributor | 登录响应返回 |

可直接访问后台接口：

```text
Authorization: Bearer mock-admin-token
```

## 固定数据

- 10 篇文章：7 篇 `published`、3 篇 `draft`。
- 每篇文章的评论数互不相同，完整覆盖 0 到 9 条，共 45 条。
- 评论覆盖 `visible`、`hidden`、`deleted`，同时包含顶级评论和嵌套回复。
- 3 个用户覆盖 `admin` 和 `contributor`；contributor 只能管理自己的文章及其评论。
- 5 个分类、5 条友链，包含空 favicon、长描述等展示边界。
- 文章覆盖有/无 `cover_img`、正文首图、空标签、多标签、Markdown、LaTeX 和原生 HTML。
- cover、头像、favicon 由 `/mock-assets/{name}` 动态生成 PNG，不依赖外网。

公开接口只返回已发布文章；草稿及草稿评论只能从后台看到。所有 JSON 业务响应保持生产契约：HTTP 状态为 200，真实结果读取响应体的 `code`。RSS 和 CORS 预检仍是例外。

## 已实现路由

| 分组 | 路由 |
|---|---|
| 服务 | `GET /service/ping`、`GET /service/test_turnstile` |
| 认证 | `POST /auth/login`、`GET /auth/me` |
| 公开内容 | `GET /posts`、`GET /search`、`GET /posts/{slug}`、`GET/POST /posts/{slug}/comments` |
| 公开元数据 | `GET /categories`、`GET /tags`、`GET /links` |
| 用户后台 | `/admin/users` 全套 CRUD、密码更新 |
| 内容后台 | `/admin/categories`、`/admin/links`、`/admin/posts` 全套 CRUD，文章发布/撤回 |
| 评论后台 | `GET /admin/comments`，评论隐藏、显示、软删除 |
| 上传 | `POST /admin/uploads/presign`、`POST /mock-upload` |
| RSS | `GET /rss/article`、`GET /rss/comment` |

列表接口支持现有 `limit` 和不透明 `cursor`。公开文章还支持 `category`、`tag` 筛选，`/search` 对标题、摘要和正文执行大小写不敏感的包含搜索。后台写操作会在当前进程内真实生效。

## 与生产服务的差异

- Bearer token 是可读的 mock token，不是真实 JWT，也没有到期校验。
- Turnstile 只检查请求头非空，不访问 Cloudflare。
- 不执行限流，不连接 PostgreSQL，不持久化变更。
- 预签名上传接收并丢弃 multipart 内容，返回的 `public_url` 指向本机生成的占位 PNG。
- 输入校验覆盖主要字段、URL、权限、唯一性和状态转换，但不应替代生产服务的集成测试。

## 验证

```powershell
go test ./mock -count=1
go test -race ./mock -count=1
```
