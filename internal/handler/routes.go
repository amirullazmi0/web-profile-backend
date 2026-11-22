package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/internal/middleware"
	"github.com/amirullazmi0/web-profile-backend/internal/repository"
	"github.com/amirullazmi0/web-profile-backend/internal/usecase"
	"github.com/amirullazmi0/web-profile-backend/pkg/email"
	redisclient "github.com/amirullazmi0/web-profile-backend/pkg/redis"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router configures Gin, middlewares, handlers, and all routes.
func Router(ctx context.Context, cfg *config.Config, db *sql.DB, redis *redisclient.Client) (*gin.Engine, error) {
	// Initialize email service
	emailService := email.NewEmailService(&cfg.SMTP, cfg.App.Name)

	// Initialize usecases
	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, &cfg.JWT, emailService, &cfg.App, redis)
	userHandler := NewUserHandler(userUsecase, &cfg.Cookie, &cfg.JWT, cfg.App.FrontendURL)

	// Setup Gin
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middlewares
	router.Use(middleware.Recovery())
	router.Use(requestid.New())
	router.Use(middleware.Logger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Load HTML templates
	router.LoadHTMLFiles("templates/verify-email.html")

	// API routes
	api := router.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:register",
				Limit:     5,
				Window:    time.Minute,
			}), userHandler.Register)
			auth.GET("/verify-email", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:verify-email",
				Limit:     10,
				Window:    time.Minute,
			}), userHandler.VerifyEmail)
			auth.POST("/resend-verification", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:resend-verification",
				Limit:     5,
				Window:    time.Minute,
			}), userHandler.ResendVerification)
			auth.POST("/login", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:login",
				Limit:     10,
				Window:    time.Minute,
			}), userHandler.Login)
			auth.POST("/refresh", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:refresh",
				Limit:     20,
				Window:    time.Minute,
			}), userHandler.RefreshToken)
			auth.POST("/forgot-password", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:forgot-password",
				Limit:     5,
				Window:    time.Minute,
			}), userHandler.ForgotPassword)
			auth.POST("/reset-password", middleware.RateLimit(redis, middleware.RateLimitConfig{
				KeyPrefix: "auth:reset-password",
				Limit:     10,
				Window:    time.Minute,
			}), userHandler.ResetPassword)
			auth.POST("/logout", middleware.JWTAuthOptional(&cfg.JWT), userHandler.Logout)
		}

		// User routes (protected)
		users := api.Group("/users")
		users.Use(middleware.JWTAuth(&cfg.JWT))
		{
			users.GET("/profile", userHandler.GetProfile)
			users.PUT("/profile", userHandler.UpdateProfile)
			users.PUT("/change-password", userHandler.ChangePassword)

			// Superadmin only routes
			users.GET("", middleware.RequireSuperAdmin(), userHandler.GetAllUsers)
			users.DELETE("/:id", middleware.RequireSuperAdmin(), userHandler.DeleteUser)
			users.POST("/create-admin", middleware.RequireSuperAdmin(), userHandler.CreateByAdmin)
			users.PUT("/update-admin", middleware.RequireSuperAdmin(), userHandler.UpdateByAdmin)
			users.PUT("/update-password-admin/:id", middleware.RequireSuperAdmin(), userHandler.UpdatePasswordUserByAdmin)
			users.PUT("/update-role-admin/:id", middleware.RequireSuperAdmin(), userHandler.UpdateRoleUserByAdmin)
			users.PUT("/set-active-admin/:id", middleware.RequireSuperAdmin(), userHandler.UpdateActiveUserByAdmin)
		}

		// Register new feature routes here, following the users group above.
	}

	return router, nil
}
