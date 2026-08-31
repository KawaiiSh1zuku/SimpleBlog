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
- M4 初版省略了公开查询中的 email 字段；M6 按新需求改为读取 email 仅用于生成 Gravatar URL，公开 DTO 和日志仍不包含 email。后台管理查询继续按文章作者过滤 contributor。
- 删除评论只写 deleted 墓碑，固定昵称“该评论已被删除”，同时清空 email/content；hidden 可恢复，deleted 不可恢复，hidden/deleted 都允许继续被回复。
- 评论树通过两遍迭代映射组装，不按层级递归查询数据库，因此回复深度不产生 N+1；前端仍可采用不随深度缩进的 B 站式展示。
- Turnstile 中间件改为小接口以便测试，并只传不含端口的远端 IP；HTTP client 增加 5 秒超时。

## M5 实现决策

- 预签名 PUT 不能可靠地在对象存储侧强制实际文件大小；M5 改用 S3 POST policy，以 `content-length-range` 的相同上下界校验声明大小，并精确约束 Content-Type。
- object key 固定为 `posts/{userID}/{uuid}.{server-extension}`；原始 filename 不进入路径，扩展名由 MIME 映射，避免客户端控制 bucket、目录或可执行后缀。
- 签名有效期固定 10 分钟；admin/contributor 都可申请，UploadService 仍执行角色防御，不只依赖路由中间件。
- Blog API 只返回 upload URL、必须原样提交的 form fields 和公开 URL，不代理文件字节，也不建立 uploads 表。
- MinIO Go v7 仅封装 S3 兼容签名；真实验证使用临时本地 MinIO，完成 multipart POST 与匿名公开读取后销毁测试 bucket 和临时服务。

## M6 新增需求

- 用户允许公开评论查询在服务端读取 email，只要求前端响应不返回 email，因此不新增 email_hash 列或迁移。
- Gravatar URL 由规范化邮箱（trim + 小写）的 MD5 生成；MD5 仅用于 Gravatar 标识，不用于密码或安全校验。
- 镜像源通过 `GRAVATAR_BASE_URL` 配置，公开 DTO 只增加 avatar_url；hidden/deleted 占位节点不返回头像。

## M6 初步审计

- 公开文章列表复用详情 SELECT，虽然 DTO 不返回正文，SQL 仍读取大字段 content、后台状态和多余时间字段；应拆分 list/detail 投影。
- UserRepository 的 GetByID/List/写入 RETURNING 都读取 password_hash，但只有登录 GetByUsername 需要它；应提供安全投影，减少密码哈希进入内存。
- 评论公开查询当前不读 email；按用户最新确认可读取 email 并仅在 DTO 内生成 Gravatar URL，静态测试必须确认 JSON 无 email 字段。
- 所有业务 SQL 的值均使用 PostgreSQL 参数；动态列表查询只拼接服务端生成的占位符和固定片段，未发现外部输入直接拼入 SQL。
- `LOWER(slug)` 唯一索引与多个 `slug = $1` 查询表达式不一致，可能无法利用索引；评论索引把 parent_id 放在 created_at 之前，也不能直接满足按创建时间排序。
- 管理端文章/评论列表当前无分页且读取完整集合；公开文章已有 1..100 的游标分页。M6 至少需要请求体上限、明确分页上限并对高增长查询给出优化证据。
- 当前错误日志直接记录数据库 error，PostgreSQL Detail 可能包含输入值；M6 应只记录错误类型、SQLSTATE、约束和请求元数据。
- 根路由已有 RequestID、IP 解析和 panic recovery，但没有结构化访问日志或请求体大小限制。

## M6 实现决策

