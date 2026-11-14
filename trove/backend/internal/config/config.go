package config

import "os"

type Config struct {
	Port        int
	DatabaseURL string
	RedisURL    string
	UploadsDir  string
	TLSCertFile string
	TLSKeyFile  string

	// IGDB (Twitch) client credentials for game lookups. Optional for the
	// server: without them, catalog search and linking return an error.
	IGDBClientID     string
	IGDBClientSecret string

	// Web push (VAPID) keys for sale notifications; generate with
	// `make vapid`. Without them, alerts can't be turned on.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// VAPIDSubject identifies the sender to push services: an https: URL or
	// an email address.
	VAPIDSubject string

	// AnthropicAPIKey enables device verdicts (Claude). SearxURL is the
	// SearXNG instance the worker searches for reviews of each version;
	// verdicts made while it's unreachable are redone once it's back.
	AnthropicAPIKey string
	SearxURL        string
}

func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 8080),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://lms:lms@localhost:5432/lms?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		UploadsDir:  getEnv("UPLOADS_DIR", "./uploads"),
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),

		IGDBClientID:     getEnv("IGDB_CLIENT_ID", ""),
		IGDBClientSecret: getEnv("IGDB_CLIENT_SECRET", ""),

		VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:    getEnv("VAPID_SUBJECT", "https://localhost"),

		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		SearxURL:        getEnv("SEARXNG_URL", "http://127.0.0.1:8888"),
	}
}

func (c *Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
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
