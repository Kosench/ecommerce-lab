package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
}

type ServerConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Load() (*Config, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = ":8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return &Config{
		Environment: env,
		Server: ServerConfig{
			Addr:            serverAddr,
			ShutdownTimeout: 10 * time.Second,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		},
	}, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
