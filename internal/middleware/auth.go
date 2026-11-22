package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Token types carried in the "typ" claim, so a refresh token can never be
// used as an access token and the other way round.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

// JWTAuth middleware validates JWT token
func JWTAuth(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from cookie first
		tokenString, err := c.Cookie("access_token")
		if err != nil || tokenString == "" {
			tokenString, err = c.Cookie("accessToken")
		}

		// If not found in cookie, check Authorization header
		if err != nil || tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				response.Error(c, http.StatusUnauthorized, "Authorization required", nil)
				c.Abort()
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(c, http.StatusUnauthorized, "Invalid authorization header format", nil)
				c.Abort()
				return
			}

			tokenString = parts[1]
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})

		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, "Invalid or expired token", nil)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || claims.Type != TokenTypeAccess {
			response.Error(c, http.StatusUnauthorized, "Invalid token claims", nil)
			c.Abort()
			return
		}

		// Set user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// JWTAuthOptional middleware optionally validates JWT token
func JWTAuthOptional(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from cookie first
		cookieToken, err := c.Cookie("access_token")
		if err != nil || cookieToken == "" {
			cookieToken, err = c.Cookie("accessToken")
		}
		if err == nil && cookieToken != "" {
			tokenString = cookieToken
		}

		// If not found in cookie, check Authorization header
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				// Extract token from "Bearer <token>"
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}
		}

		// If no token found, just continue to next handler
		if tokenString == "" {
			c.Next()
			return
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})

		// If token is valid, set user info to context
		if err == nil && token.Valid {
			if claims, ok := token.Claims.(*Claims); ok && claims.Type == TokenTypeAccess {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_role", claims.Role)
			}
		}

		c.Next()
	}
}

// GenerateToken generates a new JWT token
func GenerateToken(userID string, email string, role string, cfg *config.JWTConfig) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.ExpiredHour) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// GenerateRefreshToken generates a refresh token (longer expiry)
func GenerateRefreshToken(userID string, email string, role string, cfg *config.JWTConfig) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.RefreshExpiredHour) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}
