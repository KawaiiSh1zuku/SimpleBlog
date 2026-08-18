# SimpleBlog 权限设计

## 1. 角色

- `admin`：站点级管理角色。
- `contributor`：内容贡献者，只管理自己拥有的资源。
- 访客不是 users 表中的角色，表示未登录请求。

第一版不引入细粒度权限表和 PostgreSQL RLS。权限规则集中在 service 层，避免规则散落在 handler 和 SQL 中。

## 2. 权限矩阵

| 操作 | 访客 | contributor | admin |
|---|---:|---:|---:|
| 查看已发布文章 | ✓ | ✓ | ✓ |
| 查看公开分类、标签和评论 | ✓ | ✓ | ✓ |
| 提交顶级评论或回复 | ✓ | ✓ | ✓ |
| 查看后台文章列表 |  | 仅自己的 | 全部 |
| 创建文章 |  | ✓ | ✓ |
| 编辑、删除自己的文章 |  | ✓ | ✓ |
| 发布、撤回自己的文章 |  | ✓ | ✓ |
| 操作他人的文章 |  |  | ✓ |
| 获取 S3 预签名地址 |  | ✓ | ✓ |
| 管理自己文章下的评论 |  | ✓ | ✓ |
| 管理所有评论 |  |  | ✓ |
| 管理分类 |  |  | ✓ |
| 创建、查看、修改、删除用户 |  |  | ✓ |

当前设计允许 contributor 自行发布自己的文章。如果后续需要审核流，应增加 `pending_review` 状态和提交审核动作。

## 3. 权限执行位置

### Middleware

- 验证 JWT 签名和过期时间。
- 从 sub 加载当前未删除用户。
- 把当前用户 ID 和数据库中的最新 role 写入 context。
- 对纯 admin 路由执行快速角色拦截。

### Service

- 判断文章和评论所属关系。
- 判断 contributor 是否只能操作自己的资源。
- 保护最后一个 admin。
- 校验文章状态转换。
- 确保评论父节点与文章一致。

### Repository

- 只实现数据查询和原子写入。
- 对 contributor 的更新和删除可以额外在 SQL 中加入 `author_id = $actorID`，形成纵深保护。
- 不解析 JWT，也不生成 HTTP 响应。

## 4. 用户管理规则

- 只有 admin 可以创建用户和设置角色。
- username 在未删除用户中大小写不敏感唯一。
- 修改密码必须重新计算 bcrypt hash，DTO 不返回 password_hash。
- 删除用户是软删除；被删除用户立即不能登录或继续使用旧 JWT。
- 不允许删除或降级最后一个 admin。
- 建议禁止 admin 删除自己；如允许，必须确保还有其他 admin。

## 5. 文章权限规则

- contributor 创建的文章 author_id 固定为当前用户，不能由请求体指定。
- contributor 不能通过更新 DTO 修改 author_id。
- admin 可以管理全部文章，但第一版不提供作者转移功能；需要时应设计独立动作。
- 分类是全局资源，只允许 admin 管理。
- contributor 可以为自己的文章选择现有分类并维护 tags。

## 6. 评论权限规则

- 匿名评论不创建用户记录。
- 评论创建需要 nickname、email、content 和 Turnstile 验证。
- contributor 只能隐藏、恢复或删除自己文章下的评论。
- admin 可以管理所有评论。
- hidden 可以恢复；deleted 是清除个人信息和正文后的不可恢复墓碑，公开端固定显示“该评论已被删除”，并继续展示子回复。
- 公开响应不返回 email，日志也不得记录 email。

## 7. S3 权限规则

- 只有已登录且未删除的 admin/contributor 可以申请预签名地址。
- 服务端生成 object key，并把用户 ID 放入对象路径。
- 客户端不能指定 bucket、凭据或任意路径。
- 签名必须限制有效期、Content-Type 和文件大小能力范围。
- 任何 S3 access key、secret key 和完整签名 URL 都不得写入日志。
