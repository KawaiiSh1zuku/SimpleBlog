# SimpleBlog 架构勘察发现

> 本文件记录从仓库读取到的事实、推断及其证据。

## 仓库现状

- 代码量很小：一个 `cmd/server` 入口，`internal` 下按 config/router/middleware/handler/service/repository/dto/model/utils 分层，另有通用 `pkg/response`。
- 依赖核心为 Chi 5、pgx 5、jwtauth、bcrypt、环境变量配置和 slog/tint；没有 ORM、DI 框架、迁移工具或验证库，符合“小而精”的倾向。
- 当前功能只有健康/连通性探针、Turnstile 测试和管理员登录；尚无文章、分类/标签、公开读取等博客核心功能。
- 没有 README、`docs/`、迁移 SQL、测试、容器/部署文件。
- 工作区已有用户未提交改动：认证 DTO/handler/middleware/router/service 等正在调整；本轮必须以当前工作树为分析对象并避免覆盖。

## 当前请求链路

1. `cmd/server/main.go` 加载日志与环境配置，创建 pgxpool、Chi router 和各层依赖。
2. `/service` 挂载 ping 和 Turnstile 测试；`/admin` 仅挂载登录。
3. 登录链路为 `router -> AdminHandler -> AuthService -> AuthRepository -> admins`。
4. `AuthService` 用 bcrypt 校验，并用 HS256 JWT 返回 `sub` 与 `role` claims。
5. Turnstile 由 middleware 调用 service，再请求 Cloudflare siteverify。

## 初步风险/不完整点

- 数据库构造后未 `Ping`，创建连接池失败仅记录日志而不退出，随后仍可能注入空 pool；数据库 URL 也未做安全编码或 TLS 参数设计。
- 使用 `http.ListenAndServe`，缺少读写/空闲超时、优雅关闭和请求级超时。
- JWT 没有 `exp`/`iat`/`iss`，也尚未挂载验证中间件；配置给生产秘密提供了不安全默认值。
- 登录 JSON 没有限制 body 大小、未知字段或必填校验；仓库错误被统一映射成 401，会掩盖数据库故障。
- `response.Error` 无论传入何种状态都返回 HTTP 200，与注释和 handler 调用方式容易产生误解，也削弱 HTTP 语义。
- Turnstile `http.Client` 没有超时；传入 `r.RemoteAddr` 可能包含端口；未校验 expected hostname/action。
- 暂未发现博客领域模型与内容发布状态设计，因此不能把现有 admin/auth 骨架误当成完整博客架构。

## 版本与工程状态

- Git 只有 3 个提交：初始化、`/service/ping`、Turnstile；当前工作树正在把登录 DTO 独立出来并把 `service/admin.go` 改名为 `service/auth.go`。
- `.github/copilot-instructions.md` 为空；根 `.gitignore` 只忽略 `.env`，IDE 项目文件已被跟踪。
- 本地存在 `.env`，为避免泄露，本轮不读取其内容；设计必须把敏感配置的启动校验与样例配置分开。
- 当前分层职责总体可辨识，但按“全局技术层”拆包会让每个领域功能横跨 5 个目录；对只有博客与认证两个领域的小项目，后续功能增长时可读性未必最佳。

## 当前可验证性与依赖能力

- `gofmt -d cmd internal pkg` 通过，无格式差异。
- `go test ./...` 与 `go vet ./...` 通过，但所有包都显示 `[no test files]`，所以只证明当前代码可编译和通过基础静态检查。
- 已按当前依赖版本查询接口：`jwtauth.Verifier` 只解析并写入上下文，仍需 `Authenticator` 或自定义鉴权中间件；`pgxpool.Pool.Ping(ctx)` 可用于启动/就绪检查。
- Chi 当前版本提供 RequestID 和 Timeout；其 `RealIP` 已因任意信任转发头存在 IP spoofing 风险而弃用。现有代码使用的 `ClientIPFromHeader` 也必须只在可信反向代理确实覆盖该头时启用。

## 目标范围假设（设计时必须显式说明）

- 推荐基线按“单站点、单管理员/单作者、Markdown 文章、前后台分离但同站部署”设计。
- MVP 必需：登录、文章草稿/发布、公开文章列表与详情、管理端 CRUD、健康检查。
- 默认不做：访客注册、评论、点赞、收藏、搜索、对象存储上传、分类/标签、刷新令牌、多角色 RBAC、消息队列、Redis、ORM。
- 如果用户实际需要多作者、跨域第三方客户端或评论系统，认证、权限、数据模型都会显著变化，应在实施前确认。

## 整体结构决策

