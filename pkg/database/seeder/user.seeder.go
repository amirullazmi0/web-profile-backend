package seeder

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/internal/model"
	"github.com/amirullazmi0/web-profile-backend/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// SeedSuperAdmin seeds the initial super admin user from SEED_SUPERADMIN_* env vars.
func SeedSuperAdmin(db *sql.DB, cfg *config.SeedConfig) error {
	email := cfg.SuperAdminEmail
	password := cfg.SuperAdminPassword
	name := cfg.SuperAdminName
	role := model.RoleSuperAdmin

	if email == "" || password == "" {
		return errors.New("SEED_SUPERADMIN_EMAIL and SEED_SUPERADMIN_PASSWORD must be set")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if user already exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		logger.Info("Superadmin already exists, skipping...", zap.String("email", email))
		return nil
	}

	// Insert user
	// Note: use gen_random_uuid() for id matching schema.prisma default
	query := `
		INSERT INTO users (id, email, password, name, role, is_active, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW(), NOW())
	`
	_, err = db.Exec(query, email, string(hashedPassword), name, role, true)
	if err != nil {
		return fmt.Errorf("failed to insert superadmin: %w", err)
	}

	logger.Info("Superadmin seeded successfully", zap.String("email", email))
	return nil
}
