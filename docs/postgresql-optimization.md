# PostgreSQL 查询优化记录

## 1. 目标与边界

M6 针对实际 API 查询优化 PostgreSQL 访问，重点检查：索引与筛选/排序是否匹配、是否读取不必要的大字段或敏感字段、是否存在 N+1、外部输入是否参数化，以及最后管理员事务的锁竞争行为。

以下结果是本机 PostgreSQL 18 上的可重复执行计划验证，不是生产容量基准或延迟 SLA。真实部署仍应结合数据分布、缓存状态、硬件和 `pg_stat_statements` 复核。

## 2. 代表性数据

集成测试在随机临时 schema 中依次应用全部 migration，并生成：

| 表 | 行数 | 关键分布 |
|---|---:|---|
| users | 10,000 | 1 admin，其余 contributor |
| categories | 1,000 | slug 唯一 |
| posts | 20,000 | 80% published；每篇正文 1 KiB；少量 rare 标签 |
| comments | 50,000 | 均匀关联文章 |

数据写入后对四张表执行 `ANALYZE`。每条关键查询使用 `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)`，测试要求命中指定索引，并禁止高增长主表出现可避免的 `Seq Scan`。

## 3. 结果摘要

| 查询 | 主要索引 | 本次执行时间 | shared hit |
|---|---|---:|---:|
| 公开文章列表 | `posts_public_idx` | 0.159 ms | 133 |
| 分类文章列表 | `categories_slug_uq`、`posts_category_public_idx` | 0.094 ms | 25 |
| 稀有标签筛选 | `posts_published_tags_gin` | 2.855 ms | 58 |
| contributor 文章列表 | `posts_author_updated_idx` | 0.066 ms | 53 |
| admin 文章列表 | `posts_updated_idx` | 0.037 ms | 11 |
| 公开评论树 | `posts_slug_uq`、`comments_post_created_idx` | 0.060 ms | 9 |
| 后台评论列表 | `comments_created_idx` | 0.422 ms | 156 |
| contributor 评论列表 | `posts_author_updated_idx`、`comments_post_created_idx` | 2.054 ms | 1,303 |
| 登录用户查询 | `users_username_active_uq` | 0.016 ms | 3 |

九条查询均命中预期索引，高增长表未出现被测试禁止的顺序扫描。时间和缓冲区数字只记录本次样本，验收依据是执行计划形状与索引匹配，不以亚毫秒结果推断生产性能。

## 4. SQL 与数据最小化

- 公开文章列表与详情使用不同投影；列表不读取 `content`。
- 后台文章列表返回 `PostSummary`，编辑器按 ID 获取正文。
- 用户安全查询不读取 `password_hash`；只有登录查询读取。
- 公开评论查询读取 email 仅用于生成 Gravatar URL，公开 DTO 与日志不包含 email。
- 文章和评论后台列表采用 `(时间, id)` keyset pagination，默认 50、最大 100，不使用随页数退化的 offset。
- 评论树一次读取文章全部节点并在内存中组装，不按层级查询，因此不产生 N+1。
- contributor 评论列表先物化其文章集合，再以 LATERAL 查询命中逐文章评论索引；代表性数据下避免了原计划扫描约 5,001 条全站评论并进行同数量文章主键查找。
- SQL 值全部通过 pgx 参数传递；可选条件只拼接服务端固定片段和编号占位符。

## 5. 事务、错误与可观测性

- 最后管理员保护使用 transaction-level advisory lock，并设置事务内 3 秒 `lock_timeout`；并发删除两个 admin 的集成测试验证恰有一个成功、另一个返回最后管理员冲突。
- SQLSTATE 将输入错误、冲突和可重试错误分开；序列化失败、死锁、锁不可用和连接耗尽映射为业务 code 503。
- `DB_SLOW_QUERY_MS` 默认 250 ms，范围 1 到 60000。慢查询日志只包含规范化 SQL 的 SHA-256 短指纹、耗时、命令标签、SQLSTATE 和 request ID，不记录 SQL 文本、参数或原始 PostgreSQL Detail。

## 6. 复现

使用独立测试数据库 URL：

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres?sslmode=disable"
go test ./internal/repository -run TestRepresentativePostgreSQLQueryPlans -count=1 -v
Remove-Item Env:TEST_DATABASE_URL
```

测试只在随机临时 schema 中工作并在结束时删除该 schema。也可按仓库集成测试约定启用 `SIMPLEBLOG_TEST_LOCAL_DB`。执行计划发生变化时，应先判断 PostgreSQL 版本、统计信息与数据分布差异，不能为了强制命中索引而关闭 planner 的正常选择。
