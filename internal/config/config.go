package config

import "os"

type Config struct {
	DBURL     string
	HTTPAddr  string
	JWTSecret string
}

func Load() Config {
	return Config{
		DBURL:     env("DB_URL", "postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"),
		HTTPAddr:  env("HTTP_ADDR", ":8081"),
		JWTSecret: env("JWT_SECRET", "tree-time-secret"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
