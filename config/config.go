package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	BasePath           string
	SSOBaseURL         string
	SSOLoginURL        string
	InternalAPIKey     string
	ChartsRequiredRole string
	CTPAPIURL          string
	CTPAPIKey          string
}

var C Config

func Load() {
	godotenv.Load()

	C = Config{
		Port:               getEnv("PORT", "3000"),
		BasePath:           getEnv("BASE_PATH", "/app"),
		SSOBaseURL:         getEnv("SSO_BASE_URL", "https://sso.example.com"),
		SSOLoginURL:        getEnv("SSO_LOGIN_URL", "https://sso.example.com/login"),
		InternalAPIKey:     getEnv("INTERNAL_API_KEY", ""),
		ChartsRequiredRole: getEnv("CHARTS_REQUIRED_ROLE", ""),
		CTPAPIURL:          getEnv("CTP_API_URL", "http://localhost:8080"),
		CTPAPIKey:          getEnv("CTP_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
