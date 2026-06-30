package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type appConfig struct {
	Port               string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	GuestSessionTTL    time.Duration
	InterestProfileTTL time.Duration
	CookieSecure       bool
	CookieDomain       string
	IPHashSecret       string
	FrontendOrigins    []string
	BodyLimitBytes     int64
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
}

func loadAppConfig() appConfig {
	return appConfig{
		Port:               env("PORT", "8080"),
		JWTSecret:          env("JWT_SECRET", ""),
		AccessTokenTTL:     envDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:    envDuration("REFRESH_TOKEN_TTL", 14*24*time.Hour),
		GuestSessionTTL:    envDuration("GUEST_SESSION_TTL", 7*24*time.Hour),
		InterestProfileTTL: envDuration("INTEREST_PROFILE_TTL", 30*24*time.Hour),
		CookieSecure:       envBool("COOKIE_SECURE", false),
		CookieDomain:       env("COOKIE_DOMAIN", ""),
		IPHashSecret:       env("IP_HASH_SECRET", env("JWT_SECRET", "")),
		FrontendOrigins:    envList("FE_URL", []string{"http://localhost:3000"}),
		BodyLimitBytes:     envInt64("BODY_LIMIT_BYTES", 1<<20),
		RedisAddr:          env("REDIS_HOST", "localhost") + ":" + env("REDIS_PORT", "6379"),
		RedisPassword:      env("REDIS_PASSWORD", ""),
		RedisDB:            envInt("REDIS_DB", 0),
	}
}

func newRedisClient(cfg appConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			values = append(values, item)
		}
	}

	if len(values) == 0 {
		return fallback
	}

	return values
}
