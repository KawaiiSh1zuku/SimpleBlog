# 数据库迁移

完整初始化、升级与回滚边界见 [运行与部署](../docs/operations.md)，测试方式见 [测试与验收](../docs/testing.md)。

PostgreSQL schema 使用有序 SQL 文件维护。初始化空数据库或升级时，只按文件名顺序执行尚未应用的 migration：

```powershell
Get-ChildItem migrations/*.sql | Sort-Object Name | ForEach-Object {
    psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f $_.FullName
}
```

仓库目前没有 migration history 表或自动 runner；升级环境必须由部署系统记录已应用版本，不能每次重新执行全部文件。

每个 migration 都包含事务。API 启动时不会自动执行 schema；部署流程必须先应用与目标应用版本匹配的新文件，再启动服务。

使用显式测试数据库验证全部 schema，且不保留测试对象：

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres"
go test ./migrations -run TestInitialSchema -v
```

测试在事务或独立连接中创建随机 schema，并在结束时回滚或删除。`000003_rss_links_security.sql` 增加友情链接、RSS 评论索引和评论纯文本约束；`000004_post_cover_search.sql` 增加文章头图字段并启用 `pg_trgm` 全文字段子串搜索索引。升级环境仍只执行尚未应用的文件。

执行 `000004` 的 migration 账号必须能在目标数据库运行 `CREATE EXTENSION IF NOT EXISTS pg_trgm`。扩展只需安装一次，应用运行账号不需要该 DDL 权限。
