package config

import (
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Loglevel string `env:"LOG_LEVEL" envDefault:"INFO"`
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
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET" envDefault:"secret"`
}

// LoadConfig 加载 .env 文件（若存在）并从环境变量解析配置
func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // 忽略错误，若无 .env 则仅使用系统环境变量
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
