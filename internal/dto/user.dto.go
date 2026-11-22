package dto

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
}

type RegisterUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
	Role     string `json:"role" validate:"required"`
}

// LoginRequest represents user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UpdateUserRequest represents user update request
type UpdateUserRequest struct {
	Name   string  `json:"name" validate:"omitempty,min=2"`
	Avatar *string `json:"avatar" validate:"omitempty,url"`
}

type UpdateUserByAdminRequest struct {
	ID     string  `json:"id" validate:"required"`
	Name   string  `json:"name" validate:"omitempty,min=2"`
	Avatar *string `json:"avatar" validate:"omitempty,url"`
	Role   string  `json:"role" validate:"omitempty"`
}

// ChangePasswordRequest represents change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// UserResponse represents user response
type UserResponse struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	Avatar   *string `json:"avatar"`
	IsActive bool    `json:"is_active"`
}

// UserListResponse represents user response for list (includes timestamps)
type UserListResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	Avatar    *string `json:"avatar"`
	IsActive  bool    `json:"is_active"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
	User         UserResponse `json:"user"`
}

type UpdatePasswordUserRequest struct {
	Password string `json:"password" validate:"required,min=6"`
}

type UpdateRoleUserRequest struct {
	Role string `json:"role" validate:"required"`
}
