package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config stores application level configuration values.
type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	BudgetLimitMode    string
	Storage            StorageConfig
	OCR                OCRConfig
	CORS               CORSConfig
	MaxUploadSizeBytes int64
}

// StorageConfig describes object storage connectivity settings.
type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
}

// OCRConfig describes OCR adapter settings.
type OCRConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// CORSConfig describes HTTP cross-origin headers.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

// Load reads configuration from environment variables (with optional .env file).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		BudgetLimitMode:    strings.ToLower(getEnv("BUDGET_LIMIT_MODE", "soft")),
		MaxUploadSizeBytes: parseInt64Env("MAX_UPLOAD_SIZE_BYTES", 10*1024*1024),
	}

	fmt.Println(cfg)

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	var err error
	if cfg.AccessTokenTTL, err = parseDuration("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return nil, err
	}

	if cfg.RefreshTokenTTL, err = parseDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour); err != nil {
		return nil, err
	}

	if cfg.Storage, err = loadStorageConfig(); err != nil {
		return nil, err
	}

	if cfg.OCR, err = loadOCRConfig(); err != nil {
		return nil, err
	}

	if cfg.CORS, err = loadCORSConfig(); err != nil {
		return nil, err
	}

	if cfg.BudgetLimitMode != "soft" && cfg.BudgetLimitMode != "hard" {
		return nil, fmt.Errorf("BUDGET_LIMIT_MODE must be 'soft' or 'hard'")
	}

	return cfg, nil
}

func loadStorageConfig() (StorageConfig, error) {
	useSSL, err := parseBoolEnv("MINIO_USE_SSL", false)
	if err != nil {
		return StorageConfig{}, err
	}

	cfg := StorageConfig{
		Endpoint:  os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		Bucket:    getEnv("MINIO_BUCKET", "receipts"),
		UseSSL:    useSSL,
		Region:    getEnv("MINIO_REGION", ""),
	}

	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return StorageConfig{}, fmt.Errorf("MINIO_ENDPOINT, MINIO_ACCESS_KEY, and MINIO_SECRET_KEY are required")
	}

	return cfg, nil
}

func loadOCRConfig() (OCRConfig, error) {
	timeout, err := parseDuration("OCR_TIMEOUT", 15*time.Second)
	if err != nil {
		return OCRConfig{}, err
	}

	cfg := OCRConfig{
		BaseURL: getEnv("OCR_BASE_URL", ""),
		APIKey:  getEnv("OCR_API_KEY", ""),
		Timeout: timeout,
	}

	if cfg.BaseURL == "" {
		return OCRConfig{}, fmt.Errorf("OCR_BASE_URL is required")
	}

	return cfg, nil
}

func loadCORSConfig() (CORSConfig, error) {
	allowCredentials, err := parseBoolEnv("CORS_ALLOW_CREDENTIALS", false)
	if err != nil {
		return CORSConfig{}, err
	}

	rawOrigins := getEnv("CORS_ALLOWED_ORIGINS", "*")
	origins := splitAndTrim(rawOrigins)
	if len(origins) == 0 {
		return CORSConfig{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}

	if allowCredentials {
		for _, origin := range origins {
			if origin == "*" {
				return CORSConfig{}, fmt.Errorf("CORS_ALLOW_CREDENTIALS cannot be true when CORS_ALLOWED_ORIGINS allows '*'")
			}
		}
	}

	return CORSConfig{
		AllowedOrigins:   origins,
		AllowCredentials: allowCredentials,
	}, nil
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}

	return duration, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	switch strings.ToLower(val) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean for %s", key)
	}
}

func parseInt64Env(key string, fallback int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	res := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			res = append(res, part)
		}
	}
	return res
}
