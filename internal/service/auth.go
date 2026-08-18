package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const defaultTokenTTL = 2 * time.Hour

type AuthUserStore interface {
	GetByUsername(context.Context, string) (*model.User, error)
}

type AuthService struct {
	Users     AuthUserStore
	TokenAuth *jwtauth.JWTAuth
	TokenTTL  time.Duration
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
}

func NewAuthService(users AuthUserStore, tokenAuth *jwtauth.JWTAuth) *AuthService {
	return &AuthService{Users: users, TokenAuth: tokenAuth, TokenTTL: defaultTokenTTL}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.Users.GetByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.TokenTTL)
	claims := map[string]any{
		"sub":  user.ID.String(),
		"role": string(user.Role),
		"iss":  "simpleblog",
	}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, expiresAt)
	_, token, err := s.TokenAuth.Encode(claims)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, ExpiresAt: expiresAt}, nil
}