- 方案 A：保留当前全局技术分层。迁移成本最低、目录直观，但新增一个功能会同时修改 router/handler/service/repository/dto/model 多处，容易产生只有转发作用的薄层。
- 方案 B：按领域组织轻量模块。`auth` 与 `post` 各自内聚 HTTP、业务和存储代码；config/database/httpserver 等放基础设施层。模块少、依赖方向清楚，更适合本项目，推荐采用。
- 推荐不是完整 DDD，也不引入框架式 DI：`main` 仍是唯一组合根，显式构造依赖；只在需要替换以便测试的边界定义小接口。
- 请求依赖方向固定为 `HTTP -> application/service -> repository -> PostgreSQL`，领域代码不得反向依赖 router、具体 HTTP 响应或全局变量。
- 核心领域仅 `auth` 与 `post`；Turnstile 属于认证的外部防护能力，health 属于基础设施，不应继续作为独立的业务 service 模块扩张。

## 最小数据与生命周期

- `admins`：保留账号、密码哈希、展示信息；单管理员基线下不需要复杂 RBAC，`role` 是否保留取决于是否明确支持多作者。
- `posts`：`id`、`slug`、`title`、可选 `summary`、Markdown `content`、`status`（draft/published）、`published_at`、`created_at`、`updated_at`。
- 发布规则集中在 post service：公开读取只返回 published；首次发布写入 `published_at`；撤回改为 draft；slug 唯一；删除使用硬删除，暂不引入软删除。
- 标签、分类、图片上传、评论、全文搜索不是 MVP 表，待真实需求出现后再加，避免为假想扩展预建抽象。

## 横切能力边界

- 配置：启动时校验生产必需项；数据库优先单一 DSN；秘密不能使用可运行的默认值。
- HTTP：统一真实 HTTP 状态码、错误模型、请求体限制、panic 恢复、request ID、访问日志、服务器超时和优雅关闭。
- 认证：登录保护、短时效 JWT 或同站安全 Cookie 二选一；不同时维护 access/refresh token 体系。
- 数据库：显式迁移文件、启动连接校验、所有调用传递 request context；无跨多表一致性需求时不滥用事务。
- 可观测性：沿用 slog；开发文本日志、生产结构化日志；内部错误写日志，外部返回稳定且不泄露细节的错误。
- 测试：优先覆盖发布状态规则、认证失败分类、HTTP 错误映射和 repository 的关键 SQL；不追求形式化的全层 mock。

## 2026-08-18 需求修订

- 用户确认保留 `router/handler/service/repository/dto/model` 的技术分层结构，因此此前“改为领域模块化”的建议作废。
- `response` 已从 `pkg/response` 移至 `internal/response`，符合仅供本程序使用的边界；引用方也在同步调整。
- 产品改为单站点多用户，角色固定为 `admin` 与 `contributor`。
- 文章正文将 Markdown 与 LaTeX 公式块作为同一份原始文本存入 PostgreSQL，不需要拆分公式表或后端预渲染。
- 必须设计文章完整生命周期、用户 CRUD、匿名评论与后台审核、分类外键、文章内嵌标签、S3 兼容直传。
- 当前业务工作树仍有用户未提交改动，本轮仅更新规划记录，不修改业务代码。

## 数据模型定案

- 将 `admins` 重命名为 `users`。角色使用 `TEXT/VARCHAR + CHECK (role IN ('admin','contributor'))`，不使用 PostgreSQL enum，便于未来迁移。
- `users` 使用软删除 `deleted_at` 保留文章作者关系；认证和默认用户查询必须排除已删除用户。最后一个 admin 不允许删除或降级。
- `posts.author_id -> users.id ON DELETE RESTRICT`；`posts.category_id -> categories.id ON DELETE RESTRICT`；文章硬删除时 `comments.post_id -> posts.id ON DELETE CASCADE`。
- 文章正文使用单个 `content TEXT` 保存 Markdown 与 LaTeX 原文；后端只做长度/空值校验，不解析、不拆公式表。
- 标签按用户要求作为文章属性，使用 `posts.tags TEXT[]`；service 负责 trim、去重、数量和单项长度限制，数据库建立 GIN 索引。
- 文章只有 draft/published 两态：首次发布写 `published_at`，撤回保留原发布时间但公开查询按 status 过滤。若要审核流，必须显式增加 pending_review。
- 匿名评论保存 nickname/email/content/is_hidden/created_at；公开 DTO 永不返回 email。评论默认公开，可隐藏/恢复，删除为硬删除。
- S3 直传暂不建立 uploads 表；后端只签发受限的 presigned URL，并返回 object_key/public_url。对象命名由服务端生成，客户端不能指定 bucket 或任意 key。

## 推荐权限矩阵

