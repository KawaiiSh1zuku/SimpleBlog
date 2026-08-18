package dto

import (
	"encoding/xml"
	"net/url"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
)

type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	GUID        RSSGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
}

type RSSGUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

func NewArticleRSS(siteName, siteDescription, baseURL string, articles []model.FeedArticle) RSS {
	items := make([]RSSItem, 0, len(articles))
	for _, article := range articles {
		link := articleURL(baseURL, article.Slug)
		items = append(items, RSSItem{
			Title: safePlainText(article.Title), Link: link, Description: safePlainText(article.Summary),
			GUID: RSSGUID{Value: "urn:uuid:" + article.ID.String()}, PubDate: rssDate(article.PublishedAt),
		})
	}
	return newRSS(siteName+" - 文章", siteDescription, baseURL, items)
}

func NewCommentRSS(siteName, siteDescription, baseURL string, comments []model.FeedComment) RSS {
	items := make([]RSSItem, 0, len(comments))
	for _, comment := range comments {
		link := articleURL(baseURL, comment.PostSlug) + "#comment-" + comment.ID.String()
		items = append(items, RSSItem{
			Title: safePlainText(comment.Nickname) + " 评论了 " + safePlainText(comment.PostTitle),
			Link:  link, Description: safePlainText(comment.Content),
			GUID: RSSGUID{Value: "urn:uuid:" + comment.ID.String()}, PubDate: rssDate(comment.CreatedAt),
		})
	}
	return newRSS(siteName+" - 评论", siteDescription, baseURL, items)
}

func MarshalRSS(feed RSS) ([]byte, error) {
	body, err := xml.Marshal(feed)
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func newRSS(title, description, baseURL string, items []RSSItem) RSS {
	channel := RSSChannel{Title: safePlainText(title), Link: strings.TrimRight(baseURL, "/"), Description: safePlainText(description), Items: items}
	if len(items) > 0 {
		channel.LastBuildDate = items[0].PubDate
	}
	return RSS{Version: "2.0", Channel: channel}
}

func articleURL(baseURL, slug string) string {
	return strings.TrimRight(baseURL, "/") + "/posts/" + url.PathEscape(slug)
}

func rssDate(value time.Time) string { return value.UTC().Format(time.RFC1123Z) }
