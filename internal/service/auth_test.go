package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type authUserStoreStub struct {
	user *model.User
	err  error
}

func TestAuthServiceDefaultTokenTTL(t *testing.T) {
	service := NewAuthService(authUserStoreStub{}, jwtauth.New("HS256", []byte("secret"), nil))
	if service.TokenTTL != 24*time.Hour {
		t.Fatalf("TokenTTL = %v, want %v", service.TokenTTL, 24*time.Hour)
	}
}

func (stub authUserStoreStub) GetByUsername(context.Context, string) (*model.User, error) {
	return stub.user, stub.err
}

func TestAuthServiceLoginIssuesExpiringToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{ID: uuid.New(), PasswordHash: string(hash), Role: model.RoleAdmin}
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	service := NewAuthService(authUserStoreStub{user: user}, tokenAuth)
	service.TokenTTL = 30 * time.Minute

	before := time.Now().UTC()
	result, err := service.Login(context.Background(), "admin", "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token == "" {
		t.Fatal("Login() returned an empty token")
	}
	if result.ExpiresAt.Before(before.Add(29 * time.Minute)) {
		t.Fatalf("unexpected expiry: %v", result.ExpiresAt)
	}

	token, err := jwtauth.VerifyToken(tokenAuth, result.Token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
	subject, ok := token.Subject()
	if !ok || subject != user.ID.String() {
		t.Fatalf("subject = %q, want %q", subject, user.ID)
	}
	issuer, ok := token.Issuer()
	if !ok || issuer != "simpleblog" {
		t.Fatalf("issuer = %q, want simpleblog", issuer)
	}
}

func TestAuthServiceLoginHidesCredentialFailureCause(t *testing.T) {
	for _, test := range []struct {
		name  string
		store authUserStoreStub
	}{
		{name: "missing user", store: authUserStoreStub{err: pgx.ErrNoRows}},
		{name: "wrong password", store: authUserStoreStub{user: &model.User{PasswordHash: "$2a$04$invalid"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewAuthService(test.store, jwtauth.New("HS256", []byte("secret"), nil))
			_, err := service.Login(context.Background(), "user", "bad-password")
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v", err)
			}
		})
	}
}