- 访客：读取已发布文章/分类/标签/可见评论，向已发布文章提交匿名评论。
- contributor：管理自己的文章并发布/撤回；选择分类、维护文章 tags；获取 S3 预签名地址；管理自己文章下的评论。
- admin：拥有 contributor 全部能力，并管理所有文章、全部评论、分类和用户。
- 角色/所有权判断必须在 service 层执行；handler 只解析参数，repository 只处理数据访问。不启用 PostgreSQL RLS。

## 目录内职责

- `model`：User/Post/Category/Comment 持久化实体与角色、状态常量。
- `dto`：公开响应、管理响应和写请求；禁止直接序列化 model，尤其不能暴露 password_hash/comment.email。
- `repository`：auth/user/post/category/comment 五组 PostgreSQL 查询。
- `service`：认证、权限、文章状态机、分类约束、标签规范化、评论审核和上传签名。
- `handler`：按资源拆分 Auth/User/Post/Category/Comment/Upload handler，避免单个 AdminHandler 膨胀。
- `middleware`：JWT 解析、当前用户加载、角色守卫、Turnstile；用户 CRUD 场景下受保护请求应查询当前用户状态，避免删除/改角色后旧 JWT 长时间有效。
- `router`：公开 API、认证 API、管理 API 分组；管理路径是后台入口，不等同于只有 admin 可访问。

## 实施阶段

1. 建立 migrations，创建 users/categories/posts/comments 和索引；迁移现有 admins 数据。
2. 将 Admin 语义改为 User，完成登录、JWT、当前用户与 role/owner 权限中间件。
3. 实现分类 CRUD，再实现文章 CRUD 与 draft/publish/withdraw 状态动作。
4. 实现公开文章列表/详情、分类与标签筛选。
5. 实现匿名评论、Turnstile、防输入滥用及后台隐藏/恢复/删除。
6. 实现 S3 presign；上传内容不经过 Go 服务，不建媒体表。
7. 补齐错误映射、分页、请求限制、日志、测试、迁移验证和文档。

## 2026-08-18 最终修订

- 评论改为多级回复：comments 增加 parent_id 自引用，采用邻接表；父评论和回复必须属于同一文章，parent_id 创建后不可修改。
- 单独删除评论采用墓碑语义而非物理删除：状态改为 deleted 并清空昵称、邮箱、正文，公开端显示“该评论已被删除”，保留节点与后代回复；删除文章时才级联物理删除整棵评论树。
- 当前 PostgreSQL 尚未创建业务表，因此不需要 admins -> users 数据迁移；首份 schema/migration 直接创建 users。
- 用户确认业务异常继续使用 HTTP 200，响应体 code 为真实业务状态；文档需要求所有 handler/middleware/404/405/recoverer 统一遵循此契约。
- 用户明确要求同步到 docs；将创建 architecture.md、database.md、permissions.md、api.md。

## Milestone 执行约束

- 用户批准按 M0-M7 顺序实施；一次只完成一个 milestone，完成后提交并等待用户验收/签名确认。
- 评论后端不限制回复深度；数据仍使用 parent_id 邻接表，前端采用不随层级增加缩进的展示。
- S3 首版限制已确认：JPEG、PNG、WebP、GIF，单文件最大 10 MiB。
- 当前工作区在实施前为 clean，Git 已配置 commit.gpgsign=true 和 signing key；从 `codex/simpleblog-milestones` 分支开始实施。

## M0 实现勘察

- response 的公开调用目前只有 Success/Error 包装函数，没有代码直接调用底层 JSON，因此可以把“HTTP 永远 200”收口到 response.JSON 而不破坏现有调用方。
- 配置结构仅在 `cmd/server/main.go` 使用，适合在 Config.Load 后集中 Validate，并为 DatabaseConfig 增加安全构造连接字符串的方法。
- 根 Chi router 目前在 main 内创建，NotFound/MethodNotAllowed/panic 无法统一测试；M0 将增加 `router.New` 作为根路由装配函数。
- 本机没有 psql 或 Docker，只能先通过静态检查与可选 PostgreSQL 集成测试验证 schema；若本地测试数据库可用，再执行事务内 schema 验证。
- 现有 docs/database.md 已包含最终四表 DDL，M0 的 `migrations/000001_init.sql` 应与该设计保持一致。
- PostgreSQL 18 本地实例可用；初始 SQL 已在随机 schema 的事务中完整执行，四表创建与“删除文章级联删除父子评论”均通过，随后回滚且未留下对象。
- 应用通过唯一的 response.JSON 写入 HTTP status；静态扫描未发现 http.Error 或其他直接 WriteHeader 调用。

## M1 实现决策

