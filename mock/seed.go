package main

import (
	"fmt"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

var seedEpoch = time.Date(2026, time.August, 1, 8, 0, 0, 0, time.UTC)

func seedID(name string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("simpleblog-mock:"+name))
}

func newSeedStore() *mockStore {
	users := seedUsers()
	categories := seedCategories()
	posts := seedPosts(users, categories)
	return &mockStore{
		users:      users,
		passwords:  map[uuid.UUID]string{users[0].ID: "admin123456", users[1].ID: "writer123456", users[2].ID: "editor123456"},
		categories: categories,
		posts:      posts,
		comments:   seedComments(posts),
		links:      seedLinks(),
	}
}

func seedUsers() []model.User {
	return []model.User{
		{ID: seedID("user-admin"), Username: "admin", Nickname: "站点管理员", AvatarURL: "http://localhost:8090/mock-assets/avatar-admin.png", Role: model.RoleAdmin, CreatedAt: seedEpoch, UpdatedAt: seedEpoch},
		{ID: seedID("user-writer"), Username: "writer", Nickname: "林墨", AvatarURL: "http://localhost:8090/mock-assets/avatar-writer.png", Role: model.RoleContributor, CreatedAt: seedEpoch.Add(time.Hour), UpdatedAt: seedEpoch.Add(time.Hour)},
		{ID: seedID("user-editor"), Username: "editor", Nickname: "周遥", AvatarURL: "", Role: model.RoleContributor, CreatedAt: seedEpoch.Add(2 * time.Hour), UpdatedAt: seedEpoch.Add(2 * time.Hour)},
	}
}

func seedCategories() []model.Category {
	definitions := []struct{ name, slug, description string }{
		{"后端工程", "backend", "Go、数据库与服务端工程实践"},
		{"前端体验", "frontend", "浏览器、交互和性能"},
		{"运维手记", "operations", "部署、监控与故障处理"},
		{"随笔", "notes", "没有固定主题的短记录"},
		{"未归档", "uncategorized", "用于覆盖暂未归类的内容"},
	}
	result := make([]model.Category, 0, len(definitions))
	for index, definition := range definitions {
		createdAt := seedEpoch.Add(time.Duration(index) * time.Hour)
		result = append(result, model.Category{ID: seedID("category-" + definition.slug), Name: definition.name, Slug: definition.slug, Description: definition.description, CreatedAt: createdAt, UpdatedAt: createdAt})
	}
	return result
}

func seedPosts(users []model.User, categories []model.Category) []model.Post {
	titles := []string{
		"用 Go 构建可观测的 HTTP 服务",
		"PostgreSQL 游标分页的边界条件",
		"React 评论树的异步锚点处理",
		"一次从超时到索引命中的排障记录",
		"Markdown、LaTeX 与原生 HTML 共存",
		"S3 直传为什么要由服务端签名",
		"小型博客的权限模型",
		"草稿：新的站点导航设计",
		"草稿：还没有头图的长文",
		"草稿：空标签与最短摘要",
	}
	slugs := []string{"observable-go-http", "postgres-cursor-pagination", "react-comment-anchors", "slow-query-debugging", "markdown-latex-html", "s3-presigned-post", "blog-permission-model", "draft-site-navigation", "draft-without-cover", "draft-minimal-metadata"}
	tags := [][]string{{"go", "http", "observability"}, {"postgresql", "pagination"}, {"react", "comments"}, {"postgresql", "operations"}, {"markdown", "latex", "html"}, {"s3", "security"}, {"authorization", "design"}, {"ux", "navigation"}, {"writing"}, {}}
	result := make([]model.Post, 0, len(titles))
	for index := range titles {
		createdAt := seedEpoch.Add(time.Duration(index) * 24 * time.Hour)
		updatedAt := createdAt.Add(time.Duration(index+1) * time.Hour)
		status := model.PostStatusPublished
		var publishedAt *time.Time
		if index < 7 {
			value := createdAt.Add(12 * time.Hour)
			publishedAt = &value
		} else {
			status = model.PostStatusDraft
		}
		var cover *string
		if index%2 == 1 {
			value := fmt.Sprintf("http://localhost:8090/mock-assets/cover-%02d.png", index+1)
			cover = &value
		}
		content := fmt.Sprintf("# %s\n\n这是第 %d 篇 mock 文章，用于验证 Markdown、中文与 `code`。\n\n## 示例\n\n- 固定种子数据\n- 可重复联调\n- 评论数量为 %d\n\n```go\nfmt.Println(\"mock post %d\")\n```\n", titles[index], index+1, index, index+1)
		if index == 4 {
			content += "\n行内公式 $E=mc^2$，以及原生 HTML：<mark>可执行内容由前端决定如何渲染</mark>。\n"
		}
		if index == 8 {
			content += "\n![正文首图](http://localhost:8090/mock-assets/body-first.png)\n"
		}
		result = append(result, model.Post{
			ID: seedID(fmt.Sprintf("post-%02d", index+1)), AuthorID: users[index%len(users)].ID, CategoryID: categories[index%len(categories)].ID,
			Slug: slugs[index], Title: titles[index], Summary: fmt.Sprintf("第 %d 篇固定 mock 文章，含 %d 条评论。", index+1, index), CoverImg: cover,
			Content: content, Tags: tags[index], Status: status, PublishedAt: publishedAt, CreatedAt: createdAt, UpdatedAt: updatedAt,
		})
	}
	sortPosts(result)
	return result
}

