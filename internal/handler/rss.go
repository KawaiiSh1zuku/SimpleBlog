package handler

import (
	"log/slog"
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

type RSSHandler struct{ RSS *service.RSSService }

func NewRSSHandler(rss *service.RSSService) *RSSHandler { return &RSSHandler{RSS: rss} }

func (h *RSSHandler) Articles(w http.ResponseWriter, r *http.Request) {
	items, err := h.RSS.Articles(r.Context())
	if err != nil {
		writeRSSError(w, r, err)
		return
	}
	h.write(w, r, dto.NewArticleRSS(h.RSS.Site.Name, h.RSS.Site.Description, h.RSS.Site.BaseURL, items))
}

func (h *RSSHandler) Comments(w http.ResponseWriter, r *http.Request) {
	items, err := h.RSS.Comments(r.Context())
	if err != nil {
		writeRSSError(w, r, err)
		return
	}
	h.write(w, r, dto.NewCommentRSS(h.RSS.Site.Name, h.RSS.Site.Description, h.RSS.Site.BaseURL, items))
}

func (h *RSSHandler) write(w http.ResponseWriter, r *http.Request, feed dto.RSS) {
	body, err := dto.MarshalRSS(feed)
	if err != nil {
		writeRSSError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func writeRSSError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "rss request failed", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
