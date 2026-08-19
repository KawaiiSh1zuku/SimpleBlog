# SimpleBlog 安全边界

## SQL 注入

- repository 的外部值全部通过 pgx `$n` 参数传入；可选查询只拼接服务端固定 SQL 片段和占位符编号。
- 友情链接、RSS、文章、评论等查询不接受客户端提供的列名、排序表达式或原始 SQL。
- 集成测试会把 `x'); DROP TABLE friend_links; --` 作为普通字段写入并原样读回，再确认表仍存在。
- 慢查询摘要不记录 SQL 参数；普通错误日志保留完整 error 链，PostgreSQL Detail 可能包含输入值，因此日志必须作为受控调试数据管理。

代码审计和回归测试能保证当前已知查询路径不存在直接 SQL 拼接入口；以后增加动态排序或筛选时，必须使用服务端枚举映射，不能把请求字符串放入 SQL 结构位置。

## XSS 与内容信任边界

文章由已登录的 admin/contributor 发布，Markdown 中的可执行 HTML 是产品明确允许的能力，因此后端不清洗文章正文。部署方必须把 contributor 视为可以发布站点脚本的受信任作者。

匿名评论是不可信输入，按纯文本处理：

- 写入时统一换行、trim，拒绝 `<`、`>` 和除换行/制表符外的控制字符。
- PostgreSQL CHECK 纵深限制新写入；迁移使用 `NOT VALID` 以兼容旧数据。
- 公开与后台 DTO、评论 RSS 会再次中和旧行的尖括号并删除危险控制字符。
- JSON 使用 `encoding/json`，RSS 使用 `encoding/xml`；RSS 不拼接用户 XML。
- 全局返回 `X-Content-Type-Options: nosniff`。

前端仍必须使用 `textContent` 或框架默认文本插值渲染评论，禁止 `innerHTML`、危险属性插值或把评论拼成脚本。后端无法保护前端主动绕过文本渲染边界的行为。

## 外部 URL

友情链接和 favicon 只接受绝对 HTTP(S) URL并拒绝 userinfo，因而不接受 `javascript:`。服务端只存储并返回地址，不主动抓取目标内容，不形成 SSRF。`SITE_BASE_URL`、S3 公开基址和 Gravatar 基址均在启动时校验为 HTTP(S) URL。

## 验证

默认执行：

```powershell
go test ./...
go vet ./...
```

真实 PostgreSQL 安全与执行计划测试：

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres?sslmode=disable"
go test ./internal/repository ./migrations -count=1 -v
Remove-Item Env:TEST_DATABASE_URL
```
