package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestConfigValidateAllowsDevelopmentDefaults(t *testing.T) {
	cfg := validConfig()
	cfg.App.Debug = true
	cfg.Database.Password = "password"
	cfg.JWT.Secret = "secret"
	cfg.Turnstile.Secret = "secret"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned an error in debug mode: %v", err)
	}
}

func TestConfigValidateRejectsProductionDefaults(t *testing.T) {
	cfg := validConfig()
	cfg.Database.Password = "password"
	cfg.JWT.Secret = "secret"
	cfg.Turnstile.Secret = "secret"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() did not reject production defaults")
	}

	message := err.Error()
	for _, expected := range []string{"DB_PASSWORD", "JWT_SECRET", "TURNSTILE_SECRET"} {
		if !strings.Contains(message, expected) {
			t.Errorf("Validate() error %q does not mention %s", message, expected)
		}
	}
}

func TestConfigValidateRejectsInvalidPorts(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = "0"
	cfg.Database.Port = "not-a-port"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() did not reject invalid ports")
	}
	if !strings.Contains(err.Error(), "SERVER_PORT") || !strings.Contains(err.Error(), "DB_PORT") {
		t.Fatalf("Validate() error does not contain both invalid ports: %v", err)
	}
}

func TestDatabaseConnectionStringEscapesCredentials(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "db.example.com",
		Port:     "5432",
		Username: "blog@user",
		Password: "p@ss:/word",
		DBName:   "simple blog",
		SSLMode:  "require",
	}

	parsed, err := url.Parse(cfg.ConnectionString())
	if err != nil {
		t.Fatalf("Parse(ConnectionString()) returned an error: %v", err)
	}
	password, ok := parsed.User.Password()
	if !ok || parsed.User.Username() != cfg.Username || password != cfg.Password {
		t.Fatalf("credentials did not round-trip: %q", cfg.ConnectionString())
	}
	if parsed.Host != "db.example.com:5432" || parsed.Path != "/simple blog" {
		t.Fatalf("unexpected connection URL: %q", cfg.ConnectionString())
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("sslmode was not preserved: %q", cfg.ConnectionString())
	}
}

func validConfig() *Config {
	return &Config{
		App: AppConfig{Loglevel: "INFO"},
		Server: ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			Username: "postgres",
			Password: "a-secure-password",
			DBName:   "simpleblog",
			SSLMode:  "disable",
		},
		JWT:       JWTConfig{Secret: strings.Repeat("j", 32)},
		Turnstile: TurnstileConfig{Secret: "turnstile-secret"},
	}
}
