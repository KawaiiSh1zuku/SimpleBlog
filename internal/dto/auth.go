package dto

// LoginRequest 用于登录请求的结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
