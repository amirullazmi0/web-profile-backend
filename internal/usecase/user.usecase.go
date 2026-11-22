package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/internal/dto"
	"github.com/amirullazmi0/web-profile-backend/internal/middleware"
	"github.com/amirullazmi0/web-profile-backend/internal/model"
	"github.com/amirullazmi0/web-profile-backend/internal/repository"
	"github.com/amirullazmi0/web-profile-backend/pkg/email"
	redisclient "github.com/amirullazmi0/web-profile-backend/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
)

type UserUsecase interface {
	Register(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	VerifyEmail(token string) error
	ResendVerification(email string) error
	Login(req *dto.LoginRequest) (*dto.AuthResponse, error)
	ForgotPassword(email string) error
	ResetPassword(token string, newPassword string) error
	RefreshToken(req *dto.RefreshTokenRequest) (*dto.AuthResponse, error)
	Logout(userID string) error
	GetProfile(userID string) (*dto.UserResponse, error)
	GetAllUsers(q *dto.QueryGlobal) ([]dto.UserListResponse, int, error)
	UpdateProfile(userID string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	ChangePassword(userID string, req *dto.ChangePasswordRequest) error
	DeleteUser(userID string) error
	CreateUserByAdmin(req *dto.RegisterUserRequest) (string, error)
	UpdateUserByAdmin(req *dto.UpdateUserByAdminRequest) error
	UpdatePasswordUserByAdmin(id string, req *dto.UpdatePasswordUserRequest) error
	UpdateRoleUserByAdmin(id string, req *dto.UpdateRoleUserRequest) error
	UpdateActiveUserByAdmin(id string) error
}
type userUsecase struct {
	userRepo     repository.UserRepository
	jwtCfg       *config.JWTConfig
	emailService *email.EmailService
	appConfig    *config.AppConfig
	redisClient  *redisclient.Client
}

// NewUserUsecase creates a new user usecase
func NewUserUsecase(userRepo repository.UserRepository, jwtCfg *config.JWTConfig, emailService *email.EmailService, appConfig *config.AppConfig, redisClient *redisclient.Client) UserUsecase {
	return &userUsecase{
		userRepo:     userRepo,
		jwtCfg:       jwtCfg,
		emailService: emailService,
		appConfig:    appConfig,
		redisClient:  redisClient,
	}
}

func (u *userUsecase) verificationTokenKey(token string) string {
	return fmt.Sprintf("verification:email:token:%s", token)
}

func (u *userUsecase) verificationUserKey(userID string) string {
	return fmt.Sprintf("verification:email:user:%s", userID)
}

func (u *userUsecase) resetPasswordTokenKey(token string) string {
	return fmt.Sprintf("reset:password:token:%s", token)
}

func (u *userUsecase) resetPasswordUserKey(userID string) string {
	return fmt.Sprintf("reset:password:user:%s", userID)
}

func (u *userUsecase) backendBaseURL() string {
	return u.appConfig.BackendURL
}

func (u *userUsecase) frontendBaseURL() string {
	if strings.TrimSpace(u.appConfig.FrontendURL) != "" {
		return strings.TrimRight(u.appConfig.FrontendURL, "/")
	}

	return "http://localhost:3000"
}

func (u *userUsecase) Register(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Check if user already exists
	existingUser, _ := u.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// Create new user
	user := &model.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     model.RoleUser,
	}

	// Hash password
	if err := user.HashPassword(); err != nil {
		return nil, err
	}

	// Save to database
	userID, err := u.userRepo.Create(user)
	if err != nil {
		return nil, err
	}
	user.ID = userID

	// Generate verification token
	verificationToken, err := email.GenerateVerificationToken()
	if err != nil {
		return nil, err
	}

	// Save verification token in Redis (expires in 24 hours)
	if err := u.saveVerificationToken(user.ID, verificationToken); err != nil {
		return nil, err
	}

	// Send verification email in background (goroutine)
	go func() {
		if err := u.emailService.SendVerificationEmail(user.Email, user.Name, verificationToken, u.backendBaseURL()); err != nil {
			log.Printf("Failed to send verification email to %s: %v", user.Email, err)
		} else {
			log.Printf("Verification email sent successfully to %s", user.Email)
		}
	}()

	// Don't generate tokens yet - user needs to verify email first
	return &dto.AuthResponse{
		AccessToken:  "",
		RefreshToken: "",
		ExpiresIn:    0,
		User: dto.UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			IsActive: user.IsActive,
		},
	}, nil
}

