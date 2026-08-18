package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CurrentUserStore interface {
	GetByID(context.Context, uuid.UUID) (*model.User, error)
}

type currentUserContextKey struct{}

func Authenticate(users CurrentUserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, claims, tokenErr := jwtauth.FromContext(r.Context())
			subject, ok := claims["sub"].(string)
			userID, idErr := uuid.Parse(subject)
			if tokenErr != nil || !ok || idErr != nil {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}

			user, err := users.GetByID(r.Context(), userID)
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if err != nil {
				slog.ErrorContext(r.Context(), "load authenticated user", "error", err)
				response.Error(w, http.StatusInternalServerError, "internal server error")
				return
			}
			ctx := context.WithValue(r.Context(), currentUserContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...model.Role) func(http.Handler) http.Handler {
	allowed := make(map[model.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				response.Error(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CurrentUser(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(currentUserContextKey{}).(*model.User)
	return user, ok
}
