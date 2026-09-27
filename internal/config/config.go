package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	HTTPAddr        = "HTTP_ADDR"
	LogLevel        = "LOG_LEVEL"
	ShutdownTimeout = "SHUTDOWN_TIMEOUT"

	DatabaseURL             = "DATABASE_URL"
	DatabaseMaxConns        = "DATABASE_MAX_CONNS"
	DatabaseMinConns        = "DATABASE_MIN_CONNS"
	DatabaseMaxConnLifetime = "DATABASE_MAX_CONN_LIFETIME"
	DatabaseConnectTimeout  = "DATABASE_CONNECT_TIMEOUT"
	DatabaseQueryTimeout    = "DATABASE_QUERY_TIMEOUT"

	HTTPReadTimeout       = "HTTP_READ_TIMEOUT"
	HTTPReadHeaderTimeout = "HTTP_READ_HEADER_TIMEOUT"
	HTTPWriteTimeout      = "HTTP_WRITE_TIMEOUT"
	HTTPIdleTimeout       = "HTTP_IDLE_TIMEOUT"
)

type Config struct {
	HTTPAddr        string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration

	Database DatabaseConfig

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

func Load() (Config, error) {
	var cfg Config
	db := &cfg.Database
	var err error

	if cfg.HTTPAddr, err = required(HTTPAddr); err != nil {
		return Config{}, err
	}

	rawLevel, err := required(LogLevel)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(rawLevel)); err != nil {
		return Config{}, fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	if cfg.ShutdownTimeout, err = positiveDuration(ShutdownTimeout); err != nil {
		return Config{}, err
	}

	if cfg.ReadTimeout, err = positiveDuration(HTTPReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ReadHeaderTimeout, err = positiveDuration(HTTPReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = positiveDuration(HTTPWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.IdleTimeout, err = positiveDuration(HTTPIdleTimeout); err != nil {
		return Config{}, err
	}

	if db.URL, err = required(DatabaseURL); err != nil {
		return Config{}, err
	}
	if db.MaxConns, err = int32Value(DatabaseMaxConns); err != nil {
		return Config{}, err
	}
	if db.MinConns, err = int32Value(DatabaseMinConns); err != nil {
		return Config{}, err
	}
	if db.MaxConns < 1 || db.MinConns < 0 || db.MinConns > db.MaxConns {
		return Config{}, fmt.Errorf("invalid database pool sizes")
	}

	if db.MaxConnLifetime, err = positiveDuration(DatabaseMaxConnLifetime); err != nil {
		return Config{}, err
	}
	if db.ConnectTimeout, err = positiveDuration(DatabaseConnectTimeout); err != nil {
		return Config{}, err
	}
	if db.QueryTimeout, err = positiveDuration(DatabaseQueryTimeout); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func required(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func positiveDuration(name string) (time.Duration, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return duration, nil
}

func int32Value(name string) (int32, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return int32(number), nil
}
