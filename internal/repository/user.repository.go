package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/amirullazmi0/web-profile-backend/internal/dto"
	"github.com/amirullazmi0/web-profile-backend/internal/model"
	"github.com/amirullazmi0/web-profile-backend/pkg/database"
)

type UserRepository interface {
	Create(user *model.User) (string, error)
	FindByID(id string) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindByRefreshToken(refreshToken string) (*model.User, error)
	FindAll(q *dto.QueryGlobal) ([]model.User, int, error)
	Update(userID string, user *model.User) error
	SaveRefreshToken(userID string, refreshToken string, expiresAt time.Time) error
	ClearRefreshToken(userID string) error
	VerifyEmail(userID string) error
	Delete(id string) error
	CreateUserByAdmin(user *model.User) (string, error)
	UpdatePasswordUser(id string, password string) error
	UpdateRoleUser(id string, role string) error
	UpdateActiveUser(id string, isActive bool) error
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) (string, error) {
	id, err := database.NewInsertBuilder("users").
		Set("email", user.Email).
		Set("password", user.Password).
		Set("name", user.Name).
		Set("role", model.RoleUser).
		Set("is_active", false).
		Execute(r.db)

	return id, err
}

func (r *userRepository) CreateUserByAdmin(user *model.User) (string, error) {
	id, err := database.NewInsertBuilder("users").
		Set("email", user.Email).
		Set("password", user.Password).
		Set("name", user.Name).
		Set("role", user.Role).
		Execute(r.db)

	return id, err
}

func (r *userRepository) FindByID(id string) (*model.User, error) {
	query := `
		SELECT 
			u.id, u.email, u.password, u.name, u.role, u.avatar, 
			u.refresh_token, u.token_expiry, u.is_active, u.created_at, u.updated_at, u.deleted_at, 
			u.created_by, u.updated_by, u.deleted_by
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := database.RawQueryRow(r.db, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.Avatar,
		&user.RefreshToken,
		&user.TokenExpiry,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
		&user.CreatedBy,
		&user.UpdatedBy,
		&user.DeletedBy,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	fmt.Printf("user: %v\n", user)

	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	query := `
		SELECT 
			u.id, u.email, u.password, u.name, u.role, u.avatar, 
			u.refresh_token, u.token_expiry, u.is_active, u.created_at, u.updated_at, u.deleted_at, 
			u.created_by, u.updated_by, u.deleted_by
		FROM users u
		WHERE u.email = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := database.RawQueryRow(r.db, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.Avatar,
		&user.RefreshToken,
		&user.TokenExpiry,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
		&user.CreatedBy,
		&user.UpdatedBy,
		&user.DeletedBy,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindAll(q *dto.QueryGlobal) ([]model.User, int, error) {
	qb := database.NewQueryBuilder("users").
		Select("id", "email", "name", "role", "avatar", "refresh_token", "token_expiry", "is_active", "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "deleted_by").
		WhereNull("deleted_at")

	if q.Search != "" {
		term := "%" + q.Search + "%"
		qb.Where("(name ILIKE $1 OR email ILIKE $1)", term)
	}

	// Helpers for sorting and pagination
	sortBy, order := database.GetSorting(q.SortBy, q.Order, map[string]string{
		"id":         "id",
		"name":       "name",
		"email":      "email",
		"role":       "role",
		"created_at": "created_at",
		"is_active":  "is_active",
	}, "created_at")

	qb.OrderBy(sortBy + " " + order)

	// Count total data before pagination
	countQuery, countArgs := qb.BuildCount()
	var totalData int64
	err := database.RawQueryRow(r.db, countQuery, countArgs...).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// Pagination
	database.Paginate(qb, q.Page, q.Limit)

	query, args := qb.Build()

	rows, err := database.RawQuery(r.db, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID, &user.Email, &user.Name, &user.Role,
			&user.Avatar, &user.RefreshToken, &user.TokenExpiry, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
			&user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, int(totalData), nil
}

func (r *userRepository) Update(userID string, user *model.User) error {
	_, err := database.NewUpdateBuilder("users").
		Set("name", user.Name).
		Set("avatar", user.Avatar).
		Set("role", user.Role).
		Set("is_active", user.IsActive).
		Where("id = $1", userID).
		Execute(r.db)

	return err
}

func (r *userRepository) UpdatePasswordUser(id string, password string) error {
	_, err := database.NewUpdateBuilder("users").
		Set("password", password).
		Set("updated_at", time.Now()).
		Where("id = $1", id).
		Execute(r.db)

	return err
}

func (r *userRepository) UpdateRoleUser(id string, role string) error {
	_, err := database.NewUpdateBuilder("users").
		Set("role", role).
		Set("updated_at", time.Now()).
		Where("id = $1", id).
		Execute(r.db)

	return err
}

func (r *userRepository) Delete(id string) error {
	_, err := database.NewDeleteBuilder("users").
		Where("id = $1", id).
		Execute(r.db)

	return err
}

func (r *userRepository) SaveRefreshToken(userID string, refreshToken string, expiresAt time.Time) error {
	_, err := database.NewUpdateBuilder("users").
		Set("refresh_token", refreshToken).
		Set("token_expiry", expiresAt).
		Set("updated_at", time.Now()).
		Where("id = $1", userID).
		Execute(r.db)

	return err
}

func (r *userRepository) ClearRefreshToken(userID string) error {
	_, err := database.NewUpdateBuilder("users").
		Set("refresh_token", nil).
		Set("token_expiry", nil).
		Set("updated_at", time.Now()).
		Where("id = $1", userID).
		Execute(r.db)

	return err
}

func (r *userRepository) VerifyEmail(userID string) error {
	_, err := database.NewUpdateBuilder("users").
		Set("is_active", true).
		Set("updated_at", time.Now()).
		Where("id = $1", userID).
		Execute(r.db)

	return err
}

func (r *userRepository) FindByRefreshToken(refreshToken string) (*model.User, error) {
	query := `
		SELECT 
			u.id, u.email, u.password, u.name, u.role, u.refresh_token, u.token_expiry, 
			u.is_active, u.created_at, u.updated_at, u.deleted_at, u.created_by, u.updated_by, u.deleted_by
		FROM users u
		WHERE u.refresh_token = $1 AND u.deleted_at IS NULL AND u.token_expiry > $2
		LIMIT 1
	`

	var user model.User
	err := database.RawQueryRow(r.db, query, refreshToken, time.Now()).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.RefreshToken,
		&user.TokenExpiry,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
		&user.CreatedBy,
		&user.UpdatedBy,
		&user.DeletedBy,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateActiveUser(id string, isActive bool) error {
	_, err := database.NewUpdateBuilder("users").
		Set("is_active", isActive).
		Set("updated_at", time.Now()).
		Where("id = $1", id).
		Execute(r.db)

	return err
}
