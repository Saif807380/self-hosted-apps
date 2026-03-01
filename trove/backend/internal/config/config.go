package config

import "os"

type Config struct {
	Port        int
	DatabaseURL string
	RedisURL    string
	UploadsDir  string
}

func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 8080),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://lms:lms@localhost:5432/lms?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		UploadsDir:  getEnv("UPLOADS_DIR", "./uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var result int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		result = result*10 + int(c-'0')
	}
	return result
}
