# SimpleBlog 分析进度

## 2026-08-18

- 已读取 `planning-with-files-zh` 技能说明。
- 已确认仓库内不存在旧的 `task_plan.md`、`findings.md`、`progress.md`。
- 已查询长期记忆索引，未发现 SimpleBlog 的既有项目决策。
- 已创建本次只读架构分析所需的规划文件。
- 已完成第一轮文件清单和全部 Go 源码阅读；确认当前工作树存在用户自己的未提交认证改动。
- 已梳理启动流程、登录链路、Turnstile 链路和第一批结构性风险。
- 已核对 staged/unstaged diff、提交历史、Git 跟踪范围与根目录文件；未读取本地 `.env`。
- 阶段 1 完成，进入请求/数据链路与可构建性核验。
- 已运行 `gofmt -d`、`go test ./...`、`go vet ./...`，均通过；当前没有任何测试文件。
- 已查询当前 jwtauth、pgxpool、Chi 中间件接口，确认鉴权链、数据库 Ping 和代理 IP 的准确行为。
- 阶段 2、3 完成，开始形成最小博客目标设计和方案比较。
- 用户要求暂不继续查询 API 文档；后续设计已收束到整体结构。
- 已完成全局技术分层与领域模块化方案比较，推荐轻量领域模块化单体。
- 已完成整体设计、差距与分阶段落地顺序，未修改任何业务代码。
- 用户补充了多用户、文章、评论、分类、标签与 S3 直传的明确边界，并确认保留技术分层目录。
- 已核对 `response` 已迁入 `internal/response`，开始重新设计数据表与实施路线。
- 已完成 users/categories/posts/comments 四表设计；tags 使用 posts.text[]，S3 暂不建表。
- 已完成 admin/contributor/访客权限矩阵、现有技术分层内的职责分配和七阶段实施路线。
- 用户确认多级评论、无存量表、HTTP 200 + body code 合约，并要求设计落入 docs。
- 已确认仓库尚无 docs 目录，准备创建四份相互一致的设计文档。
- 已创建 architecture/database/permissions/api 四份文档；评论删除最终确认为墓碑节点，固定显示“该评论已被删除”并保留子回复。
- 第一次相对链接检查脚本参数错误，已记录；内容一致性扫描通过，等待修正后重验链接。
- 已用修正后的脚本验证四份文档的相对链接全部可解析，并确认 UTF-8 中文标题可正常读取。
- 阶段 9 完成；本轮只新增/修改设计与规划文档，没有修改业务代码或数据库。
- 用户批准 milestone 计划并要求每个 milestone 单独签名 commit 后通知验收。
- 已确认工作区干净且 Git 签名已配置，创建 `codex/simpleblog-milestones` 分支。
- M0 开始：先固化 milestone 文档与验收边界。
- 已新增 `docs/milestones.md` 并同步无限评论深度与 S3 文件限制；开始实现初始 schema。
- 已核对 response/config/router 的调用范围，确定 M0 可在不改变业务 handler 合约的前提下集中实现基础能力。
- 已创建 `migrations/000001_init.sql` 与执行说明，schema 与数据库设计文档一致。
- 已实现配置集中校验、数据库 URL 安全构造、启动 Ping、HTTP 超时和信号驱动优雅关闭。
- 已新增根路由装配、RequestID、自定义 panic 恢复以及统一的 404/405 HTTP 200 响应，并补充单元测试。
- 首轮 `gofmt -d`、`go test ./...`、`go vet ./...` 均通过。
- SQL 集成测试发现 `.env` 指向的 SimpleBlog 数据库尚不存在；记录后改用 postgres 维护库进行非持久化验证。
- SQL 在随机 schema 中执行成功，并验证删除文章可级联清理父子评论；事务已回滚。
- 最终通过 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`gofmt -d` 和文档链接检查。
- 使用不存在的数据库启动服务时按预期在 Ping 阶段返回错误并以非零状态退出。
- M0 实现完成并创建签名 commit，等待用户验收后进入 M1。
- 用户验收 M0，同意进入 M1。
- M1 开始：User 语义重构、用户 CRUD、JWT/当前用户、角色守卫和首个管理员 CLI。
- 已核对现有认证栈与本地 jwtauth/pgx 接口，确定 UserRepository 复用、JWT claims、当前用户加载和事务保护方案。
- 用户确认后端路由不使用 `/api` 前缀；认证与后台路由调整为 `/auth`、`/admin`，并同步 API 文档。
- 已完成 User 模型/仓储重构、用户 CRUD、bcrypt、2 小时 JWT、数据库当前用户加载、admin 角色守卫与首个管理员 CLI。
- 已实现事务级 advisory lock 保护最后一个 admin，并禁止当前 admin 删除自己。
- 新增认证、用户服务、中间件单元测试及 UserRepository PostgreSQL 集成测试；默认测试与本地 postgres 临时 schema 验证均通过。
- M1 最终通过 `gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...` 与文档链接检查，准备创建独立签名 commit。
- 用户验收 M1，同意进入 M2，并要求 JWT 有效期从 2 小时调整为 24 小时。
- M2 开始：分类 CRUD、文章后台 CRUD、Markdown + LaTeX 原文、TEXT[] 标签、作者权限、发布/撤回状态机。
- M2 首次真实 PostgreSQL 测试确认 `ON DELETE RESTRICT` 返回 SQLSTATE 23001；已扩展引用冲突映射，避免误报 500。
- 已完成 Category 与 Post 的 model/repository/service/dto/handler，接入 `/admin/categories` 与 `/admin/posts` 路由和 server 组合根。
- 已完成分类 admin 权限、文章作者所有权、slug 校验、标签规范化、draft/published 发布撤回状态机；普通 PATCH 无法提交 author_id/status。
- 已新增 M2 service、router、handler 单元测试和 PostgreSQL 内容生命周期集成测试；Markdown + LaTeX 原文与 published_at 保留规则验证通过。
- M2 最终通过 `gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、真实 PostgreSQL 临时 schema、`go build ./...` 与文档链接检查，准备创建独立签名 commit。
- 用户验收 M2，同意继续完成 M3。
- M3 开始：公开已发布文章列表与详情、分类和标签列表、分类/标签筛选、稳定游标分页。
- 已实现 published-only 公开查询、作者/分类安全联表字段、分类与标签筛选，以及 `(published_at, id)` Base64URL 游标分页。
- 已新增公开 DTO、PublicService/PublicHandler/PublicRouter 并挂载 `/posts`、`/categories`、`/tags`，无需登录。
- 单元、路由与真实 PostgreSQL 测试已验证草稿不会进入列表、详情或标签聚合，列表不返回正文，详情保留 Markdown + LaTeX 原文。
- M3 最终测试门禁通过；首次 DTO 静态扫描命令因 PowerShell 正则引号错误失败，已记录并改用单引号重跑。
- M3 最终通过 `gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、真实 PostgreSQL 临时 schema、`go build ./...`、公开字段静态扫描与文档链接检查，准备创建独立签名 commit。
- 用户要求继续 M4 与 M5，并在开始前扩充 M6：加入 PostgreSQL 查询深度优化、安全与性能验证；已同步到 milestone 范围和可验收标准。
- 已完成 M4 Comment model/repository/service/dto/handler，接入公开评论树、Turnstile 创建与后台审核路由。
- 评论创建原子限制为已发布文章并由复合外键阻止跨文章回复；公开 SQL 不读取 email，后台按 admin/contributor 所有权过滤。
- 隐藏可恢复，删除写入不可恢复墓碑并清空个人信息；真实 PostgreSQL 已验证任意层级回复、回复墓碑和子回复保留。
- M4 最终通过 `gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、真实 PostgreSQL 临时 schema、`go build ./...`、公开邮箱字段扫描与文档链接检查，准备创建独立签名 commit。
- M4 已创建签名 commit `00eb7fcad6ced25da1b0c4fa2252c53eb8e60f44`，GPG 验证为 Good signature；按用户对 M4/M5 的连续授权进入 M5。
- M5 采用 S3 POST policy，完成 S3 配置、MinIO 签名适配、UploadService/DTO/handler 与 `/admin/uploads/presign` 路由。
- 上传策略限制 JPEG/PNG/WebP/GIF、1 到 10 MiB、精确声明字节数、服务端 key 和 10 分钟有效期；图片数据不经过 Go 服务。
- 单元/路由测试覆盖角色、MIME、大小、客户端文件名隔离和签名 policy；临时真实 MinIO 已验证 multipart POST 上传与匿名公开读取，随后停止并清理。
- M5 最终通过 `gofmt -d`、`go mod verify`、`go vet ./...`、`go test ./...`、`go test -race ./...`、真实 PostgreSQL 回归、`go build ./...`、静态敏感信息扫描与文档链接检查，准备创建独立签名 commit。
- 用户验收 M5 并批准进入 M6；补充公开评论需返回按邮箱生成的 Gravatar 头像 URL，且镜像源可配置。
- 用户确认公开查询可以读取 email，只要前端响应不返回 email；因此采用 DTO 即时哈希方案，不修改 comments schema。
- 已完成 M6 第一轮代码/SQL/索引审计：发现公开文章列表读取正文、用户安全读取仍取 password_hash、slug 索引表达式不匹配、评论排序索引不匹配、日志原始错误泄露风险及缺少全局 body 限制。
- 已实现可配置 Gravatar：公开查询读取 email，DTO 仅返回 avatar_url，hidden/deleted 不返回头像；创建与树响应均覆盖镜像源和邮箱规范化测试。
- 已增加 2 MiB 全局 body 限制、413 业务响应、字段级长度限制、结构化访问日志和数据库错误脱敏日志。
- 已增加 PostgreSQL 错误分类与 503 映射；公开文章列表移除 content 读取，非登录用户查询移除 password_hash 读取。
- 已为后台文章/评论增加默认 50、最大 100 的 keyset 分页；后台文章列表 DTO/SQL 不再包含正文。
- 已加入登录 IP 限流和显式可信代理配置，避免直接部署时伪造 X-Real-IP 绕过 Turnstile/限流。
- 已新增 000002 索引优化迁移并让所有数据库测试按文件名顺序执行迁移；并发测试验证两个 admin 同时删除时恰有一个被最后管理员保护拒绝。
- 代表性数据 EXPLAIN 测试已通过：10k users、1k categories、20k posts、50k comments，8 条关键查询均命中预期索引且高增长表无可避免的顺序扫描。
- 已接入可配置慢查询 tracer 并通过脱敏测试：日志不包含 SQL 文本、查询参数、邮箱或 PostgreSQL Detail。
- 已同步 API、架构、权限、数据库和 milestone 文档，并新增 PostgreSQL 优化记录与复现方式。
- 已补测 contributor 评论执行计划并修正全站索引扫描热点；九条代表性查询均通过索引与顺序扫描断言。
- 已让列表、登录和认证中间件统一把可重试 PostgreSQL 错误映射为业务 code 503。
- M6 最终通过 `go mod verify`、`gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`、真实 PostgreSQL 全量集成测试、文档链接与敏感信息一致性扫描，准备创建签名 commit。
- 用户确认 M7 可标记完成，并提出 M8：文章/评论 RSS、友情链接、SQL 注入与 XSS 防护。
- 已恢复规划文件并确认上一轮文档已被暂存；保留现有 staged 状态，不提交。
- 已把 M7 标记完成并登记 M8 草案；RSS、友情链接与 XSS 策略等待产品确认。
- 用户确认文章 HTML 可执行是预期行为，评论才是 XSS 防护范围；后续安全设计不得清洗文章 Markdown/HTML。
- 已审计现有 repository SQL 构造：外部值均参数化，动态部分仅为固定子句和占位符编号；尚未发现 SQL 注入入口。
- 已确认评论内容当前原样存储/返回，JSON 传输转义不能替代前端 DOM 安全；M8 需在评论写入与历史数据输出两侧建立纯文本防线。
- 用户批准 M8 推荐方案；开始实现 RSS 2.0、友情链接公开列表/admin CRUD、评论纯文本防线和 SQL 注入回归测试。
- 已新增 `SITE_NAME`、`SITE_DESCRIPTION`、`SITE_BASE_URL`，实现 `/rss/article` 与 `/rss/comment` 原始 RSS 2.0 XML；RSS 失败使用真实 HTTP 500并由访问日志记录真实状态。
- 已实现 `friend_links` migration 及公开 `GET /links`、admin-only `/admin/links` CRUD；URL 只允许 HTTP(S)，服务端不抓取外部资源。
- 已把评论昵称/正文收束为规范化纯文本：新输入拒绝 HTML 尖括号和危险控制字符，DTO/RSS 对历史旧数据执行输出中和，文章可信 HTML 保持原样。
- 已加入 `X-Content-Type-Options: nosniff`、评论数据库纵深约束、危险旧链接 DTO 降级、RSS XML 编码与可见性测试。
- 默认质量门禁全部通过：`go mod verify`、`gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`。
- 真实 PostgreSQL 全量测试通过：迁移、RSS published/visible 过滤、评论 CHECK、SQL 注入载荷、查询计划与既有 repository 回归均成功；RSS 评论查询命中部分索引。
- 已同步 README 与 docs 中的 API、架构、数据库、权限、配置、部署、测试、限制、PostgreSQL 优化、安全边界及 milestone；文档链接和过期术语扫描通过。
- M8 已完成，按用户本轮要求保留为未提交工作区变更。

## 2026-08-19

- 按用户要求恢复完整 tint/slog 错误输出，移除仅记录 error_type/SQLSTATE 的包装器；启动实测已能显示完整 `ping database` 错误链。
- 开始实现访问日志按日留存：独立访问日志文件、历史 gzip、可配置自然日保留策略。
- 已新增标准库实现的 `internal/logging.AccessWriter`：当前日文本、历史 gzip、临时文件压缩、自然日过期清理和互斥轮转；加入配置与核心轮转测试。
- 已把访问日志 writer 显式注入根路由；每条访问记录继续输出到终端 tint，同时以 nginx 风格写入独立按日文件，完整运行错误日志路径保持不变。
- 文件行同时记录 HTTP status 与业务 code，不记录用户明确移除的 request_id，也不记录 query、请求体、认证头或评论邮箱；定向测试已通过。
- 访问日志实现已同步 README、配置、架构、运维、测试与安全文档；`ACCESS_LOG_RETENTION_DAYS` 默认 30，范围 1–3650，包含当天。
- 阶段 19 最终通过 `go mod verify`、`gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...` 与 `git diff --check`，按用户要求暂不提交。
- 首次文档链接脚本误递归扫描 `frontend/node_modules`；已记录并改为只检查 `README.md` 与 `docs/*.md`，不触碰用户前端目录。
- 收紧范围后的后端文档链接检查全部通过；工作区未暂存、未提交，`frontend` 及用户维护的额外文档未被修改。

## 2026-08-19 CORS

- 已确认前端 Vite 5173 → API 8080 的直接请求，以及 Content-Type/Authorization 触发预检的原因。
- 已新增标准库 CORS 白名单中间件、`CORS_ALLOWED_ORIGINS` 配置校验并接入根路由；登录预检、普通响应和拒绝未知来源的定向测试通过。
- 已同步配置、API、架构、部署、测试和安全文档；未修改 frontend。
- CORS 阶段最终通过 `go mod verify`、`gofmt -d`、`go vet ./...`、`go test ./...`、`go test -race ./...`、`go build ./...`、后端文档链接检查和 `git diff --check`；暂未提交。

## 2026-08-19 JWT 签发修复

- 根据实际登录日志定位到组合根把 JWT_SECRET string 传给 HS256 signer；已改为 []byte 并增加真实 Encode/Verify 回归测试。
- 修复最终通过 `gofmt`、`go vet ./...`、`go test ./...`、`go test -race ./...` 与 `go build ./...`；暂未提交。

## 2026-08-22 文章头图与全文搜索

- 已按 planning-with-files-zh 恢复规划文件并检查中断状态；中断前没有留下 cover_img/search 半成品。
- 用户最终要求保留标签筛选；确定新增独立 `/search`，搜索 published 文章的标题、摘要和正文，使用稳定公开游标。
- 确定 cover_img 为 nullable TEXT/*string，空字符串用于后台 PATCH 清除，非空校验绝对 HTTP(S) URL。
- 已新增 `000004_post_cover_search.sql`：启用 `pg_trgm`、增加 nullable `posts.cover_img`、建立 published-only trigram 表达式 GIN 索引。
- 已贯通 model/dto/handler/service/repository 的 cover_img 创建、查询、列表、更新和清除语义；公开与后台响应统一返回 nullable 字段。
- 已新增公开 `GET /search?q=&limit=&cursor=`，只检索 published 文章的 title/summary/content，保留 `/posts?tag=` 精确筛选并复用公开稳定游标。
- 搜索 SQL 使用 pgx 参数绑定，并把 `%`、`_`、反斜杠按 LIKE 字面量转义，避免查询词扩大匹配范围。
- 已同步 API、HTTP API、OpenAPI、架构、数据库、权限、测试、PostgreSQL 优化和 migration 文档。
- 本地 PostgreSQL 随机 schema 的 migration 与 repository 集成测试通过：cover_img 写入/NULL 清除、草稿隔离、全文字段匹配、LIKE 通配符防护及标签筛选均已验证。
- 最终通过带真实 PostgreSQL 的 `go test ./... -count=1`、`go vet ./...`、Swagger JSON 解析与 `git diff --check`；按用户要求未提交。

## 2026-08-22 正文首图自动头图

- 已比较完整 Markdown AST 依赖与窄范围提取器；选择标准 HTML tokenizer + 常用 Markdown 内联图片提取，不增加完整渲染依赖。
- 已确定创建、更新、显式清除、已有头图和非法图片 URL 的行为边界，开始实现 service 与测试。
- 已新增 `post_image.go`：按正文顺序提取 Markdown 内联图片与 HTML img/src，屏蔽 fenced/inline code，并复用 cover URL 安全校验。
- 已接入 PostService 创建和更新流程；保存后空值自动回填、正文无图时清除、未传字段保留已有头图的定向测试已覆盖。
- `golang.org/x/net/html` 原本已在依赖树中，`go mod tidy` 仅将其提升为直接依赖；没有引入完整 Markdown 渲染框架。
- 已同步 API、HTTP API、架构、OpenAPI 与测试文档，等待全仓质量门禁。
- 最终语义复核纠正了“显式空字符串不回填”的先前假设：按用户原文改为任何保存后的空 cover 均尝试提取，正文无图时才清除为 NULL，并补充对应测试与文档。
- 首次语义调整测试因跨函数引用 `empty` 局部变量而编译失败；已定位为测试夹具作用域问题并在当前测试中定义该值。
- 首次修复补丁因 hunk 分隔格式无效而未应用；读取精确上下文后拆分重试，未造成部分文件修改。
- 阶段 22 最终通过带真实 PostgreSQL 的 `go test ./... -count=1`、`go test -race ./internal/service`、`go build ./...`、`go vet ./...`、`go mod verify`、Swagger JSON 解析与 `git diff --check`；未提交。

## 2026-08-29 Mock 服务

- 已读取 planning-with-files-zh 规则并恢复既有规划文件；将本次任务追加为阶段 23，没有覆盖既有计划。
- 已检查 Git 状态，确认仓库存在用户的未提交 API、模型、服务、测试与文档改动；后续保持这些改动原样。
- 已核对当前 router、DTO、model、response 及 `docs/httpapi.md` 路由清单，确认状态集合和统一 HTTP 200 JSON envelope。
- 已比较生产服务替换存储层与独立内存服务两条路径，选择 `/mock` 独立零依赖 Go 服务，下一步确定端点覆盖和可变更行为。
- 已确认实现覆盖清单：全部主要公开/认证/后台/RSS/service 路由；后台写操作在内存中生效，重启重置。
- 已确定种子分布：10 篇文章分别 0..9 条评论，共 45 条，覆盖全部文章/评论/角色状态以及 cover/tags/嵌套评论差异。
- 已核对生产权限与状态转换语义，mock 将保留 contributor 作者隔离、admin 专属资源和冲突错误行为。
- 已询问用户是否同步根 README 与 `docs/testing.md`；等待答复期间只编写 `/mock/README.md`，不修改现有文档。
- 已创建 `/mock` 程序骨架、带锁内存 store 和固定种子；文章按稳定 UUID 映射 0..9 条评论，避免展示排序影响数量分布。
- 已接入公开、认证和后台资源路由：统一 envelope、CORS、Turnstile mock、静态/动态 token、分页、筛选、搜索、评论树及用户/分类/友链 CRUD。
- 已接入文章/评论后台操作：作者隔离、创建更新删除、发布撤回、评论隐藏显示软删除及冲突状态检查。
- 已接入 RSS 与假 S3 POST 流程；预签名返回可提交的本机 `/mock-upload`，公开 URL 使用稳定可显示的测试图源。
- 首次 gofmt 发现 `deleteLink` 的 for 循环缺少闭合花括号；已读取报错上下文并定点修正，未重复失败命令。
- 修正后 `/mock` 全部 Go 文件已通过 gofmt，`go test ./mock` 编译通过；当前尚未加入行为测试。
- 已新增 5 组 httptest 行为测试并通过：种子状态/数量、公开隔离与评论脱敏、认证权限、发布转换、分页搜索 RSS 与预签名。
- 为消除外部图片依赖，开始把 cover/avatar/favicon 与上传 public_url 改为本机动态 PNG 资源端点。
- 本机动态 PNG 已完成并通过定向测试；所有种子展示资源不再依赖外网。
- 首次真实 HTTP 脚本中 `localhost` 的公开列表连接被重置，但日志无 panic、服务保持监听，随后 `127.0.0.1` 单独请求稳定返回 HTTP 200 和 7 篇公开文章；继续执行严格 smoke 排除瞬态。
- 第二次新 PowerShell 进程的首个本机请求仍被重置；检查确认 all_proxy/http_proxy/https_proxy 指向本机 7890，属于验证客户端代理路径。
- 改用 `curl.exe --noproxy '*'` 后严格 smoke 全部通过：ping code 200、公开文章 7、后台文章 10、友链 5、admin token 正确、RSS 为 200 XML、PNG 为 200 且 8511 bytes。
- 最终态重新通过 `go test ./... -count=1`、`go test -race ./mock -count=1`、`go vet ./...`、`go build ./...`、mock gofmt 差异检查与 `git diff --check`。
- 阶段 23 完成；mock 服务继续在 `http://localhost:8090` 后台运行，进程重启即可恢复固定种子。
- planning-with-files 完成脚本最初无法识别既有表格计划（0/0）；已为阶段 23 追加兼容的 `### 阶段`/`**状态：** complete` 状态块，不改写历史阶段。
