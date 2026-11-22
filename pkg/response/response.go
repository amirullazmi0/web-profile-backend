package response

import (
	"github.com/amirullazmi0/web-profile-backend/config"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type Meta struct {
	TotalData   int `json:"total_data"`
	TotalPage   int `json:"total_page"`
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
}

// Success sends a success response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a success response with metadata
func SuccessWithMeta(c *gin.Context, statusCode int, message string, data interface{}, meta *Meta) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error response
func Error(c *gin.Context, statusCode int, message string, err interface{}) {
	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Error:   err,
	})
}

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, errors interface{}) {
	c.JSON(400, Response{
		Success: false,
		Message: "Validation failed",
		Error:   errors,
	})
}

// SetAuthCookies sets access token and refresh token cookies as httpOnly.
// Secure, SameSite and Domain come from COOKIE_* env vars, so production
// (HTTPS, cross-site frontend) and local development can differ.
func SetAuthCookies(c *gin.Context, cfg *config.CookieConfig, accessToken, refreshToken string, accessMaxAge, refreshMaxAge int) {
	c.SetSameSite(cfg.SameSite)
	c.SetCookie("access_token", accessToken, accessMaxAge, "/", cfg.Domain, cfg.Secure, true)
	c.SetCookie("refresh_token", refreshToken, refreshMaxAge, "/", cfg.Domain, cfg.Secure, true)
}

// ClearAuthCookies clears authentication cookies
func ClearAuthCookies(c *gin.Context, cfg *config.CookieConfig) {
	c.SetSameSite(cfg.SameSite)
	c.SetCookie("access_token", "", -1, "/", cfg.Domain, cfg.Secure, true)
	c.SetCookie("refresh_token", "", -1, "/", cfg.Domain, cfg.Secure, true)
}