- 请求体统一限制为 2 MiB，并由 MaxBytesReader 覆盖未知 Content-Length；业务字段再限制文章正文 1 MiB、摘要 1000 字符、评论 10000 字符、分类描述 2000 字符、头像 URL 2048 字符。
- 访问日志通过 response writer 接收真实 body code，记录 path 而非 raw URL，因此不记录 query；错误日志对 PostgreSQL 只保留 SQLSTATE/constraint/type，不输出 Message/Detail。
- PostgreSQL 错误集中分类 unique、foreign key、input violation 与 retryable；可重试的序列化失败、死锁、锁不可用和连接耗尽映射为业务 code 503。
- 公开文章列表与详情拆分 SQL 投影，列表不再读取 Markdown content；UserRepository 仅登录查询读取 password_hash，当前用户、列表和写入返回均使用安全投影。
- 管理文章和评论使用带资源 kind 的 Base64URL keyset cursor，默认 50、最大 100；文章列表使用无 content 的 PostSummary，编辑器按 ID 单独获取正文。
- 登录采用单进程 IP 固定窗口限制（10 次/分钟、最多跟踪 10000 个 IP）；匿名评论继续使用 Turnstile。单站点单实例不引入 Redis。
- `X-Real-IP` 不再由 DEBUG 隐式决定；仅 `TRUST_PROXY_HEADERS=true` 时信任，且部署代理必须覆盖客户端同名请求头。
- 第二份迁移将后台文章、作者文章、已发布标签、文章评论时间序和全局评论时间序分别建立与查询一致的索引，并移除首份 schema 中前导列不匹配的索引。
- 慢查询追踪以 `DB_SLOW_QUERY_MS` 配置阈值，日志只写规范化 SQL 的 SHA-256 短指纹、耗时、command tag 和 SQLSTATE；不写 SQL 文本、参数或 PostgreSQL Detail。
- 初轮 PostgreSQL 18 数据验证覆盖 10k users、1k categories、20k posts、50k comments；八条基础查询均命中指定索引，测试未发现高增长表的可避免顺序扫描。
- 补充 contributor 评论计划后发现全局时间索引会为 51 条结果扫描约 5,001 条评论并做同数量文章查找；采用物化作者文章集合 + LATERAL 后改为作者/文章评论索引，执行时间由样本 8.520 ms 降至 2.054 ms。

## M8 初始需求与决策点

- 用户要求公开 RSS 路由固定为 `/rss/article` 和 `/rss/comment`，但 feed 的站点绝对 URL、条目数量、评论可见状态仍需确认。
- RSS 2.0 可直接使用标准库 `encoding/xml`，无需引入第三方依赖；XML 文本必须由编码器转义，不能拼接用户内容。
- 友情链接字段为名字、简介、链接、favicon 地址；尚未明确是只读配置还是 PostgreSQL 持久化 + admin CRUD。对当前多用户后台，推荐公开 GET + admin CRUD。
- SQL 注入防护应延续 pgx 参数化值；动态排序/条件只能由服务端固定枚举生成，并加入恶意输入回归测试与源码审计。
- 后端当前返回 Markdown 原文。JSON 编码不是最终 HTML 安全边界：前端解析 JSON 后仍可把 `<script>`、事件属性或危险 URL 渲染进 DOM。若要后端独立保证 XSS，需要拒绝原生 HTML/危险链接，或改为后端渲染并清洗 HTML。
- 用户确认文章内可执行 HTML 是预期能力，不能清洗文章正文；XSS 防护范围聚焦匿名评论。推荐把评论定义为纯文本，拒绝 HTML 标记和危险控制字符，JSON/RSS 使用标准编码器转义，并要求前端使用文本节点渲染。
- 上一轮文档变更在本轮开始前已处于 staged 状态；本轮必须保留该状态且不提交。
- SQL 源码复核未发现用户值直接拼入语句：动态文章/评论查询只拼接服务端固定子句和由参数数量生成的 `$n` 占位符，实际值都经 pgx args 传入。M8 新 SQL 必须保持此模式并增加恶意 name/url/filter 输入测试。
- 当前评论 Create 接受任意 HTML 字符串并原样存储，公开/后台 DTO 也原样返回；`encoding/json` 会保护 JSON 文本本身，但前端解析后若使用 `innerHTML` 仍可执行。评论纯文本校验应在 service 写入边界完成，历史数据还需在 DTO/RSS 输出边界防御。
- M8 可同时修复已记录的评论首尾空白长度偏差：将 content 规范化一次后校验并存储，避免校验值与存储值不同。
- 标准 RSS 响应必须使用 XML 而非统一 JSON envelope；为兼容订阅器，推荐把 RSS 端点作为现有“HTTP 200 + body code”协议的显式例外，并在内部错误时返回真实 5xx。
- 用户已批准推荐方案。RSS 使用 `SITE_NAME`、`SITE_DESCRIPTION`、`SITE_BASE_URL`，文章链接为 `{base}/posts/{slug}`，最新 50 条；评论仅 visible 且文章 published。
- 友情链接采用 `friend_links` 表，公开 `GET /links`，仅 admin 可在 `/admin/links` CRUD；不增加 sort_order，按名称与 ID 稳定排序。
- 评论昵称和正文按纯文本处理；新增输入拒绝 `<`/`>` 和危险控制字符，输出层还要处理迁移前历史脏数据。文章正文与摘要不做 HTML 清洗。
- M8 已按技术分层加入 `friend_links`、RSS feed 投影及独立 repository/service/handler/router；没有新增第三方依赖，RSS 由标准库 `encoding/xml` 编码。
- 新评论会统一换行、trim 后保存，并拒绝尖括号与危险控制字符；DTO/RSS 对迁移前旧数据把尖括号替换为全角字符并剔除控制字符，文章详情正文不受影响。
- 真实 PostgreSQL 注入测试将 `x'); DROP TABLE friend_links; --` 作为参数写入并原样读回，`friend_links` 表保持存在；新增 SQL 均使用 pgx 占位参数。
- 50k comments 的 RSS 查询命中 `comments_visible_created_idx`，本次 PostgreSQL 18 执行时间约 0.39 ms；这是计划形状证据而非生产 SLA。

