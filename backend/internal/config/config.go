package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                int
	DatabaseURL         string
	OpenClawToken       string
	TelegramBotToken    string
	TelegramChatID      int64
	FetchCron           string
	FetchBatchSize      int
	FetchTimeout        time.Duration
	DigestCron          string
	TrustedProxies      string
	LogLevel            string
	TZ                  string
	UserAgent           string
	CriticalThrottle    time.Duration
	FeedDeactivateAfter int
	RetentionDays       int
	RetentionCron       string
	Version             string
}

func Load() (*Config, error) {
	c := &Config{
		Port:                getInt("PORT", 3000),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		OpenClawToken:       os.Getenv("OPENCLAW_GATEWAY_TOKEN"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:      getInt64("TELEGRAM_CHAT_ID", 0),
		FetchCron:           getStr("FETCH_CRON", "*/15 * * * *"),
		FetchBatchSize:      getInt("FETCH_BATCH_SIZE", 10),
		FetchTimeout:        time.Duration(getInt("FETCH_TIMEOUT_SECONDS", 20)) * time.Second,
		DigestCron:          getStr("DIGEST_CRON", "0 8 * * *"),
		TrustedProxies:      getStr("TRUSTED_PROXIES", "0.0.0.0/0"),
		LogLevel:            getStr("LOG_LEVEL", "info"),
		TZ:                  getStr("TZ", "UTC"),
		UserAgent:           getStr("USER_AGENT", "rss-fresh/1.0 (+https://github.com/mustafaeeroglu/rss-fresh)"),
		CriticalThrottle:    time.Duration(getInt("CRITICAL_THROTTLE_SECONDS", 5)) * time.Second,
		FeedDeactivateAfter: getInt("FEED_DEACTIVATE_AFTER_ERRORS", 10),
		RetentionDays:       getInt("RETENTION_DAYS", 30),
		RetentionCron:       getStr("RETENTION_CRON", "0 4 * * *"),
		Version:             getStr("APP_VERSION", "dev"),
	}

	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if c.OpenClawToken == "" {
		return nil, errors.New("OPENCLAW_GATEWAY_TOKEN is required")
	}
	return c, nil
}

func getStr(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
		fmt.Fprintf(os.Stderr, "warn: %s=%q is not an int, using default %d\n", k, v, def)
	}
	return def
}

func getInt64(k string, def int64) int64 {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
		fmt.Fprintf(os.Stderr, "warn: %s=%q is not an int64, using default %d\n", k, v, def)
	}
	return def
}
