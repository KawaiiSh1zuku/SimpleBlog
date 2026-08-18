# SimpleBlog 已知边界

本文记录当前实现有意保留的最小化边界，以及文档核对中发现但本轮未修改的代码偏差。

## 1. 待修正偏差

评论正文契约为最多 10000 个 Unicode 字符。当前 `CommentService` 对 `TrimSpace(content)` 校验长度，但保存原始 `content`，因此大量首尾空白可在 2 MiB 全局请求体限制内绕过原始字符串长度上限。建议在 M7 前统一“校验并保存原文”或“校验并保存 trim 后正文”的语义，并补充回归测试。

## 2. 当前规模边界

- 公开评论接口一次返回文章的完整评论树，不分页；回复深度不设上限。热门文章需要评论分页时，应先设计根评论与回复分页契约，不能直接截断树。
- 用户与分类列表不分页，适用于单站点的小规模管理数据；文章和后台评论已经使用 keyset pagination。
- 登录限流保存在单进程内存中，10 次/分钟/IP，不在多实例间共享。当前单实例部署不引入 Redis。
- `/service/ping` 只验证 HTTP 进程存活，不检查 PostgreSQL、S3 或 Turnstile 就绪状态。

## 3. 运维边界

- 仓库没有 migration history 表、自动 runner 或 down migration，部署系统负责记录已应用文件和数据库备份。
- S3 只负责签发直传 policy，不维护 uploads/media 表，也不自动清理已上传但未被文章引用的对象。
- JWT 固定 24 小时且没有 refresh token 或服务端 session；删除用户或修改角色通过每次请求重新读取用户立即生效。
- `TURNSTILE_SITEKEY` 只作为配置保留，后端不向前端下发；前端必须独立配置同一站点的 site key。

