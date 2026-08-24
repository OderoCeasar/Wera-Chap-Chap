package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	FrontendOrigin      string
	DatabaseURL         string
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaShortcode      string
	MpesaPasskey        string
	S3Endpoint          string
	S3Bucket            string
}

func Load() Config {
	accessMinutes := getInt("ACCESS_TOKEN_MINUTES", 15)
	refreshHours := getInt("REFRESH_TOKEN_HOURS", 24*14)
	return Config{
		Port:                getEnv("PORT", "8080"),
		FrontendOrigin:      getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://wera:wera@localhost:5432/wera_chap_chap?sslmode=disable"),
		JWTSecret:           getEnv("JWT_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:      time.Duration(accessMinutes) * time.Minute,
		RefreshTokenTTL:     time.Duration(refreshHours) * time.Hour,
		MpesaConsumerKey:    os.Getenv("MPESA_CONSUMER_KEY"),
		MpesaConsumerSecret: os.Getenv("MPESA_CONSUMER_SECRET"),
		MpesaShortcode:      os.Getenv("MPESA_SHORTCODE"),
		MpesaPasskey:        os.Getenv("MPESA_PASSKEY"),
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
	}
}

func (c Config) Addr() string { return fmt.Sprintf(":%s", c.Port) }

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