func (u *userUsecase) CreateUserByAdmin(req *dto.RegisterUserRequest) (string, error) {
	existingUser, _ := u.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return "", errors.New("email already registered")
	}

	// Create new user
	newUser := &model.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     req.Role,
	}

	// Hash password
	if err := newUser.HashPassword(); err != nil {
		return "", err
	}

	// Save to database
	userID, err := u.userRepo.CreateUserByAdmin(newUser)
	if err != nil {
		return "", err
	}
	newUser.ID = userID

	// Generate verification token
	verificationToken, err := email.GenerateVerificationToken()
	if err != nil {
		return "", err
	}

	// Save verification token in Redis (expires in 24 hours)
	if err := u.saveVerificationToken(newUser.ID, verificationToken); err != nil {
		return "", err
	}

	// Send verification email in background (goroutine)
	go func() {
		if err := u.emailService.SendVerificationEmailAdmin(newUser.Email, newUser.Name, verificationToken, req.Password, u.backendBaseURL()); err != nil {
			log.Printf("Failed to send verification email to %s: %v", newUser.Email, err)
		} else {
			log.Printf("Verification email sent successfully to %s", newUser.Email)
		}
	}()

	return userID, nil
}

func (u *userUsecase) UpdateUserByAdmin(req *dto.UpdateUserByAdminRequest) error {
	existingUser, err := u.userRepo.FindByID(req.ID)
	if err != nil {
		return err
	}

	if req.Name != "" {
		existingUser.Name = req.Name
	}
	if req.Avatar != nil {
		existingUser.Avatar = req.Avatar
	}
	if req.Role != "" {
		existingUser.Role = req.Role
	}

	// Save to database
	if err := u.userRepo.Update(existingUser.ID, existingUser); err != nil {
		return err
	}

	return nil
}

func (u *userUsecase) VerifyEmail(token string) error {
	userID, err := u.findVerificationToken(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, goredis.Nil) {
			return errors.New("invalid or expired verification token")
		}
		return err
	}

	// Verify email
	if err := u.userRepo.VerifyEmail(userID); err != nil {
		return err
	}

	// Remove verification token from Redis after successful verification.
	if err := u.deleteVerificationTokenPair(token, userID); err != nil {
		log.Printf("Failed to delete verification token %s from redis: %v", token, err)
	}

	return nil
}

func (u *userUsecase) ResendVerification(emailAddress string) error {
	user, err := u.userRepo.FindByEmail(emailAddress)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if user.IsActive {
		return nil
	}

	verificationToken, err := email.GenerateVerificationToken()
	if err != nil {
		return err
	}

	if err := u.saveVerificationToken(user.ID, verificationToken); err != nil {
		return err
	}

	go func() {
		if err := u.emailService.SendVerificationEmail(user.Email, user.Name, verificationToken, u.backendBaseURL()); err != nil {
			log.Printf("Failed to resend verification email to %s: %v", user.Email, err)
		} else {
			log.Printf("Verification email resent successfully to %s", user.Email)
		}
	}()

	return nil
}

