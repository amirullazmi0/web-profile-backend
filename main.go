package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirullazmi0/web-profile-backend/config"
	_ "github.com/amirullazmi0/web-profile-backend/docs"
	"github.com/amirullazmi0/web-profile-backend/internal/handler"
	"github.com/amirullazmi0/web-profile-backend/pkg/database"
	"github.com/amirullazmi0/web-profile-backend/pkg/database/seeder"
	"github.com/amirullazmi0/web-profile-backend/pkg/logger"
	redisclient "github.com/amirullazmi0/web-profile-backend/pkg/redis"
	"github.com/amirullazmi0/web-profile-backend/pkg/validator"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
)

// @title Web Profile API
// @version 1.0
// @description Backend API for Web Profile with JWT authentication and email verification

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Parse flags
	seed := flag.Bool("seed", false, "Seed the database")
	healthcheck := flag.Bool("healthcheck", false, "Probe the local /health endpoint and exit (used by the Docker HEALTHCHECK)")
	flag.Parse()

	if *healthcheck {
		os.Exit(runHealthcheck())
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	loggerCfg := logger.LoggerConfig{
		Debug:       cfg.App.Debug,
		Format:      cfg.Logger.Format,
		LogToFile:   cfg.Logger.LogToFile,
		LogFilePath: cfg.Logger.LogFilePath,
		MaxSize:     cfg.Logger.MaxSize,
		MaxBackups:  cfg.Logger.MaxBackups,
		MaxAge:      cfg.Logger.MaxAge,
		Compress:    cfg.Logger.Compress,
	}
	if err := logger.InitLogger(loggerCfg); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting application", zap.String("name", cfg.App.Name), zap.String("env", cfg.App.Env))

	// Initialize validator
	validator.InitValidator()

	// Initialize database
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run seeder if -seed flag is provided
	if *seed {
		logger.Info("Running database seeder...")
		if err := seeder.SeedSuperAdmin(db.DB, &cfg.Seed); err != nil {
			logger.Fatal("Failed to seed database", zap.Error(err))
		}
		logger.Info("Database seeding completed successfully")
		os.Exit(0)
	}

	// Initialize Redis after the optional database-only seed path.
	redis, err := redisclient.NewClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to redis", zap.Error(err))
	}
	defer func() {
		if err := redis.Close(); err != nil {
			logger.Warn("Failed to close redis client", zap.Error(err))
		}
	}()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()
	router, err := handler.Router(appCtx, cfg, db.DB, redis)
	if err != nil {
		logger.Fatal("Failed to setup router", zap.Error(err))
	}

	// Setup HTTP server
	srv := &http.Server{
		Addr:           ":" + cfg.App.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server is running", zap.String("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	appCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// runHealthcheck checks liveness only (it does not ping the database), so a
// brief database outage does not make Swarm restart the container in a loop.
func runHealthcheck() int {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
