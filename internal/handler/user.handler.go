package handler

import (
	"net/http"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/internal/dto"
	"github.com/amirullazmi0/web-profile-backend/internal/model"
	"github.com/amirullazmi0/web-profile-backend/internal/usecase"
	"github.com/amirullazmi0/web-profile-backend/pkg/database"
	"github.com/amirullazmi0/web-profile-backend/pkg/logger"
	"github.com/amirullazmi0/web-profile-backend/pkg/response"
	"github.com/amirullazmi0/web-profile-backend/pkg/validator"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
	cookieCfg   *config.CookieConfig
	jwtCfg      *config.JWTConfig
	frontendURL string
}

// NewUserHandler creates a new user handler
func NewUserHandler(userUsecase usecase.UserUsecase, cookieCfg *config.CookieConfig, jwtCfg *config.JWTConfig, frontendURL string) *UserHandler {
	return &UserHandler{userUsecase: userUsecase, cookieCfg: cookieCfg, jwtCfg: jwtCfg, frontendURL: frontendURL}
}

func (h *UserHandler) setAuthCookies(c *gin.Context, result *dto.AuthResponse) {
	response.SetAuthCookies(c, h.cookieCfg, result.AccessToken, result.RefreshToken, int(result.ExpiresIn), h.jwtCfg.RefreshExpiredHour*3600)
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with email, password, and name
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register Request"
// @Success 201 {object} response.Response{data=dto.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	result, err := h.userUsecase.Register(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully. Please check your email to verify your account.", result)
}

// VerifyEmail godoc
// @Summary Verify user email
// @Description Verify user email with token from email
// @Tags auth
// @Produce html
// @Param token query string true "Verification Token"
// @Success 200 {string} html
// @Failure 400 {string} html
// @Router /api/auth/verify-email [get]
func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "verify-email.html", gin.H{
			"Success":      false,
			"ErrorMessage": "Verification token is required",
			"FrontendURL":  h.frontendURL,
		})
		return
	}

	err := h.userUsecase.VerifyEmail(token)
	if err != nil {
		c.HTML(http.StatusBadRequest, "verify-email.html", gin.H{
			"Success":      false,
			"ErrorMessage": err.Error(),
			"FrontendURL":  h.frontendURL,
		})
		return
	}

	c.HTML(http.StatusOK, "verify-email.html", gin.H{
		"Success":      true,
		"ErrorMessage": "",
		"FrontendURL":  h.frontendURL,
	})
}

