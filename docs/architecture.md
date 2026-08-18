# SimpleBlog 程序架构

## 1. 项目目标

SimpleBlog 是一个单站点、多用户的博客后端 API。项目保持小而精，使用 Go、Chi 与 PostgreSQL，按技术概念组织代码，不引入 ORM、Redis、消息队列或框架式依赖注入。

当前角色固定为：

- `admin`：站点管理员，管理用户、分类、全部文章和全部评论。
- `contributor`：内容贡献者，管理并发布自己的文章，管理自己文章下的评论。

正文以 Markdown 原文保存，LaTeX 行内公式与公式块是正文的一部分，由前端负责渲染。图片通过 S3 兼容服务直传，不经过 Go 服务。

相关文档：

- [数据库设计](database.md)
- [权限设计](permissions.md)
- [API 设计](api.md)

## 2. 总体结构

项目保留现有技术分层：

```text
cmd/server/main.go             # 组合根、启动与关闭

internal/config/               # 环境变量和启动校验
internal/model/                # 持久化实体、角色和状态常量
internal/dto/                  # HTTP 输入输出模型
internal/repository/           # PostgreSQL 数据访问
internal/service/              # 业务规则、权限和状态流转
internal/handler/              # HTTP 参数解析与响应
internal/middleware/           # 认证、角色、Turnstile 等中间件
internal/router/               # 公开、认证和后台路由
internal/response/             # 统一 HTTP 200 + body code 响应
```

依赖方向固定为：

```text
router -> middleware / handler -> service -> repository -> PostgreSQL
                                      |
                                      +-> Turnstile / S3
```

约束：

- `handler` 不直接访问数据库。
- `repository` 不依赖 HTTP、DTO 或 response。
- 角色、文章所有权和状态流转必须在 `service` 判断。
- `model` 不直接作为公开响应，防止泄露密码哈希和评论邮箱。
- `main.go` 是唯一组合根，负责显式创建并注入所有依赖。

## 3. 建议文件

```text
internal/model/
  user.go
  category.go
  post.go
  comment.go

internal/dto/
  auth.go
  user.go
  category.go
  post.go
  comment.go
  upload.go

internal/repository/
  user.go
  category.go
  post.go
  comment.go

internal/service/
  auth.go
  user.go
  category.go
  post.go
  comment.go
  upload.go
  turnstile.go

internal/handler/
  auth.go
  user.go
  category.go
  post.go
  comment.go
  upload.go

internal/middleware/
  auth.go
  role.go
  turnstile.go

internal/router/
  public.go
  auth.go
  admin.go
  service.go
```

M1 已将原有 `Admin` 模型、`AdminHandler` 和 `AuthRepository` 统一改为 `User`、`AuthHandler`/`UserHandler` 和 `UserRepository`。认证查询复用 `UserRepository.GetByUsername`，没有创建重复的数据访问实现。

## 4. 核心业务流程

### 4.1 登录与认证

1. handler 解析用户名和密码。
2. AuthService 通过 UserRepository 查询未删除用户。
3. 使用 bcrypt 校验密码。
4. 签发包含 `sub`、`role`、`iat`、`exp` 的 24 小时 JWT。
5. 受保护请求解析 JWT 后再次加载当前用户，确保用户删除或角色变化立即生效。

JWT 中的 role 可以用于前端展示，但 service 以数据库中的当前角色为最终依据。第一版不引入 refresh token。

首次部署通过本地 CLI 创建第一个管理员，不提供可被网络调用的初始化接口：

```powershell
$env:ADMIN_PASSWORD = "replace-with-a-strong-password"
go run ./cmd/admin -username admin -nickname Administrator
Remove-Item Env:ADMIN_PASSWORD
```

CLI 仅在当前不存在有效 admin 时成功；密码必须为 10 到 72 字节。后续用户维护统一使用受 admin 角色保护的 `/admin/users` 路由。

### 4.2 文章生命周期

文章只有 `draft` 与 `published` 两种状态：

- 创建：固定为 draft。
- 编辑：contributor 只能编辑自己的文章；admin 可以编辑全部文章。
- 发布：首次发布写入 `published_at`。
- 撤回：改回 draft，但保留原 `published_at`。
- 再发布：默认保留原发布时间。
- 删除：物理删除文章，并级联物理删除全部评论。

