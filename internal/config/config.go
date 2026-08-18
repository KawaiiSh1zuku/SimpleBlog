package config

import (
	"errors"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	Turnstile TurnstileConfig
	S3        S3Config
}

type AppConfig struct {
	Loglevel string `env:"LOG_LEVEL" envDefault:"INFO"`
	Debug    bool   `env:"DEBUG" envDefault:"false"`
}

type ServerConfig struct {
	Host string `env:"SERVER_HOST" envDefault:"localhost"`
	Port string `env:"SERVER_PORT" envDefault:"8080"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	Username string `env:"DB_USER" envDefault:"postgres"`
	Password string `env:"DB_PASSWORD" envDefault:"password"`
	DBName   string `env:"DB_NAME" envDefault:"mydb"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"disable"`
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET" envDefault:"secret"`
}

type TurnstileConfig struct {
	SiteKey string `env:"TURNSTILE_SITEKEY" envDefault:"secret"`
	Secret  string `env:"TURNSTILE_SECRET" envDefault:"secret"`
}

type S3Config struct {
	Endpoint      string `env:"S3_ENDPOINT" envDefault:"localhost:9000"`
	Region        string `env:"S3_REGION" envDefault:"us-east-1"`
	Bucket        string `env:"S3_BUCKET" envDefault:"simpleblog"`
	AccessKey     string `env:"S3_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey     string `env:"S3_SECRET_KEY" envDefault:"minioadmin"`
	UseSSL        bool   `env:"S3_USE_SSL" envDefault:"false"`
	PublicBaseURL string `env:"S3_PUBLIC_BASE_URL" envDefault:"http://localhost:9000/simpleblog"`
}

// LoadConfig 加载 .env 文件（若存在）并从环境变量解析配置
func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // 忽略错误，若无 .env 则仅使用系统环境变量
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 在依赖初始化前检查配置，避免服务带着无效或生产默认值启动。
func (c *Config) Validate() error {
	var errs []error

	if !isLogLevel(c.App.Loglevel) {
		errs = append(errs, fmt.Errorf("invalid LOG_LEVEL %q", c.App.Loglevel))
	}
	if strings.TrimSpace(c.Server.Host) == "" {
		errs = append(errs, errors.New("SERVER_HOST must not be empty"))
	}
	if err := validatePort("SERVER_PORT", c.Server.Port); err != nil {
		errs = append(errs, err)
	}

	if strings.TrimSpace(c.Database.Host) == "" {
		errs = append(errs, errors.New("DB_HOST must not be empty"))
	}
	if err := validatePort("DB_PORT", c.Database.Port); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(c.Database.Username) == "" {
		errs = append(errs, errors.New("DB_USER must not be empty"))
	}
	if c.Database.Password == "" {
		errs = append(errs, errors.New("DB_PASSWORD must not be empty"))
	}
	if strings.TrimSpace(c.Database.DBName) == "" {
		errs = append(errs, errors.New("DB_NAME must not be empty"))
	}
	if !isSSLMode(c.Database.SSLMode) {
		errs = append(errs, fmt.Errorf("invalid DB_SSLMODE %q", c.Database.SSLMode))
	}
	if strings.TrimSpace(c.S3.Endpoint) == "" {
		errs = append(errs, errors.New("S3_ENDPOINT must not be empty"))
	}
	if strings.TrimSpace(c.S3.Region) == "" {
		errs = append(errs, errors.New("S3_REGION must not be empty"))
	}
	if strings.TrimSpace(c.S3.Bucket) == "" {
		errs = append(errs, errors.New("S3_BUCKET must not be empty"))
	}
	if strings.TrimSpace(c.S3.AccessKey) == "" {
		errs = append(errs, errors.New("S3_ACCESS_KEY must not be empty"))
	}
	if c.S3.SecretKey == "" {
		errs = append(errs, errors.New("S3_SECRET_KEY must not be empty"))
	}
	if err := validateHTTPURL("S3_PUBLIC_BASE_URL", c.S3.PublicBaseURL); err != nil {
		errs = append(errs, err)
	}

	if c.App.Debug {
		return errors.Join(errs...)
	}

	if c.Database.Password == "password" {
		errs = append(errs, errors.New("DB_PASSWORD must not use the production default"))
	}
	if len(c.JWT.Secret) < 32 || c.JWT.Secret == "secret" {
		errs = append(errs, errors.New("JWT_SECRET must contain at least 32 characters in production"))
	}
	if c.Turnstile.Secret == "" || c.Turnstile.Secret == "secret" {
		errs = append(errs, errors.New("TURNSTILE_SECRET must be configured in production"))
	}
	if c.S3.AccessKey == "minioadmin" || c.S3.SecretKey == "minioadmin" {
		errs = append(errs, errors.New("S3 credentials must not use the production defaults"))
	}

	return errors.Join(errs...)
}

// Address 返回适用于 http.Server 的监听地址。
func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

// ConnectionString 构造经过 URL 编码的 PostgreSQL 连接字符串。
func (c DatabaseConfig) ConnectionString() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   "/" + strings.TrimPrefix(c.DBName, "/"),
	}
	query := u.Query()
	query.Set("sslmode", c.SSLMode)
	u.RawQuery = query.Encode()
	return u.String()
}

func validatePort(name string, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s must be an integer between 1 and 65535", name)
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL without query or fragment", name)
	}
	return nil
}

func isLogLevel(value string) bool {
	switch strings.ToUpper(value) {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return true
	default:
		return false
	}
}

func isSSLMode(value string) bool {
	switch value {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
