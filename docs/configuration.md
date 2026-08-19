# SimpleBlog 配置说明

## 1. 加载规则

服务启动时先尝试读取进程当前工作目录中的 `.env`，再解析环境变量。操作系统中已经存在的环境变量优先，不会被 `.env` 覆盖。

可以从根目录的 [`.env.example`](../.env.example) 开始配置。`.env` 已被 Git 忽略，不要提交真实密码、JWT、Turnstile 或 S3 密钥。

## 2. 配置项

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LOG_LEVEL` | `INFO` | `DEBUG`、`INFO`、`WARN` 或 `ERROR` |
| `DEBUG` | `false` | 为 `true` 时允许开发默认密钥；不改变路由或日志结构 |
| `ACCESS_LOG_RETENTION_DAYS` | `30` | 访问日志保留的自然日总数，包含当天；范围 1–3650 |
| `SITE_NAME` | `SimpleBlog` | RSS 使用的站点名称，不能为空 |
| `SITE_DESCRIPTION` | `A SimpleBlog site` | RSS channel 描述，不能为空 |
| `SITE_BASE_URL` | `http://localhost:3000` | 公开前端绝对基址，用于 RSS 链接，不允许 query/fragment |
| `SERVER_HOST` | `localhost` | HTTP 监听地址；容器或远程部署通常使用 `0.0.0.0` |
| `SERVER_PORT` | `8080` | 1–65535 |
| `TRUST_PROXY_HEADERS` | `false` | 是否信任反向代理写入的 `X-Real-IP` |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | 允许浏览器调用 API 的精确 Origin，多个值以逗号分隔 |
| `DB_HOST` | `localhost` | PostgreSQL 主机 |
| `DB_PORT` | `5432` | PostgreSQL 端口 |
| `DB_USER` | `postgres` | 数据库用户 |
| `DB_PASSWORD` | `password` | 数据库密码；生产环境拒绝此默认值 |
| `DB_NAME` | `mydb` | 数据库名 |
| `DB_SSLMODE` | `disable` | pgx 支持的 `disable/allow/prefer/require/verify-ca/verify-full` |
| `DB_SLOW_QUERY_MS` | `250` | 慢查询阈值，范围 1–60000 ms |
| `JWT_SECRET` | `secret` | HS256 密钥；生产环境至少 32 字符且不能使用默认值 |
| `TURNSTILE_SITEKEY` | `secret` | 供前端配置的站点密钥；当前后端不返回此值 |
| `TURNSTILE_SECRET` | `secret` | 服务端 siteverify 密钥；生产环境必须替换 |
| `S3_ENDPOINT` | `localhost:9000` | S3 兼容 endpoint，不带 URL scheme |
| `S3_REGION` | `us-east-1` | S3 region |
| `S3_BUCKET` | `simpleblog` | 已存在的正文图片 bucket |
| `S3_ACCESS_KEY` | `minioadmin` | S3 access key；生产环境拒绝默认值 |
| `S3_SECRET_KEY` | `minioadmin` | S3 secret key；生产环境拒绝默认值 |
| `S3_USE_SSL` | `false` | S3 endpoint 是否使用 HTTPS |
| `S3_PUBLIC_BASE_URL` | `http://localhost:9000/simpleblog` | 图片公开读取或 CDN 基址，不允许 query/fragment |
| `GRAVATAR_BASE_URL` | `https://www.gravatar.com/avatar` | Gravatar 或兼容镜像基址，不允许 query/fragment |

## 3. 生产环境要求

当 `DEBUG=false` 时，配置校验会拒绝默认数据库密码、短 JWT 密钥、默认 Turnstile secret 和默认 MinIO 凭据。服务还会在监听 HTTP 前 Ping PostgreSQL；配置或数据库不可用时进程直接退出。

`TRUST_PROXY_HEADERS=true` 只适用于可信反向代理。代理必须无条件覆盖客户端发送的 `X-Real-IP`；否则攻击者可以伪造 IP 绕过登录限流或污染 Turnstile 的 remote IP。

`CORS_ALLOWED_ORIGINS` 必须填写完整的 `http://` 或 `https://` Origin，不带路径、query、fragment 或用户信息，也不接受 `*`。生产部署应替换为真实前端 Origin，例如 `https://blog.example.com`；配置只影响浏览器跨域读取，不替代认证和权限检查。

慢查询日志只记录 request ID、SQL 指纹、耗时、命令标签和 SQLSTATE，不记录 SQL 文本、参数或 PostgreSQL Detail。

普通内部错误通过 tint 输出完整 Go error 链，便于直接诊断数据库、认证和外部服务故障；PostgreSQL error 可能包含 Detail 和输入值。生产环境必须限制日志读取权限并配置合适的保存周期。

访问日志固定写入进程当前工作目录的 `logs` 子目录。当天使用 `access-YYYY-MM-DD.log`，过去日期使用同名 `.log.gz`；`ACCESS_LOG_RETENTION_DAYS=1` 表示仅保留当天。目录创建或当前日志打开失败时，服务拒绝启动。

## 4. 站点 URL、S3 与 Gravatar

`SITE_BASE_URL` 指向读者访问文章的前端站点，不是 API 地址。RSS 会生成 `{SITE_BASE_URL}/posts/{slug}`；部署前必须改成生产 HTTPS 域名。

S3 bucket 必须预先创建，并允许浏览器站点源向签名地址发送 POST。若图片需要直接展示，还要为对象配置公开读取或通过 CDN 提供读取能力。`S3_PUBLIC_BASE_URL` 只用于生成最终图片地址，不负责修改 bucket 权限。

公开评论查询会读取邮箱，对 `trim + lowercase` 后的值计算 Gravatar MD5，并只向前端返回 `avatar_url`。hidden/deleted 评论返回空头像；邮箱不进入公开 DTO，但数据库错误详情可能进入受控服务端日志。
