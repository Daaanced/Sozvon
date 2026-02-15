// Authorization/db/db.go - ОБНОВЛЕННАЯ ВЕРСИЯ
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"Authorization/config"
	"Authorization/models"

	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("DSN:", dsn)

	row := db.QueryRow("SELECT current_database()")
	var currentDB string
	row.Scan(&currentDB)
	log.Println("Connected to database:", currentDB)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// ✅ УЛУЧШЕННАЯ МИГРАЦИЯ с проверкой существующей структуры
func (d *Database) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Проверяем, существует ли таблица users
	var tableExists bool
	err := d.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`).Scan(&tableExists)

	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if tableExists {
		// Проверяем, есть ли колонка 'login'
		var loginColumnExists bool
		err = d.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.columns 
				WHERE table_schema = 'public' 
				AND table_name = 'users' 
				AND column_name = 'login'
			)
		`).Scan(&loginColumnExists)

		if err != nil {
			return fmt.Errorf("failed to check column existence: %w", err)
		}

		if !loginColumnExists {
			log.Println("⚠️  WARNING: Table 'users' exists but has wrong structure!")
			log.Println("⚠️  Dropping and recreating table...")

			// ❌ ОПАСНО для production! Только для development!
			_, err = d.db.ExecContext(ctx, "DROP TABLE IF EXISTS users CASCADE")
			if err != nil {
				return fmt.Errorf("failed to drop old table: %w", err)
			}

			tableExists = false // Теперь нужно создать таблицу заново
		}
	}

	if !tableExists {
		log.Println("Creating users table...")

		query := `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				login VARCHAR(255) UNIQUE NOT NULL,
				password VARCHAR(255) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX idx_users_login ON users(login);
		`

		_, err = d.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		log.Println("✅ Users table created successfully")
	} else {
		log.Println("✅ Users table already exists with correct structure")
	}

	return nil
}

// CreateUser создает нового пользователя
func (d *Database) CreateUser(ctx context.Context, user models.User) error {
	query := `
		INSERT INTO users (login, password, created_at, updated_at) 
		VALUES ($1, $2, NOW(), NOW())
	`

	_, err := d.db.ExecContext(ctx, query, user.Login, user.Password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByLogin возвращает пользователя по логину
func (d *Database) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	query := `
		SELECT id, login, password, created_at, updated_at 
		FROM users 
		WHERE login = $1
	`

	var user models.User
	err := d.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, fmt.Errorf("user not found: %s", login)
		}
		return models.User{}, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UserExists проверяет существование пользователя
func (d *Database) UserExists(ctx context.Context, login string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`

	var exists bool
	err := d.db.QueryRowContext(ctx, query, login).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// UpdateUser обновляет данные пользователя
func (d *Database) UpdateUser(ctx context.Context, user models.User) error {
	query := `
		UPDATE users 
		SET password = $1, updated_at = NOW() 
		WHERE login = $2
	`

	result, err := d.db.ExecContext(ctx, query, user.Password, user.Login)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.Login)
	}

	return nil
}

// DeleteUser удаляет пользователя
func (d *Database) DeleteUser(ctx context.Context, login string) error {
	query := `DELETE FROM users WHERE login = $1`

	result, err := d.db.ExecContext(ctx, query, login)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", login)
	}

	return nil
}

func (d *Database) GetDB() *sql.DB {
	return d.db
}