func (u *userUsecase) saveVerificationToken(userID, token string) error {
	if u.redisClient == nil {
		return errors.New("verification token service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if oldToken, err := u.redisClient.Get(ctx, u.verificationUserKey(userID)); err == nil && oldToken != "" {
		_ = u.redisClient.Del(ctx, u.verificationTokenKey(oldToken))
	}

	if err := u.redisClient.Set(ctx, u.verificationTokenKey(token), userID, 24*time.Hour); err != nil {
		return err
	}

	return u.redisClient.Set(ctx, u.verificationUserKey(userID), token, 24*time.Hour)
}

func (u *userUsecase) findVerificationToken(token string) (string, error) {
	if u.redisClient == nil {
		return "", errors.New("verification token service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return u.redisClient.Get(ctx, u.verificationTokenKey(token))
}

func (u *userUsecase) deleteVerificationTokenPair(token, userID string) error {
	if u.redisClient == nil {
		return errors.New("verification token service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := u.redisClient.Del(ctx, u.verificationTokenKey(token)); err != nil {
		return err
	}

	return u.redisClient.Del(ctx, u.verificationUserKey(userID))
}

func (u *userUsecase) saveResetPasswordToken(userID, token string) error {
	if u.redisClient == nil {
		return errors.New("reset password service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if oldToken, err := u.redisClient.Get(ctx, u.resetPasswordUserKey(userID)); err == nil && oldToken != "" {
		_ = u.redisClient.Del(ctx, u.resetPasswordTokenKey(oldToken))
	}

	if err := u.redisClient.Set(ctx, u.resetPasswordTokenKey(token), userID, time.Hour); err != nil {
		return err
	}

	return u.redisClient.Set(ctx, u.resetPasswordUserKey(userID), token, time.Hour)
}

func (u *userUsecase) findResetPasswordToken(token string) (string, error) {
	if u.redisClient == nil {
		return "", errors.New("reset password service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return u.redisClient.Get(ctx, u.resetPasswordTokenKey(token))
}

func (u *userUsecase) deleteResetPasswordTokenPair(token, userID string) error {
	if u.redisClient == nil {
		return errors.New("reset password service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := u.redisClient.Del(ctx, u.resetPasswordTokenKey(token)); err != nil {
		return err
	}

	return u.redisClient.Del(ctx, u.resetPasswordUserKey(userID))
}

func (u *userUsecase) Login(req *dto.LoginRequest) (*dto.AuthResponse, error) {
	// Find user by email
	user, err := u.userRepo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	// Check if email is verified
	if !user.IsActive {
		return nil, errors.New("please verify your email first")
	}

	// Compare password
	if err := user.ComparePassword(req.Password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate tokens
	accessToken, err := middleware.GenerateToken(user.ID, user.Email, user.Role, u.jwtCfg)
	if err != nil {
		return nil, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(user.ID, user.Email, user.Role, u.jwtCfg)
	if err != nil {
		return nil, err
	}

	// Save refresh token to database
	refreshTokenExpiry := time.Now().Add(time.Duration(u.jwtCfg.RefreshExpiredHour) * time.Hour)
	if err := u.userRepo.SaveRefreshToken(user.ID, refreshToken, refreshTokenExpiry); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(u.jwtCfg.ExpiredHour * 3600),
		User: dto.UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			Avatar:   user.Avatar,
			IsActive: user.IsActive,
		},
	}, nil
}

func (u *userUsecase) ForgotPassword(emailAddress string) error {
	user, err := u.userRepo.FindByEmail(emailAddress)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !user.IsActive {
		return errors.New("please verify your email first")
	}

	resetToken, err := email.GenerateVerificationToken()
	if err != nil {
		return err
	}

	if err := u.saveResetPasswordToken(user.ID, resetToken); err != nil {
		return err
	}

	go func() {
		if err := u.emailService.SendPasswordResetEmail(user.Email, user.Name, resetToken, u.frontendBaseURL()); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", user.Email, err)
		} else {
			log.Printf("Password reset email sent successfully to %s", user.Email)
		}
	}()

	return nil
}

func (u *userUsecase) ResetPassword(token string, newPassword string) error {
	userID, err := u.findResetPasswordToken(token)
	if err != nil {
		if errors.Is(err, goredis.Nil) || errors.Is(err, sql.ErrNoRows) {
			return errors.New("invalid or expired reset token")
		}
		return err
	}

	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return err
	}

	if err := u.userRepo.UpdatePasswordUser(user.ID, user.Password); err != nil {
		return err
	}

	if err := u.deleteResetPasswordTokenPair(token, userID); err != nil {
		log.Printf("Failed to delete reset token %s from redis: %v", token, err)
	}

	return nil
}

func (u *userUsecase) GetProfile(userID string) (*dto.UserResponse, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	response := &dto.UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Avatar:   user.Avatar,
		IsActive: user.IsActive,
	}

	return response, nil
}

func (u *userUsecase) GetAllUsers(q *dto.QueryGlobal) ([]dto.UserListResponse, int, error) {
	users, count, err := u.userRepo.FindAll(q)
	if err != nil {
		return nil, 0, err
	}

	var res []dto.UserListResponse
	for _, user := range users {
		res = append(res, dto.UserListResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			Avatar:    user.Avatar,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		})
	}

	return res, count, nil
}

func (u *userUsecase) UpdateProfile(userID string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// Update fields
	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Avatar != nil {
		user.Avatar = req.Avatar
	}

	if err := u.userRepo.Update(userID, user); err != nil {
		return nil, err
	}

	response := &dto.UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Avatar:   user.Avatar,
		IsActive: user.IsActive,
	}

	return response, nil
}

func (u *userUsecase) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	// Verify old password
	if err := user.ComparePassword(req.OldPassword); err != nil {
		return errors.New("invalid old password")
	}

	// Set new password
	user.Password = req.NewPassword
	if err := user.HashPassword(); err != nil {
		return err
	}

	return u.userRepo.UpdatePasswordUser(userID, user.Password)
}

func (u *userUsecase) DeleteUser(userID string) error {
	_, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	return u.userRepo.Delete(userID)
}

func (u *userUsecase) RefreshToken(req *dto.RefreshTokenRequest) (*dto.AuthResponse, error) {
	// Find user by refresh token
	user, err := u.userRepo.FindByRefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired refresh token")
		}
		return nil, err
	}

	// Generate new access token
	accessToken, err := middleware.GenerateToken(user.ID, user.Email, user.Role, u.jwtCfg)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token
	newRefreshToken, err := middleware.GenerateRefreshToken(user.ID, user.Email, user.Role, u.jwtCfg)
	if err != nil {
		return nil, err
	}

	// Update refresh token in database
	refreshTokenExpiry := time.Now().Add(time.Duration(u.jwtCfg.RefreshExpiredHour) * time.Hour)
	if err := u.userRepo.SaveRefreshToken(user.ID, newRefreshToken, refreshTokenExpiry); err != nil {
		return nil, err
	}

	resp := &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(u.jwtCfg.ExpiredHour * 3600),
		User: dto.UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
			Avatar:   user.Avatar,
			IsActive: user.IsActive,
		},
	}

	return resp, nil
}

func (u *userUsecase) Logout(userID string) error {
	// Verify user exists
	_, err := u.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	// Clear refresh token
	return u.userRepo.ClearRefreshToken(userID)
}

func (u *userUsecase) UpdatePasswordUserByAdmin(id string, req *dto.UpdatePasswordUserRequest) error {
	user, err := u.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	user.Password = req.Password

	if err := user.HashPassword(); err != nil {
		return err
	}

	return u.userRepo.UpdatePasswordUser(user.ID, user.Password)
}

func (u *userUsecase) UpdateRoleUserByAdmin(id string, req *dto.UpdateRoleUserRequest) error {

	user, err := u.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	user.Role = req.Role
	return u.userRepo.UpdateRoleUser(user.ID, user.Role)
}

func (u *userUsecase) UpdateActiveUserByAdmin(id string) error {
	user, err := u.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	user.IsActive = !user.IsActive
	return u.userRepo.UpdateActiveUser(user.ID, user.IsActive)
}
