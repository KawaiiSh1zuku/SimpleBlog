# SimpleBlog 已知边界

本文记录当前实现有意保留的最小化边界。

## 1. 当前规模边界

- 公开评论接口一次返回文章的完整评论树，不分页；回复深度不设上限。热门文章需要评论分页时，应先设计根评论与回复分页契约，不能直接截断树。
- 用户与分类列表不分页，适用于单站点的小规模管理数据；文章和后台评论已经使用 keyset pagination。
- 登录限流保存在单进程内存中，10 次/分钟/IP，不在多实例间共享。当前单实例部署不引入 Redis。
- `/service/ping` 只验证 HTTP 进程存活，不检查 PostgreSQL、S3 或 Turnstile 就绪状态。

## 2. 内容安全边界

- 文章由已登录作者维护，Markdown 中允许可执行原生 HTML；部署方必须把 contributor 视为可发布站点脚本内容的受信任角色。
- 评论是纯文本，后端拒绝 HTML 尖括号并中和历史数据；前端仍必须使用文本节点渲染，不能把字符串送入 `innerHTML` 或不安全属性。
- 友情链接只允许 HTTP(S) URL；服务端不抓取目标页面或 favicon，因此没有由此产生的 SSRF。

## 3. 运维边界

- 仓库没有 migration history 表、自动 runner 或 down migration，部署系统负责记录已应用文件和数据库备份。
- S3 只负责签发直传 policy，不维护 uploads/media 表，也不自动清理已上传但未被文章引用的对象。
- JWT 固定 24 小时且没有 refresh token 或服务端 session；删除用户或修改角色通过每次请求重新读取用户立即生效。
- `TURNSTILE_SITEKEY` 只作为配置保留，后端不向前端下发；前端必须独立配置同一站点的 site key。
- 评论纯文本数据库约束以 `NOT VALID` 加入，升级前历史行需在清理后由运维显式 `VALIDATE CONSTRAINT`；API 输出在验证前也会安全降级。
