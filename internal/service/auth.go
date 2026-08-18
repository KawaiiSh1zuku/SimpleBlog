package service

import (
	"context"
	"errors"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/go-chi/jwtauth/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	AuthRepo  *repository.AuthRepository
	TokenAuth *jwtauth.JWTAuth
}

func NewAuthService(
	authRepo *repository.AuthRepository,
	tokenAuth *jwtauth.JWTAuth,
) *AuthService {
	return &AuthService{
		AuthRepo:  authRepo,
		TokenAuth: tokenAuth,
	}
}

func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	// 1. 根据用户名查询用户
	user, err := s.AuthRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return "", errors.New("用户名或密码错误")
	}

	// 3. 创建 JWT Payload
	claims := map[string]interface{}{
		"sub":  user.ID,
		"role": user.Role,
	}

	// 4. 签发 JWT
	_, token, err := s.TokenAuth.Encode(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}