// ResendVerification godoc
// @Summary Resend verification email
// @Description Resend a new verification email for accounts that are not active yet
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationRequest true "Resend Verification Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/auth/resend-verification [post]
func (h *UserHandler) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.ResendVerification(req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "If the account exists and is not verified, a new verification email has been sent.", nil)
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login Request"
// @Success 200 {object} response.Response{data=dto.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	result, err := h.userUsecase.Login(&req)
	if err != nil {
		if err.Error() == "please verify your email first" {
			response.Error(c, http.StatusForbidden, err.Error(), nil)
			return
		}
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Set authentication cookies
	h.setAuthCookies(c, result)

	response.Success(c, http.StatusOK, "Login successful", result)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send a password reset link to the user's email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/auth/forgot-password [post]
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.ForgotPassword(req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "If the account exists, a password reset email has been sent.", nil)
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset password using token sent to email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Reset Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/auth/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.ResetPassword(req.Token, req.NewPassword); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Password reset successfully", nil)
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user profile
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	result, err := h.userUsecase.GetProfile(userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Profile retrieved successfully", result)
}

// GetAllUsers godoc
// @Summary Get all users
// @Description Get list of all users
// GetAllUsers godoc
// @Summary Get all users
// @Description Get list of all users with search, sort, and pagination
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name or email"
// @Param sort query string false "Sort by field (e.g. name, email, created_at)"
// @Param order query string false "Order (asc, desc)"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} response.Response{data=[]dto.UserListResponse}
// @Failure 401 {object} response.Response
// @Router /api/users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	var q dto.QueryGlobal
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid query parameters", err.Error())
		return
	}

	users, total, err := h.userUsecase.GetAllUsers(&q)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get users", err.Error())
		return
	}

	meta := &response.Meta{
		TotalData:   total,
		TotalPage:   database.CalculateTotalPage(total, q.Limit),
		CurrentPage: q.Page,
		Limit:       q.Limit,
	}

	response.SuccessWithMeta(c, http.StatusOK, "Users retrieved successfully", users, meta)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update current user profile
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpdateUserRequest true "Update User Request"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	result, err := h.userUsecase.UpdateProfile(userID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", result)
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change current user password
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ChangePasswordRequest true "Change Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/change-password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.ChangePassword(userID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Password changed successfully", nil)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user by ID
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := h.userUsecase.DeleteUser(id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "User deleted successfully", nil)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh Token Request"
// @Success 200 {object} response.Response{data=dto.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/auth/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	// The body is optional: a browser client can rely on the httpOnly
	// refresh_token cookie instead.
	var req dto.RefreshTokenRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}
	}
	if req.RefreshToken == "" {
		if cookie, err := c.Cookie("refresh_token"); err == nil {
			req.RefreshToken = cookie
		}
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	result, err := h.userUsecase.RefreshToken(&req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Set authentication cookies
	h.setAuthCookies(c, result)

	response.Success(c, http.StatusOK, "Token refreshed successfully", result)
}

// Logout godoc
// @Summary Logout user
// @Description Logout and invalidate refresh token
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /api/auth/logout [post]
//
// Logout always succeeds. The route runs on optional auth so that an expired
// access token still gets the session torn down: refusing to end a session
// leaves the refresh token alive in the database and the auth cookies set in
// the browser, which is the opposite of what the caller asked for.
func (h *UserHandler) Logout(c *gin.Context) {
	// An expired or missing token means we cannot tell which refresh token to
	// revoke, but the cookies below still have to go.
	if userID := c.GetString("user_id"); userID != "" {
		if err := h.userUsecase.Logout(userID); err != nil {
			logger.Warn("failed to clear refresh token on logout", zap.String("user_id", userID), zap.Error(err))
		}
	}

	// Clear authentication cookies
	response.ClearAuthCookies(c, h.cookieCfg)

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// CreateSuperAdmin godoc
// @Summary Create a super admin user
// @Description Create a new super admin user with email, password, and name (superadmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.RegisterUserRequest true "Register Request"
// @Success 201 {object} response.Response{data=string}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/create-admin [post]
func (h *UserHandler) CreateByAdmin(c *gin.Context) {
	var req dto.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	result, err := h.userUsecase.CreateUserByAdmin(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	msg := ""
	if req.Role == model.RoleSuperAdmin {
		msg = "Super admin created successfully"
	} else if req.Role == model.RoleAdmin {
		msg = "Admin created successfully"
	} else {
		msg = "User created successfully"
	}

	response.Success(c, http.StatusCreated, msg, result)
}

// UpdateByAdmin godoc
// @Summary Update user by admin
// @Description Update user by admin (superadmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpdateUserByAdminRequest true "Update User Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/update-admin [put]
func (h *UserHandler) UpdateByAdmin(c *gin.Context) {
	var req dto.UpdateUserByAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.UpdateUserByAdmin(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "User updated successfully", nil)
}

// UpdatePasswordUser godoc
// @Summary Update user password by admin
// @Description Update password for a specific user by ID (superadmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body dto.UpdatePasswordUserRequest true "Update Password Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/update-password-admin/{id} [put]
func (h *UserHandler) UpdatePasswordUserByAdmin(c *gin.Context) {
	userID := c.Param("id")
	var req dto.UpdatePasswordUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.UpdatePasswordUserByAdmin(userID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Password updated successfully", nil)
}

// UpdateRoleUser godoc
// @Summary Update user role by admin
// @Description Update role for a specific user by ID (superadmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body dto.UpdateRoleUserRequest true "Update Role Request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/update-role-admin/{id} [put]
func (h *UserHandler) UpdateRoleUserByAdmin(c *gin.Context) {
	userID := c.Param("id")
	var req dto.UpdateRoleUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		response.ValidationError(c, validator.FormatValidationErrors(err))
		return
	}

	if err := h.userUsecase.UpdateRoleUserByAdmin(userID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Role updated successfully", nil)
}

// UpdateActiveUserByAdmin godoc
// @Summary Toggle user active status by admin
// @Description Toggle active status for a specific user by ID (superadmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/users/set-active-admin/{id} [put]
func (h *UserHandler) UpdateActiveUserByAdmin(c *gin.Context) {
	userID := c.Param("id")
	if err := h.userUsecase.UpdateActiveUserByAdmin(userID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "User updated successfully", nil)
}