func seedComments(posts []model.Post) []model.Comment {
	// Posts are sorted by update time, so comment counts follow their original seed number encoded in the ID lookup.
	counts := make(map[uuid.UUID]int, len(posts))
	for index := 1; index <= 10; index++ {
		counts[seedID(fmt.Sprintf("post-%02d", index))] = index - 1
	}
	result := make([]model.Comment, 0, 45)
	for _, post := range posts {
		count := counts[post.ID]
		var rootID uuid.UUID
		for index := 0; index < count; index++ {
			id := seedID(fmt.Sprintf("comment-%s-%02d", post.Slug, index+1))
			var parentID *uuid.UUID
			if index == 0 {
				rootID = id
			} else if index%3 != 0 {
				value := rootID
				parentID = &value
			}
			status := model.CommentStatusVisible
			var moderatedAt *time.Time
			switch index % 6 {
			case 1:
				status = model.CommentStatusHidden
				value := post.CreatedAt.Add(time.Duration(index+2) * time.Hour)
				moderatedAt = &value
			case 2:
				status = model.CommentStatusDeleted
				value := post.CreatedAt.Add(time.Duration(index+2) * time.Hour)
				moderatedAt = &value
			}
			result = append(result, model.Comment{
				ID: id, PostID: post.ID, ParentID: parentID, Nickname: fmt.Sprintf("访客%02d", index+1), Email: fmt.Sprintf("visitor%02d@example.com", index+1),
				Content: fmt.Sprintf("这是《%s》的第 %d 条评论。", post.Title, index+1), Status: status,
				CreatedAt: post.CreatedAt.Add(time.Duration(index+1) * time.Hour), ModeratedAt: moderatedAt,
			})
		}
	}
	sortComments(result)
	return result
}

func seedLinks() []model.FriendLink {
	definitions := []struct{ name, description, url, favicon string }{
		{"Go 官方站点", "Go 语言文档与下载", "https://go.dev", "http://localhost:8090/mock-assets/favicon-go.png"},
		{"PostgreSQL", "可靠的开源关系型数据库", "https://www.postgresql.org", "http://localhost:8090/mock-assets/favicon-postgresql.png"},
		{"React", "用于 Web 和原生界面的库", "https://react.dev", "http://localhost:8090/mock-assets/favicon-react.png"},
		{"没有图标的友站", "覆盖 favicon_url 为空的展示状态", "https://example.com/friend", ""},
		{"长描述友站", "这是一条较长的友链说明，用来检查列表换行、窄屏布局和文本截断是否符合前端设计预期。", "https://example.org", "http://localhost:8090/mock-assets/favicon-long.png"},
	}
	result := make([]model.FriendLink, 0, len(definitions))
	for index, definition := range definitions {
		createdAt := seedEpoch.Add(time.Duration(index) * time.Hour)
		result = append(result, model.FriendLink{ID: seedID(fmt.Sprintf("link-%02d", index+1)), Name: definition.name, Description: definition.description, URL: definition.url, FaviconURL: definition.favicon, CreatedAt: createdAt, UpdatedAt: createdAt})
	}
	return result
}
