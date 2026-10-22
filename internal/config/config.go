package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

type Config struct {
	DatabaseURL string
	Address     string
	MediaDir    string
	PublicURL   string
}

func Load() (Config, error) {
	return Read(os.Getenv)
}
func Read(get func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if v := get(key); v != "" {
			return v
		}
		return fallback
	}
	c := Config{
		DatabaseURL: get("DATABASE_URL"), Address: value("KURA_ADDR", "127.0.0.1:8094"),
		MediaDir: value("KURA_MEDIA_DIR", ".local/media"), PublicURL: value("KURA_PUBLIC_URL", "http://localhost:5174"),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required; copy .env.example to .env for local development")
	}
	databaseURL, err := url.Parse(c.DatabaseURL)
	if err != nil || databaseURL.Host == "" || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") {
		return Config{}, fmt.Errorf("DATABASE_URL must be a PostgreSQL connection URL")
	}
	publicURL, err := url.Parse(c.PublicURL)
	if err != nil || publicURL.Host == "" || publicURL.User != nil || publicURL.RawQuery != "" || publicURL.Fragment != "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return Config{}, fmt.Errorf("KURA_PUBLIC_URL must be an absolute http(s) URL without credentials, query or fragment")
	}
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return Config{}, fmt.Errorf("KURA_ADDR must have host:port format")
	}
	return c, nil
}
