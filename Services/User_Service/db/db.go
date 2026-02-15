// User_Service/db/db.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"User_Service/config"
	"User_Service/models"

	_ "github.com/lib/pq"
)

// Database представляет соединение с базой данных
type Database struct {
	db     *sql.DB
	config *config.Config
}

// NewDatabase создает новое подключение к БД с connection pooling
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	log.Println("DSN:", dsn)
	// Проверка соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

// Close закрывает соединение с БД
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Migrate выполняет миграции базы данных
func (d *Database) Migrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			login VARCHAR(50) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255),
			info TEXT,
			picture VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := d.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetAllUsers возвращает всех пользователей с пагинацией
func (d *Database) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, int, error) {
	// Получение общего количества
	var total int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Получение пользователей с пагинацией
	query := `
		SELECT id, login, name, email, info, picture, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := d.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID,
			&u.Login,
			&u.Name,
			&u.Email,
			&u.Info,
			&u.Picture,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return users, total, nil
}

// GetUserByLogin возвращает пользователя по логину
func (d *Database) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	query := `
		SELECT id, login, name, email, info, picture, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	var u models.User
	err := d.db.QueryRowContext(ctx, query, login).Scan(
		&u.ID,
		&u.Login,
		&u.Name,
		&u.Email,
		&u.Info,
		&u.Picture,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, fmt.Errorf("user not found: %s", login)
		}
		return models.User{}, fmt.Errorf("failed to get user: %w", err)
	}

	return u, nil
}

// CreateUser создает нового пользователя
func (d *Database) CreateUser(ctx context.Context, user models.User) error {
	query := `
		INSERT INTO users (login, name, email, info, picture, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`

	_, err := d.db.ExecContext(
		ctx,
		query,
		user.Login,
		user.Name,
		user.Email,
		user.Info,
		user.Picture,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUser обновляет данные пользователя
func (d *Database) UpdateUser(ctx context.Context, login string, user models.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, info = $3, picture = $4, updated_at = NOW()
		WHERE login = $5
	`

	result, err := d.db.ExecContext(
		ctx,
		query,
		user.Name,
		user.Email,
		user.Info,
		user.Picture,
		login,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
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

// SearchUsers поиск пользователей по имени или логину
func (d *Database) SearchUsers(ctx context.Context, searchTerm string, limit, offset int) ([]models.User, int, error) {
	searchPattern := "%" + searchTerm + "%"

	// Подсчет результатов
	var total int
	countQuery := `
		SELECT COUNT(*) FROM users
		WHERE login ILIKE $1 OR name ILIKE $1
	`
	err := d.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Поиск пользователей
	query := `
		SELECT id, login, name, email, info, picture, created_at, updated_at
		FROM users
		WHERE login ILIKE $1 OR name ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := d.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID,
			&u.Login,
			&u.Name,
			&u.Email,
			&u.Info,
			&u.Picture,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

// GetDB возвращает *sql.DB для прямого доступа
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// SetConfig устанавливает конфигурацию (для avatar URL)
func (d *Database) SetConfig(cfg *config.Config) {
	d.config = cfg
}

// --- Обратная совместимость ---

var DB *sql.DB

func Init() {
	cfg := config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "",
		DBName:          "userdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	database, err := NewDatabase(cfg)
	if err != nil {
		panic(err)
	}

	DB = database.db
}

func GetAllUsers() ([]models.User, error) {
	ctx := context.Background()
	rows, err := DB.QueryContext(ctx, "SELECT login, name, email, info, picture FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.Login, &u.Name, &u.Email, &u.Info, &u.Picture); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUserByLogin(login string) (models.User, error) {
	ctx := context.Background()
	var u models.User
	err := DB.QueryRowContext(ctx,
		"SELECT login, name, email, info, picture FROM users WHERE login=$1",
		login,
	).Scan(&u.Login, &u.Name, &u.Email, &u.Info, &u.Picture)
	return u, err
}

func CreateUser(u models.User) error {
	ctx := context.Background()
	_, err := DB.ExecContext(ctx,
		"INSERT INTO users (login, name, email, info, picture) VALUES ($1,$2,$3,$4,$5)",
		u.Login, u.Name, u.Email, u.Info, u.Picture,
	)
	return err
}

func DeleteUser(login string) error {
	ctx := context.Background()
	_, err := DB.ExecContext(ctx, "DELETE FROM users WHERE login=$1", login)
	return err
}

func UpdateUser(u models.User) error {
	ctx := context.Background()
	_, err := DB.ExecContext(ctx,
		"UPDATE users SET name=$1, email=$2, info=$3, picture=$4 WHERE login=$5",
		u.Name, u.Email, u.Info, u.Picture, u.Login,
	)
	return err
}