## 2026-08-19 访问日志留存

- 现有 `AccessLog` 使用全局 `slog` 输出 `request completed`，包含 request_id、method、path、remote_ip、business_code、duration_ms、response_bytes。
- 采用独立访问日志 writer：终端继续走全局 tint 完整运行日志，访问记录额外写入 `logs/access-YYYY-MM-DD.log`，避免把所有运行错误与访问记录混入归档。
- `ACCESS_LOG_RETENTION_DAYS` 默认 30，语义为包含当天的自然日总数；范围建议 1–3650。
- 轮转 writer 使用标准库 `compress/gzip`，历史压缩采用临时文件 + rename，避免留下半成品；启动及跨日首写时执行压缩和过期清理。
- 运行目录无法创建日志文件时服务拒绝启动；单进程部署下 mutex 足以保护写入与跨日轮转，不增加第三方轮转依赖。
- 用户要求文件内容参考 nginx access log；文件行将同时记录 HTTP status 与 body business_code，因为普通 JSON 业务错误的真实 HTTP status 固定为 200。
- 用户明确不需要 request_id；文件行只保留 IP、完成时间、request line、HTTP status、响应字节、耗时和 business_code，且 request line 不含 query。
- 首轮 Windows 测试发现 gzip 源文件必须在 `os.Remove` 前显式关闭，不能只依赖函数退出时 defer。
- 最终文档链接检查必须限定后端文档范围；递归仓库会进入用户的 `frontend/node_modules`，第三方 README 的仓库内相对链接不属于本项目验收范围。

## 2026-08-19 CORS

- 前端 `.env` 直接使用 `http://localhost:8080`，Vite 开发页来源为 `http://localhost:5173`；Axios 固定发送 JSON Content-Type，登录会触发跨域预检，登录后的接口还会发送 Authorization。
- 采用标准库白名单中间件，不新增 go-chi/cors 依赖；精确 Origin 由 `CORS_ALLOWED_ORIGINS` 逗号列表配置，不支持通配符。
- JWT 存在 localStorage 并通过 Authorization 发送，不依赖 Cookie，因此不设置 Access-Control-Allow-Credentials。
- CORS 位于访问日志之后、业务限制与路由之前；OPTIONS 预检会被记录为 HTTP/business 204，同时不会进入业务 handler。

## 2026-08-19 JWT 签发修复

- jwtauth v5 的 HS256 signer 要求 `[]byte` key；组合根此前把环境变量字符串原样传入，构造阶段不报错，直到登录 Encode 才返回 invalid key type。
- service 单元测试一直使用 `[]byte`，因此未覆盖组合根的类型偏差；新增 server 级签发/验证测试锁定真实初始化路径。

## 2026-08-22 文章头图与全文搜索

- 用户纠正需求：保留现有 `GET /posts?tag=...` 精确标签筛选，同时新增独立全文搜索 API。
- 搜索采用 `GET /search?q=...&limit=...&cursor=...`，只匹配 published 文章，搜索 title/summary/content，按 `(published_at,id)` 倒序并复用公开游标。
- PostgreSQL 使用 `pg_trgm` 和 published-only 组合表达式 GIN 索引；相比内置词典 tsvector，对中英文 Markdown 包含搜索更符合当前单站点博客，不引入外部搜索服务。
- cover_img 数据库列为 nullable TEXT；model/响应使用 `*string`。非空值只接受无 userinfo 的绝对 HTTP(S) URL，更新传空字符串清除。
- 本轮开始时工作区已有 RSS URL、Turnstile header、CORS、统一 data、S3 路径等用户/前序未提交改动；实现必须保留并共同验证。
- 仅参数绑定不足以定义正确的 LIKE 搜索语义：`%` 和 `_` 虽不会造成 SQL 注入，但会成为通配符并扩大结果集；repository 需先转义反斜杠、百分号和下划线，再配合固定 `ESCAPE '\\'` 子句。
- 本机 PostgreSQL 可成功启用 pg_trgm 并应用全部 migration；真实 repository 测试确认搜索投影中的 cover_img 扫描顺序与 SQL 一致，避免只靠编译无法发现的行扫描错位。

