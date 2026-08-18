# SimpleBlog 测试与验收

## 1. 默认质量门禁

以下命令不依赖外部 PostgreSQL 或 S3；相应集成测试会自动跳过：

```powershell
go mod verify
gofmt -d cmd internal migrations
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

测试覆盖 service 业务规则、handler 响应、路由权限、中间件、DTO 脱敏、S3 policy 和 migration 解析。

## 2. PostgreSQL 集成测试

推荐使用专用测试数据库或 PostgreSQL 的维护数据库：

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres?sslmode=disable"
go test ./internal/repository ./migrations -count=1 -v
Remove-Item Env:TEST_DATABASE_URL
```

测试在随机 schema 中应用全部 migration，并在结束时删除或回滚该 schema，不会创建或删除数据库。不要把 `TEST_DATABASE_URL` 指向权限和隔离策略不明确的生产实例。

本地开发也可以启用 `.env` 中的数据库配置：

```powershell
$env:SIMPLEBLOG_TEST_LOCAL_DB = "1"
go test ./internal/repository ./migrations -count=1
Remove-Item Env:SIMPLEBLOG_TEST_LOCAL_DB
```

## 3. PostgreSQL 执行计划

代表性数据测试会创建 10k users、1k categories、20k posts 和 50k comments，执行 `EXPLAIN (ANALYZE, BUFFERS)` 并断言关键索引：

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres?sslmode=disable"
go test ./internal/repository -run TestRepresentativePostgreSQLQueryPlans -count=1 -v
Remove-Item Env:TEST_DATABASE_URL
```

该结果用于检查计划形状和索引匹配，不是生产容量或延迟 SLA。详细证据见 [PostgreSQL 优化记录](postgresql-optimization.md)。

## 4. S3 真实上传测试

该测试会创建临时 bucket、上传一个对象、验证公开读取，然后删除对象和 bucket。只能对允许创建/删除测试 bucket 的隔离 S3 兼容实例运行：

```powershell
$env:SIMPLEBLOG_TEST_S3_LOCAL = "1"
$env:SIMPLEBLOG_TEST_S3_ENDPOINT = "127.0.0.1:9000"
$env:SIMPLEBLOG_TEST_S3_REGION = "us-east-1"
$env:SIMPLEBLOG_TEST_S3_ACCESS_KEY = "minioadmin"
$env:SIMPLEBLOG_TEST_S3_SECRET_KEY = "minioadmin"
go test ./internal/storage -run TestS3SignerRealUploadAndPublicRead -count=1 -v
```

完成后移除上述测试环境变量。

## 5. 重点验收项

- 所有应用业务结果为 HTTP 200，真实结果位于 body `code`。
- 公开文章不出现 draft，列表不读取或返回正文。
- contributor 不能管理他人文章或评论。
- 最后一个 admin 不能被删除或降级，并发操作仍保持至少一个 admin。
- 公开评论响应和日志不出现邮箱；头像 URL 可由配置镜像生成。
- hidden/deleted 父评论保留回复树，deleted 固定显示“该评论已被删除”。
- 图片 POST policy 限制 MIME、精确声明大小、对象 key 和 10 分钟有效期。
- 慢查询与错误日志不包含 SQL 参数、密码、JWT、邮箱或 S3 签名。
- 评论昵称/正文拒绝 HTML 尖括号和危险控制字符，旧数据的公开 JSON/RSS 输出不含可执行 HTML 分隔符。
- RSS 返回可解析的 RSS 2.0 XML，文章只含 published、评论只含 visible + published post，失败使用真实 HTTP 500。
- SQL 注入载荷以参数写入友情链接后原样读回，表结构保持存在；RSS 查询命中部分索引。