- 现有认证代码仍查询 `admins.avatar_img`，与 M0 的 `users.avatar_url/deleted_at/timestamps` schema 不一致；M1 必须一次性完成模型、SQL、路由和构造函数重命名。
- 使用同一个 UserRepository 服务登录、当前用户加载和用户 CRUD；不保留独立 AuthRepository。
- jwtauth v5 提供 SetIssuedNow/SetExpiry，但没有 SetIssuer helper；issuer 直接写入 claims。
- JWT role 只用于客户端展示；Authenticate middleware 从 sub 解析 UUID 后查询未删除 users，并把最新 User 放入 context。
- 最后一个 admin 的删除/降级需要跨请求串行化；推荐由 UserRepository 在事务内获取 PostgreSQL advisory transaction lock，再检查并更新。
- 用户列表规模很小，M1 不引入分页；密码使用独立更新动作，任何 DTO 都不返回 password_hash/deleted_at。
- 首个 admin 使用本地 CLI 和 ADMIN_PASSWORD 环境变量创建，不开放匿名 bootstrap HTTP 路由，也不新增终端密码依赖。
- 用户确认这是纯后端项目，所有资源路由不使用 `/api` 前缀；M1 使用 `/auth` 和 `/admin`，后续公开资源直接使用 `/posts`、`/categories` 等顶层路径。
- `/admin` 是后台命名空间而非 admin 角色命名空间；JWT 与当前用户校验作用于整个路由器，admin 角色守卫只包围 `/admin/users`，为 M2 的 contributor 文章管理保留正确边界。
- 登录首版不额外引入 Turnstile；匿名评论在 M4 使用现有 Turnstile 能力，登录滥用防护留在 M6 统一处理。

## M2 实现决策

- JWT 默认有效期按用户要求从 2 小时调整为 24 小时；当前用户和角色仍在每个受保护请求中重新加载，因此软删除和降权立即生效。
- 分类为全局资源，仅 admin CRUD；删除关联文章的分类由 PostgreSQL 外键原子拒绝，repository 将 SQLSTATE 23503 映射为业务冲突。
- 文章所有权集中在 PostService；author_id 创建后不可修改。contributor 列表在 SQL 中按 author_id 过滤，admin 查看全部。
- slug 由客户端显式提供并规范为小写 URL-safe 连字符格式，不从中英文标题自动转写，避免额外依赖和隐式规则。
- 标签继续存入 posts.tags TEXT[]；service 负责 trim、英文小写、去空、去重、保序，最多 10 个且每个最多 30 个 Unicode 字符。
- 发布/撤回使用独立动作：首次发布写 published_at，撤回保留 published_at，再发布不刷新原发布时间；普通 PATCH 不能修改 status、author_id 或 published_at。

## M3 实现决策

- 公开文章分页采用 `(published_at, id)` 降序游标，而不是 offset/page；新发布内容不会使后续页出现重复或边界漂移。
- 游标是 Base64URL 编码的版本化 JSON，客户端只透传；服务端严格校验版本、时间和 UUID，不把数据库结构暴露为多个查询参数。
- 列表默认 20 条、最大 100 条，repository 查询 `limit + 1` 判断是否存在下一页；响应只在确有下一页时返回 next_cursor。
- 所有公开文章 SQL 必须直接包含 `status = 'published'`；列表不返回 content，详情才返回 Markdown + LaTeX 原文。
- 公开文章响应包含作者展示信息和分类摘要，不返回用户名、密码、文章后台状态或其他内部字段。
- category 按规范化 slug 筛选，tag 按规范化后的单标签精确匹配；分类不存在时列表返回空集合，文章详情 slug 不存在时返回 404。

## M6 预先修订

- 用户要求 M6 增加 PostgreSQL 深度优化。验收不能只写“优化 SQL”，必须以关键查询的执行计划、缓冲区、索引命中、N+1、慢查询、锁竞争和参数化检查为证据。
- 安全优化同时要求最小字段读取、敏感列不进入非必要查询、所有外部输入参数化；性能优化必须基于代表性数据，不以小空表上的耗时作为结论。

## M4 实现决策

- 匿名评论创建使用已发布文章 `INSERT ... SELECT`，与 comments 的 `(parent_id, post_id)` 复合外键共同保证发布状态和同文章父子关系，避免“先查再插”的竞态。
- 公开评论查询只选择展示所需字段，刻意不读取 email；后台管理查询才读取 email，并按文章作者过滤 contributor。
- 删除评论只写 deleted 墓碑，固定昵称“该评论已被删除”，同时清空 email/content；hidden 可恢复，deleted 不可恢复，hidden/deleted 都允许继续被回复。
- 评论树通过两遍迭代映射组装，不按层级递归查询数据库，因此回复深度不产生 N+1；前端仍可采用不随深度缩进的 B 站式展示。
- Turnstile 中间件改为小接口以便测试，并只传不含端口的远端 IP；HTTP client 增加 5 秒超时。
