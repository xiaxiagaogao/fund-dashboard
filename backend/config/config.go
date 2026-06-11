// Package config loads the dashboard's runtime config from environment
// variables (with optional .env file support for local dev).
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// Required for any operation
	FundDBPath string // e.g. ./data/fund.db (local) or /data/fund.db (in container)

	// Required for live Binance reads (NAV snapshots, positions, trade fills sync)
	BinanceAPIKey    string
	BinanceAPISecret string

	// Required for HTTP server
	JWTSecret string
	HTTPAddr  string // default :8090
}

// Load reads env vars, falling back to a .env file in the current directory
// (or path passed via DASHBOARD_ENV_FILE).
func Load() (*Config, error) {
	loadDotenv()

	c := &Config{
		FundDBPath:       envOr("FUND_DB_PATH", "./data/fund.db"),
		BinanceAPIKey:    os.Getenv("BINANCE_API_KEY"),
		BinanceAPISecret: os.Getenv("BINANCE_API_SECRET"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		HTTPAddr:         envOr("HTTP_ADDR", ":8090"),
	}
	return c, nil
}

func RequireBinance(c *Config) error {
	if c.BinanceAPIKey == "" || c.BinanceAPISecret == "" {
		return fmt.Errorf("BINANCE_API_KEY and BINANCE_API_SECRET must be set (in .env or environment)")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotenv reads a simple KEY=VALUE .env file from cwd (or DASHBOARD_ENV_FILE).
// Lines starting with # are comments. Values may be quoted with " or '.
// Existing env vars take precedence (so docker-compose env: still wins).
func loadDotenv() {
	path := os.Getenv("DASHBOARD_ENV_FILE")
	if path == "" {
		path = ".env"
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
}
