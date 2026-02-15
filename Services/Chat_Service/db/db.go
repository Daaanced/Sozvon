// Chat_Service/db/db.go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Chat_Service/config"
	"Chat_Service/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Database представляет соединение с базой данных
type Database struct {
	db *sql.DB
}

// NewDatabase создает новое подключение к БД
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pooling
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

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

// Migrate выполняет миграции
func (d *Database) Migrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS chats (
			id UUID PRIMARY KEY,
			active BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS chat_members (
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			login TEXT NOT NULL,
			PRIMARY KEY (chat_id, login)
		);

		CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY,
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			sender_login TEXT NOT NULL,
			text TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created ON messages(chat_id, created_at);
		CREATE INDEX IF NOT EXISTS idx_chats_active ON chats(active);
		CREATE INDEX IF NOT EXISTS idx_chat_members_login ON chat_members(login);
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := d.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// FindExistingChat ищет существующий чат между двумя пользователями
func (d *Database) FindExistingChat(ctx context.Context, user1, user2 string) (string, error) {
	query := `
		SELECT c.id
		FROM chats c
		JOIN chat_members m1 ON m1.chat_id = c.id AND m1.login = $1
		JOIN chat_members m2 ON m2.chat_id = c.id AND m2.login = $2
		LIMIT 1
	`

	var chatID string
	err := d.db.QueryRowContext(ctx, query, user1, user2).Scan(&chatID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find chat: %w", err)
	}

	return chatID, nil
}

// CreateChat создает новый чат
func (d *Database) CreateChat(ctx context.Context, members []string, active bool) (string, error) {
	chatID := uuid.NewString()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Создание чата
	_, err = tx.ExecContext(ctx,
		`INSERT INTO chats (id, active) VALUES ($1, $2)`,
		chatID, active,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert chat: %w", err)
	}

	// Добавление участников
	for _, member := range members {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO chat_members (chat_id, login) VALUES ($1, $2)`,
			chatID, member,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chatID, nil
}

// GetChatMembers возвращает участников чата
func (d *Database) GetChatMembers(ctx context.Context, chatID string) ([]string, error) {
	query := `SELECT login FROM chat_members WHERE chat_id = $1`

	rows, err := d.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var login string
		if err := rows.Scan(&login); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, login)
	}

	return members, nil
}

// SaveMessage сохраняет сообщение
func (d *Database) SaveMessage(ctx context.Context, chatID, senderLogin, text string) (string, error) {
	messageID := uuid.NewString()

	query := `
		INSERT INTO messages (id, chat_id, sender_login, text)
		VALUES ($1, $2, $3, $4)
	`

	_, err := d.db.ExecContext(ctx, query, messageID, chatID, senderLogin, text)
	if err != nil {
		return "", fmt.Errorf("failed to save message: %w", err)
	}

	return messageID, nil
}

// ActivateChat активирует чат
func (d *Database) ActivateChat(ctx context.Context, chatID string) (bool, error) {
	query := `
		UPDATE chats
		SET active = true
		WHERE id = $1 AND active = false
		RETURNING true
	`

	var activated bool
	err := d.db.QueryRowContext(ctx, query, chatID).Scan(&activated)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to activate chat: %w", err)
	}

	return activated, nil
}

// GetUserChats возвращает список чатов пользователя
func (d *Database) GetUserChats(ctx context.Context, login string) ([]models.ChatListItem, error) {
	query := `
		SELECT
			c.id,
			array_agg(cm.login) AS members,
			COALESCE(m.text, '') AS last_message,
			COALESCE(m.created_at, c.created_at) AS updated_at
		FROM chats c
		JOIN chat_members cm ON cm.chat_id = c.id
		LEFT JOIN LATERAL (
			SELECT text, created_at
			FROM messages
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		WHERE c.id IN (
			SELECT chat_id FROM chat_members WHERE login = $1
		) AND c.active = true
		GROUP BY c.id, m.text, m.created_at, c.created_at
		ORDER BY COALESCE(m.created_at, c.created_at) DESC
	`

	rows, err := d.db.QueryContext(ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}
	defer rows.Close()

	var chats []models.ChatListItem
	for rows.Next() {
		var item models.ChatListItem
		var members pqStringArray

		err := rows.Scan(
			&item.ChatID,
			&members,
			&item.LastMessage,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat: %w", err)
		}

		item.Members = []string(members)
		chats = append(chats, item)
	}

	return chats, nil
}

// GetChatMessages возвращает сообщения чата
func (d *Database) GetChatMessages(ctx context.Context, chatID string, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT id, chat_id, sender_login, text, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := d.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderLogin,
			&msg.Text,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ChatExists проверяет существование чата
func (d *Database) ChatExists(ctx context.Context, chatID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)`

	var exists bool
	err := d.db.QueryRowContext(ctx, query, chatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check chat existence: %w", err)
	}

	return exists, nil
}

// pqStringArray для работы с PostgreSQL массивами
type pqStringArray []string

func (a *pqStringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return a.scanBytes(v)
	case string:
		return a.scanBytes([]byte(v))
	default:
		return fmt.Errorf("cannot scan %T into pqStringArray", src)
	}
}

func (a *pqStringArray) scanBytes(src []byte) error {
	// Простой парсер PostgreSQL массивов: {val1,val2,val3}
	if len(src) < 2 || src[0] != '{' || src[len(src)-1] != '}' {
		return fmt.Errorf("invalid array format")
	}

	content := string(src[1 : len(src)-1])
	if content == "" {
		*a = []string{}
		return nil
	}

	// Разделение по запятым (упрощенная версия)
	*a = splitArray(content)
	return nil
}

func splitArray(s string) []string {
	var result []string
	var current string
	inQuotes := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				result = append(result, current)
				current = ""
				continue
			}
			current += string(c)
		default:
			current += string(c)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// GetDB возвращает *sql.DB
func (d *Database) GetDB() *sql.DB {
	return d.db
}
