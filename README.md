# SimpleBlog

SimpleBlog 是一个小而精的单站点、多用户博客后端，使用 Go、Chi 和 PostgreSQL。项目不包含前端、不使用 ORM，也不引入 Redis、消息队列或媒体代理服务。

当前已完成 M0–M6：

- `admin` / `contributor` 用户、JWT 登录和权限控制。
- 分类、Markdown + LaTeX 文章、`TEXT[]` 标签及草稿/发布/撤回流程。
- 公开文章筛选和稳定游标分页。
- 匿名任意层级评论、Turnstile、隐藏/恢复和删除墓碑。
- 根据评论邮箱生成可配置镜像源的 Gravatar URL；公开响应不返回邮箱。
- S3 兼容 POST policy 直传，图片字节不经过 API。
- 请求限制、登录限流、日志脱敏、慢查询追踪和 PostgreSQL 查询优化。

## 快速开始

需要 Go 1.25、PostgreSQL 12+、`psql`、一个 S3 兼容存储和 Cloudflare Turnstile 密钥。当前执行计划已在 PostgreSQL 18 上验证。

```powershell
Copy-Item .env.example .env
# 编辑 .env，替换数据库、JWT、Turnstile 和 S3 配置

$env:DATABASE_URL = "postgres://postgres:password@localhost:5432/simpleblog?sslmode=disable"
Get-ChildItem migrations/*.sql | Sort-Object Name | ForEach-Object {
    psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f $_.FullName
}

$env:ADMIN_PASSWORD = "replace-with-a-strong-password"
go run ./cmd/admin -username admin -nickname Administrator
Remove-Item Env:ADMIN_PASSWORD

go run ./cmd/server
```

服务默认监听 `localhost:8080`。检查服务：

```powershell
Invoke-RestMethod http://localhost:8080/service/ping
```

应用处理的业务结果始终使用 HTTP 200；客户端必须读取响应体 `code`：

```json
{"code": 200, "msg": "success", "data": {}}
```

## 文档

- [配置说明](docs/configuration.md)
- [运行与部署](docs/operations.md)
- [API 契约](docs/api.md)
- [程序架构](docs/architecture.md)
- [数据库设计](docs/database.md)
- [权限模型](docs/permissions.md)
- [测试与验收](docs/testing.md)
- [PostgreSQL 优化记录](docs/postgresql-optimization.md)
- [已知边界](docs/limitations.md)
- [Milestone 状态](docs/milestones.md)

## 开发验证

```powershell
gofmt -d cmd internal migrations
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

需要真实 PostgreSQL 或 S3 的测试默认跳过，启用方法见[测试与验收](docs/testing.md)。
