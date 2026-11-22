package config

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Cookie   CookieConfig
	CORS     CORSConfig
	SMTP     SMTPConfig
	Logger   LoggerConfig
	Seed     SeedConfig
}

type AppConfig struct {
	Name        string
	Env         string
	Port        string
	FrontendURL string
	// BackendURL is the public base URL of this API including the /api prefix.
	// Links that point back to the API, such as email verification, use it.
	BackendURL string
	Debug      bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret             string
	ExpiredHour        int
	RefreshExpiredHour int
}

type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type SMTPConfig struct {
	Email     string
	Password  string
	Host      string
	Port      int
	FromName  string
	FromEmail string
}

type LoggerConfig struct {
	// Format is "console" (human readable, coloured on a terminal), "json"
	// (Loki/Promtail friendly), or empty to auto-detect.
	Format      string
	LogToFile   bool
	LogFilePath string
	MaxSize     int
	MaxBackups  int
	MaxAge      int
	Compress    bool
}

type SeedConfig struct {
	SuperAdminName     string
	SuperAdminEmail    string
	SuperAdminPassword string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if _, err := os.Stat(".env"); err == nil {
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("JWT_EXPIRED_HOUR", 24)
	viper.SetDefault("JWT_REFRESH_EXPIRED_HOUR", 168)
	viper.SetDefault("COOKIE_SAME_SITE", "lax")
	viper.SetDefault("LOG_FILE_PATH", "logs/app.log")
	viper.SetDefault("SEED_SUPERADMIN_NAME", "Super Admin")

	config := &Config{
		App: AppConfig{
			Name:        viper.GetString("APP_NAME"),
			Env:         viper.GetString("APP_ENV"),
			Port:        viper.GetString("APP_PORT"),
			FrontendURL: strings.TrimRight(viper.GetString("APP_FRONTEND_URL"), "/"),
			BackendURL:  strings.TrimRight(viper.GetString("APP_BACKEND_URL"), "/"),
			Debug:       viper.GetBool("APP_DEBUG"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSL_MODE"),
			TimeZone: viper.GetString("DB_TIMEZONE"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:             viper.GetString("JWT_SECRET"),
			ExpiredHour:        viper.GetInt("JWT_EXPIRED_HOUR"),
			RefreshExpiredHour: viper.GetInt("JWT_REFRESH_EXPIRED_HOUR"),
		},
		Cookie: CookieConfig{
			Secure:   viper.GetBool("COOKIE_SECURE"),
			SameSite: parseSameSite(viper.GetString("COOKIE_SAME_SITE")),
			Domain:   strings.TrimSpace(viper.GetString("COOKIE_DOMAIN")),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitAndTrim(viper.GetString("CORS_ALLOWED_ORIGINS")),
		},
		SMTP: SMTPConfig{
			Email:     viper.GetString("SMTP_EMAIL"),
			Password:  viper.GetString("SMTP_PASSWORD"),
			Host:      viper.GetString("SMTP_HOST"),
			Port:      viper.GetInt("SMTP_PORT"),
			FromName:  viper.GetString("SMTP_FROM_NAME"),
			FromEmail: viper.GetString("SMTP_FROM_EMAIL"),
		},
		Logger: LoggerConfig{
			Format:      strings.TrimSpace(viper.GetString("LOG_FORMAT")),
			LogToFile:   viper.GetBool("LOG_TO_FILE"),
			LogFilePath: viper.GetString("LOG_FILE_PATH"),
			MaxSize:     viper.GetInt("LOG_MAX_SIZE"),
			MaxBackups:  viper.GetInt("LOG_MAX_BACKUPS"),
			MaxAge:      viper.GetInt("LOG_MAX_AGE"),
			Compress:    viper.GetBool("LOG_COMPRESS"),
		},
		Seed: SeedConfig{
			SuperAdminName:     viper.GetString("SEED_SUPERADMIN_NAME"),
			SuperAdminEmail:    strings.TrimSpace(viper.GetString("SEED_SUPERADMIN_EMAIL")),
			SuperAdminPassword: viper.GetString("SEED_SUPERADMIN_PASSWORD"),
		},
	}

	if config.App.BackendURL == "" {
		config.App.BackendURL = "http://localhost:" + config.App.Port + "/api"
	}

	return config, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode, c.TimeZone,
	)
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
