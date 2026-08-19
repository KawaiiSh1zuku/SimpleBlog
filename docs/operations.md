# SimpleBlog 运行与部署

## 1. 依赖

- Go 1.25。
- PostgreSQL 12+；部署账号需要对目标数据库执行 migration 的权限。当前执行计划已在 PostgreSQL 18 上验证。
- `psql` 命令行工具。
- 已创建的 S3 兼容 bucket。
- Cloudflare Turnstile site key 与 secret。

当前仓库没有 Dockerfile、容器编排或自动迁移器。迁移由部署流程显式执行，API 进程不会自动修改 schema。

## 2. 初始化数据库

在空数据库上按文件名顺序执行全部 SQL：

```cmd
set DATABASE_URL="postgres://user:password@localhost:5432/simpleblog?sslmode=disable"
for /f "delims=" %i in ('dir /b /on migrations\*.sql') do "E:\Program Files\PostgreSQL\18\bin\psql.exe" %DATABASE_URL% -v ON_ERROR_STOP=1 -f "migrations\%i"
```

每个 migration 自带事务。升级时先备份数据库，再执行尚未应用的新文件；仓库目前不提供 down migration，不应通过反向执行 SQL 文件回滚。

当前没有 migration history 表或自动 runner。首次初始化可以执行全部文件；已有环境必须由部署系统记录已应用版本，并只执行新增文件。

应用本身使用分离的 `DB_*` 配置，`DATABASE_URL` 仅是上述 `psql` 示例变量。

## 3. 创建首个管理员

服务不暴露匿名初始化接口。完成 migration 和 `.env` 后，在服务器本地执行：

```powershell
$env:ADMIN_PASSWORD = "replace-with-a-strong-password"
go run ./cmd/admin -username admin -nickname Administrator
Remove-Item Env:ADMIN_PASSWORD
```

可选参数为 `-avatar-url`。密码长度必须为 10–72 字节；当数据库中已经存在有效 admin 时，命令会拒绝再次初始化。

## 4. 启动与停止

```powershell
go run ./cmd/server
```

也可以先构建再运行：

```powershell
go build -o simpleblog.exe ./cmd/server
.\simpleblog.exe
```

进程在启动阶段校验配置、创建数据库连接池并 Ping PostgreSQL。收到 `Ctrl+C` 或 `SIGTERM` 后最多等待 10 秒完成 HTTP 优雅关闭。

HTTP 超时为：header 5 秒、读取 15 秒、写入 30 秒、空闲连接 60 秒。单个 API 请求体最大 2 MiB。

## 5. 健康与诊断端点

```text
GET /service/ping
GET /service/test_turnstile
```

`/service/ping` 是轻量存活端点，不执行数据库查询，因此不能替代数据库就绪检查。`/service/test_turnstile` 需要 `X-Turnstile-Token`，仅用于验证当前 Turnstile 配置。

## 6. 反向代理

生产环境建议由反向代理终止 TLS，再转发到 SimpleBlog。默认保持 `TRUST_PROXY_HEADERS=false`；只有在代理会删除客户端原始 `X-Real-IP` 并写入可信值时才开启。

前端与 API 不同源时，将前端的精确 Origin 写入 `CORS_ALLOWED_ORIGINS`。例如：

```ini
CORS_ALLOWED_ORIGINS=https://blog.example.com,https://admin.example.com
```

服务端允许 GET、HEAD、POST、PATCH、DELETE、OPTIONS，并允许浏览器预检 `Authorization`、`Content-Type` 与 `X-Turnstile-Token`。JWT 从 `Authorization` 读取，因此 CORS 不启用 Cookie credentials。开发默认允许 Vite 的 `localhost:5173` 和 `127.0.0.1:5173`。

JSON 业务响应始终写成 HTTP 200，因此代理监控应解析 body `code` 或使用结构化访问日志中的 `business_code`。RSS 端点是例外，失败会返回真实 HTTP 5xx，可按标准状态监控。

## 7. 访问日志留存

访问日志独立写入启动进程当前工作目录下的 `logs`：

```text
logs/
  access-2026-08-18.log.gz
  access-2026-08-19.log
```

当天文件保持纯文本；启动时和跨日后的首次请求会将旧 `.log` 压缩为 gzip，并删除超过 `ACCESS_LOG_RETENTION_DAYS` 的归档。保留天数包含当天，默认 30 天。运行账号必须能创建和写入该目录；同一运行目录只应由一个 SimpleBlog 进程使用。

每个请求写一行 nginx 风格记录：

```text
203.0.113.9 - - [19/Aug/2026:11:30:00 +0800] "POST /comments HTTP/1.1" 200 96 duration_ms=4 business_code=400
```

字段依次为客户端 IP、完成时间、请求方法/路径/协议、HTTP 状态、响应字节数、耗时和响应体业务 `code`。不记录 request ID、query、请求体、Authorization 或评论邮箱。终端运行日志仍由 tint 输出，内部错误保留完整 error 链，不会混入访问日志归档。

## 8. S3 直传检查

部署前确认：

- bucket 已存在，region 与配置一致。
- S3 CORS 允许前端站点源执行 POST，并允许签名表单需要的请求头。
- `S3_PUBLIC_BASE_URL` 对浏览器可访问，对象或 CDN 具备公开读取能力。
- 前端原样提交预签名响应中的全部 `fields`，最后以 `file` 字段附加文件。
- API 只签发 POST policy，不接收图片字节。

## 9. 上线检查

- `DEBUG=false`，所有默认密钥已替换。
- PostgreSQL 生产连接使用合适的 SSL mode。
- migration 已按顺序执行并完成备份。
- 首个 admin 已创建，`ADMIN_PASSWORD` 已从进程环境移除。
- Turnstile、S3 CORS、公开图片 URL 和 Gravatar 镜像已验证。
- `SITE_BASE_URL` 已指向生产前端 HTTPS 域名，RSS 中的文章/评论链接可访问。
- 反向代理覆盖 `X-Real-IP` 后才开启可信代理配置。
- `CORS_ALLOWED_ORIGINS` 已替换为实际前端 HTTPS Origin，未使用通配符。
- 运行目录可写，`ACCESS_LOG_RETENTION_DAYS` 符合审计要求，且 `logs` 只对运维账号开放。
- `/service/ping`、登录、文章读取、评论提交和 S3 直传链路均完成烟雾测试。
- `/rss/article`、`/rss/comment` 可被 RSS 阅读器解析，`/links` 只返回预期 HTTP(S) 地址。