文章 slug 由客户端显式提供，服务端统一转为小写并限制为字母、数字和单连字符；不从标题自动生成拼音或转写。标签在 service 中 trim、转小写、去空、去重并保序，最多 10 个且每项最多 30 个 Unicode 字符。

如果以后要求管理员审核 contributor 文章，应显式增加 `pending_review` 状态，而不是复用 draft。

### 4.3 多级评论

评论使用 `parent_id` 邻接表表达任意层级回复：

- 顶级评论的 `parent_id` 为 NULL。
- 回复指向直属父评论。
- 父评论必须与回复属于同一文章。
- `parent_id` 创建后不可修改，避免环和跨树移动。
- API 可返回平铺数据和 parent_id，或由 CommentService 组装 replies 树；第一版推荐服务端组装树。
- 后端不限制回复深度；前端采用类似 B 站的平铺视觉，不随真实层级持续增加缩进。

隐藏或删除父评论时必须保留树节点：

- hidden：保留原数据，公开端显示“评论已隐藏”占位，后台可以恢复。
- deleted：清空昵称、邮箱和正文，保留关系节点；公开端固定显示“该评论已被删除”，子回复继续展示且不允许恢复。
- 删除文章：数据库级联物理删除整棵评论树。

### 4.4 S3 图片直传

1. 已登录用户请求预签名上传地址。
2. UploadService 校验文件类型、大小并生成 object key。
3. API 返回 upload URL、必要 headers 和 public URL。
4. 前端直接把图片上传到 S3。
5. 前端把 public URL 插入 Markdown 正文。

第一版不建立 uploads 表。对象路径由服务端生成，客户端不能指定 bucket 或任意 key。用于正文的 Bucket 应提供公开或 CDN 读取地址。

首版只允许 JPEG、PNG、WebP、GIF，单文件最大 10 MiB。

## 5. 横切能力

- 配置：生产必需配置不得使用可运行的默认 secret。
- 数据库：启动时验证连接，所有查询传递 request context。
- HTTP：配置读取、写入、空闲超时与优雅关闭。
- 输入：限制 body 大小、文章正文大小、标签数量、评论长度和图片大小。
- 日志：记录 request ID、当前用户 ID、业务 code、耗时和内部错误；不得记录密码、JWT、评论邮箱或 S3 签名。
- 响应：所有应用层业务结果返回 HTTP 200，真实结果由 JSON `code` 表示，详见 API 文档。

## 6. 实施路线

### 阶段 1：数据库基线

- 新建第一份 schema/migration，直接创建 users、categories、posts、comments。
- 当前数据库没有业务表，不做 admins 数据迁移。
- 验证所有外键、CHECK、唯一索引和删除行为。

影响：model、repository、配置与启动流程。

### 阶段 2：用户、认证与权限

- 将 Admin 语义改为 User。
- 实现用户 CRUD、软删除、最后一个 admin 保护。
- 完成 JWT、当前用户加载和权限判断。
- 统一认证中间件的 HTTP 200 响应契约。

重点验证：admin/contributor 权限矩阵、已删除用户、旧 JWT、最后一个 admin。

### 阶段 3：分类和文章后台

- 先实现分类 CRUD。
- 再实现文章 CRUD、发布和撤回动作。
- 完成作者所有权、分类约束和标签规范化。

重点验证：contributor 不能操作他人文章，公开查询不能出现 draft。

### 阶段 4：公开文章 API

- 实现文章列表、详情、分类筛选、标签筛选和分页。
- 公开文章使用 slug，后台管理使用 UUID。

### 阶段 5：多级评论

- 实现顶级评论和回复创建。
- 校验父评论与文章关系。
- 实现树组装、隐藏、恢复和墓碑删除。
- 公开 DTO 排除 email，并对匿名提交启用 Turnstile。

### 阶段 6：S3 直传

- 实现预签名接口和 S3 兼容配置。
- 验证文件不经过 Go 进程。
- 使用真实 S3 兼容服务完成上传和公开读取测试。

### 阶段 7：工程加固

- 补齐分页边界、请求限制、日志、超时和优雅关闭。
- 增加 service 单元测试、handler 测试和 PostgreSQL 集成测试。
- 验证 404、405、panic、middleware 与业务 handler 都遵循统一响应协议。