## 2026-08-22 正文首图自动头图

- 自动提取放在 PostService，避免 handler 或前端分别实现导致创建/更新语义漂移。
- 不引入完整 Markdown 渲染依赖：Markdown 只识别常用内联图片 `![alt](url)`；HTML 图片使用 `golang.org/x/net/html` tokenizer 读取真实 img/src 属性。
- 提取前屏蔽 fenced code、行内 code，HTML tokenizer 本身忽略注释并把 script 内容作为文本，避免把示例代码中的图片误当头图。
- 候选仍复用 cover_img 的绝对 HTTP(S)、无 userinfo、最大长度校验；相对 URL、data/javascript 协议不入库，并继续寻找后续第一个合法候选。
- cover_img 非空时直接设置；保存后的值为空时从最终正文回填，因此显式空字符串也会触发。正文无合法图片时空字符串清除为 NULL；字段缺失时保留已有 cover，只有当前值为空才回填。

## 2026-08-29 Mock 服务

- 用户要求在 `/mock` 下新增可运行的 mock 服务，至少包含 10 篇文章、每篇不同评论数量、数条友链，并覆盖各种现有业务状态。
- 当前真实契约的有限状态集合为：文章 `draft/published`，评论 `visible/hidden/deleted`，用户角色 `admin/contributor`；公开文章与标签聚合必须隔离草稿。
- JSON 业务响应始终为 HTTP 200，响应体固定 `{code,msg,data}`；RSS 是 XML 和真实 HTTP 状态的例外。
- 方案比较：复用生产 router/service 需要替换大量 repository 并处理 JWT/S3/Turnstile 依赖；独立内存服务能保持零外部依赖，并可直接按当前 DTO/路由契约实现，推荐后者。
- `cover_img` 必须保持当前字段名和 nullable/空值语义；种子应同时覆盖有头图、无头图、正文首图、空标签、多标签、已发布和草稿。
- 工作区开始时已有大量未提交业务与文档改动；本任务只追加 `/mock` 及必要文档/规划记录，不回退或认领现有改动。
- Mock 将覆盖当前所有主要路由组：`/service`、`/auth`、`/admin`、`/rss` 和公开根路由；上传只返回假预签名数据，不承诺实际对象存储上传。
- 为支持真实前端操作，后台发布/撤回、评论审核和用户/分类/文章/友链 CRUD 使用带锁内存状态；进程重启恢复固定种子。
- Mock 认证使用显式测试账号与固定 Bearer token 区分 admin/contributor；Turnstile 只在需要的端点检查非空请求头，不调用外部服务。
- 10 篇文章将固定覆盖每篇 0 到 9 条评论（总计 45 条），同时分布根评论、嵌套回复以及 visible/hidden/deleted 状态；草稿评论只通过后台可见。
- 生产权限语义：contributor 只能查看和操作自己作者 ID 下的文章与评论；admin 可管理全部，并独占用户、分类和友链 CRUD。
- 生产状态转换会拒绝冲突操作：只能发布 draft、撤回 published、隐藏 visible、显示 hidden，deleted 不能重复删除。
- RSS 仅包含 published 文章与其 visible 评论，使用 RSS 2.0 XML；这是统一 JSON envelope 的明确例外。
- 已向用户询问是否将 mock 入口同步到根 README 与 `docs/testing.md`；确认前仅创建自包含的 `mock/README.md`，不触碰已有未提交文档。
- 用户环境的 `all_proxy`、`http_proxy`、`https_proxy` 均指向 `127.0.0.1:7890`；PowerShell `Invoke-RestMethod` 对本机 mock 也发生代理连接重置。实际 smoke 应显式使用 `curl.exe --noproxy '*'` 或配置 `NO_PROXY=127.0.0.1,localhost`。
- 最终本机资源改为 mock 动态生成 PNG，cover/avatar/favicon 及上传 public_url 不依赖境外图片服务；默认绝对资源 URL 绑定 `localhost:8090`。
