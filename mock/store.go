package main

import (
	"sort"
	"sync"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type mockStore struct {
	mu         sync.RWMutex
	users      []model.User
	passwords  map[uuid.UUID]string
	categories []model.Category
	posts      []model.Post
	comments   []model.Comment
	links      []model.FriendLink
}

func (s *mockStore) userByID(id uuid.UUID) (*model.User, bool) {
	for index := range s.users {
		if s.users[index].ID == id {
			return &s.users[index], true
		}
	}
	return nil, false
}

func (s *mockStore) userByUsername(username string) (*model.User, bool) {
	for index := range s.users {
		if s.users[index].Username == username {
			return &s.users[index], true
		}
	}
	return nil, false
}

func (s *mockStore) categoryByID(id uuid.UUID) (*model.Category, bool) {
	for index := range s.categories {
		if s.categories[index].ID == id {
			return &s.categories[index], true
		}
	}
	return nil, false
}

func (s *mockStore) postByID(id uuid.UUID) (*model.Post, bool) {
	for index := range s.posts {
		if s.posts[index].ID == id {
			return &s.posts[index], true
		}
	}
	return nil, false
}

func (s *mockStore) postBySlug(slug string) (*model.Post, bool) {
	for index := range s.posts {
		if s.posts[index].Slug == slug {
			return &s.posts[index], true
		}
	}
	return nil, false
}

func (s *mockStore) commentByID(id uuid.UUID) (*model.Comment, bool) {
	for index := range s.comments {
		if s.comments[index].ID == id {
			return &s.comments[index], true
		}
	}
	return nil, false
}

func (s *mockStore) linkByID(id uuid.UUID) (*model.FriendLink, bool) {
	for index := range s.links {
		if s.links[index].ID == id {
			return &s.links[index], true
		}
	}
	return nil, false
}

func sortPosts(posts []model.Post) {
	sort.SliceStable(posts, func(i, j int) bool {
		if posts[i].UpdatedAt.Equal(posts[j].UpdatedAt) {
			return posts[i].ID.String() > posts[j].ID.String()
		}
		return posts[i].UpdatedAt.After(posts[j].UpdatedAt)
	})
}

func sortComments(comments []model.Comment) {
	sort.SliceStable(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID.String() > comments[j].ID.String()
		}
		return comments[i].CreatedAt.After(comments[j].CreatedAt)
	})
}

func removeAt[T any](values []T, index int) []T {
	return append(values[:index], values[index+1:]...)
}
