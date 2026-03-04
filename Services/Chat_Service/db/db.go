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

func (d *Database) Migrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS chats (
			id         UUID PRIMARY KEY,
			active     BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS chat_members (
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL,
			PRIMARY KEY (chat_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS messages (
			id         UUID PRIMARY KEY,
			chat_id    UUID REFERENCES chats(id) ON DELETE CASCADE,
			sender_id  INTEGER NOT NULL,
			text       TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created ON messages(chat_id, created_at);
		CREATE INDEX IF NOT EXISTS idx_chats_active ON chats(active);
		CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members(user_id);
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := d.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// FindExistingChat ищет чат между двумя пользователями по user_id
func (d *Database) FindExistingChat(ctx context.Context, userID1, userID2 int) (string, error) {
	query := `
		SELECT c.id
		FROM chats c
		JOIN chat_members m1 ON m1.chat_id = c.id AND m1.user_id = $1
		JOIN chat_members m2 ON m2.chat_id = c.id AND m2.user_id = $2
		LIMIT 1
	`

	var chatID string
	err := d.db.QueryRowContext(ctx, query, userID1, userID2).Scan(&chatID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find chat: %w", err)
	}

	return chatID, nil
}

// CreateChat создаёт новый чат с участниками по user_id
func (d *Database) CreateChat(ctx context.Context, memberIDs []int, active bool) (string, error) {
	chatID := uuid.NewString()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chats (id, active) VALUES ($1, $2)`,
		chatID, active,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert chat: %w", err)
	}

	for _, userID := range memberIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)`,
			chatID, userID,
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

// GetChatMembers возвращает user_id участников чата
func (d *Database) GetChatMembers(ctx context.Context, chatID string) ([]int, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT user_id FROM chat_members WHERE chat_id = $1`,
		chatID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var members []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, uid)
	}

	return members, nil
}

// IsMember проверяет, является ли пользователь участником чата
func (d *Database) IsMember(ctx context.Context, chatID string, userID int) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 AND user_id = $2)`,
		chatID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return exists, nil
}

// SaveMessage сохраняет сообщение, sender — user_id
func (d *Database) SaveMessage(ctx context.Context, chatID string, senderID int, text string) (*models.Message, error) {
	messageID := uuid.NewString()
	now := time.Now()

	_, err := d.db.ExecContext(ctx,
		`INSERT INTO messages (id, chat_id, sender_id, text, created_at) VALUES ($1, $2, $3, $4, $5)`,
		messageID, chatID, senderID, text, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return &models.Message{
		ID:        messageID,
		ChatID:    chatID,
		SenderID:  senderID,
		Text:      text,
		CreatedAt: now,
	}, nil
}

// ActivateChat активирует чат при первом сообщении
func (d *Database) ActivateChat(ctx context.Context, chatID string) (bool, error) {
	var activated bool
	err := d.db.QueryRowContext(ctx,
		`UPDATE chats SET active = true WHERE id = $1 AND active = false RETURNING true`,
		chatID,
	).Scan(&activated)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to activate chat: %w", err)
	}
	return activated, nil
}

// GetUserChats возвращает список активных чатов пользователя
func (d *Database) GetUserChats(ctx context.Context, userID int) ([]models.ChatListItem, error) {
	query := `
		SELECT
			c.id,
			array_agg(cm.user_id) AS members,
			COALESCE(m.text, '')              AS last_message,
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
			SELECT chat_id FROM chat_members WHERE user_id = $1
		) AND c.active = true
		GROUP BY c.id, m.text, m.created_at, c.created_at
		ORDER BY COALESCE(m.created_at, c.created_at) DESC
	`

	rows, err := d.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}
	defer rows.Close()

	var chats []models.ChatListItem
	for rows.Next() {
		var item models.ChatListItem
		var members pqIntArray

		if err := rows.Scan(&item.ChatID, &members, &item.LastMessage, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chat: %w", err)
		}

		item.Members = []int(members)
		chats = append(chats, item)
	}

	return chats, nil
}

// GetChatMessages возвращает сообщения чата с пагинацией
func (d *Database) GetChatMessages(ctx context.Context, chatID string, limit, offset int) ([]models.Message, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, chat_id, sender_id, text, created_at
		 FROM messages
		 WHERE chat_id = $1
		 ORDER BY created_at ASC
		 LIMIT $2 OFFSET $3`,
		chatID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ChatExists проверяет существование чата
func (d *Database) ChatExists(ctx context.Context, chatID string) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)`,
		chatID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check chat existence: %w", err)
	}
	return exists, nil
}

// DeleteChatMembersByUserID удаляет все записи участника по user_id
func (d *Database) DeleteChatMembersByUserID(ctx context.Context, userID int) (int64, error) {
	result, err := d.db.ExecContext(ctx,
		`DELETE FROM chat_members WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete chat members: %w", err)
	}
	return result.RowsAffected()
}

func (d *Database) GetDB() *sql.DB {
	return d.db
}

// --- PostgreSQL int[] scanner ---

type pqIntArray []int

func (a *pqIntArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into pqIntArray", src)
	}

	// Формат: {1,2,3}
	if len(b) < 2 || b[0] != '{' || b[len(b)-1] != '}' {
		return fmt.Errorf("invalid array format")
	}
	content := string(b[1 : len(b)-1])
	if content == "" {
		*a = []int{}
		return nil
	}

	parts := splitArray(content)
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return fmt.Errorf("cannot parse int %q: %w", p, err)
		}
		result = append(result, n)
	}
	*a = result
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
