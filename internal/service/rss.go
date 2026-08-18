package service

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
)

type FeedStore interface {
	ListArticles(context.Context) ([]model.FeedArticle, error)
	ListComments(context.Context) ([]model.FeedComment, error)
}

type RSSSite struct{ Name, Description, BaseURL string }

type RSSService struct {
	Feeds FeedStore
	Site  RSSSite
}

func NewRSSService(feeds FeedStore, site RSSSite) *RSSService {
	return &RSSService{Feeds: feeds, Site: site}
}

func (s *RSSService) Articles(ctx context.Context) ([]model.FeedArticle, error) {
	items, err := s.Feeds.ListArticles(ctx)
	return items, mapStoreAvailability(err)
}

func (s *RSSService) Comments(ctx context.Context) ([]model.FeedComment, error) {
	items, err := s.Feeds.ListComments(ctx)
	return items, mapStoreAvailability(err)
}
