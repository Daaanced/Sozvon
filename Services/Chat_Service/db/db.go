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
		id         UUID      PRIMARY KEY,
		active     BOOLEAN   NOT NULL DEFAULT FALSE,
		type       TEXT      NOT NULL DEFAULT 'direct',
		name       TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS chat_members (
		chat_id             UUID    REFERENCES chats(id) ON DELETE CASCADE,
		user_id             INTEGER NOT NULL,
		last_read_message_id UUID   REFERENCES messages(id) ON DELETE SET NULL,
		PRIMARY KEY (chat_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS messages (
		id               UUID      PRIMARY KEY,
		chat_id          UUID      REFERENCES chats(id)    ON DELETE CASCADE,
		sender_id        INTEGER   NOT NULL,
		text             TEXT      NOT NULL DEFAULT '',
		reply_to_id      UUID      REFERENCES messages(id) ON DELETE SET NULL,
		-- пересылка: снимок
		forwarded_sender_id        INTEGER,
		forwarded_text             TEXT,
		forwarded_from_message_id  UUID,    -- только ref для отображения источника, без FK
		-- редактирование
		edited_at        TIMESTAMP,
		-- soft delete (показывать "сообщение удалено" вместо скрытия)
		deleted_at       TIMESTAMP,
		created_at       TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id         UUID    PRIMARY KEY,
		message_id UUID    REFERENCES messages(id) ON DELETE CASCADE,
		file_name  TEXT    NOT NULL,
		store_name TEXT    NOT NULL,
		mime_type  TEXT    NOT NULL,
		size       BIGINT  NOT NULL
	);

	-- Вложения пересланных сообщений (ссылки на те же файлы)
	CREATE TABLE IF NOT EXISTS forwarded_attachments (
		message_id    UUID REFERENCES messages(id)    ON DELETE CASCADE,
		attachment_id UUID REFERENCES attachments(id) ON DELETE CASCADE,
		PRIMARY KEY (message_id, attachment_id)
	);

	CREATE INDEX IF NOT EXISTS idx_attachments_message_id        ON attachments(message_id);
	CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created      ON messages(chat_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_messages_reply_to             ON messages(reply_to_id);
	CREATE INDEX IF NOT EXISTS idx_messages_forwarded_from       ON messages(forwarded_from_message_id);
	CREATE INDEX IF NOT EXISTS idx_chats_active                  ON chats(active);
	CREATE INDEX IF NOT EXISTS idx_chats_type                    ON chats(type);
	CREATE INDEX IF NOT EXISTS idx_chat_members_user_id          ON chat_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_chat_members_last_read        ON chat_members(last_read_message_id);
	CREATE INDEX IF NOT EXISTS idx_messages_deleted_at           ON messages(deleted_at);
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
func (d *Database) CreateChat(ctx context.Context, memberIDs []int, active bool, chatType string, name string) (string, error) {
	chatID := uuid.NewString()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chats (id, active, type, name) VALUES ($1, $2, $3, $4)`,
		chatID, active, chatType, name,
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
func (d *Database) SaveMessage(ctx context.Context, chatID string, senderID int, text string, replyToID *string) (*models.Message, error) {
	messageID := uuid.NewString()
	now := time.Now()

	_, err := d.db.ExecContext(ctx,
		`INSERT INTO messages (id, chat_id, sender_id, text, reply_to_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		messageID, chatID, senderID, text, replyToID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return &models.Message{
		ID:        messageID,
		ChatID:    chatID,
		SenderID:  senderID,
		Text:      text,
		ReplyToID: replyToID,
		CreatedAt: now,
	}, nil
}

func (d *Database) SaveForwardedMessages(ctx context.Context, toChatID string, senderID int, originalIDs []string) ([]*models.Message, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	var result []*models.Message
	now := time.Now()

	for _, origID := range originalIDs {
		// Читаем оригинал
		var orig models.Message
		var fwdSenderID sql.NullInt64
		var fwdText sql.NullString
		var fwdOrigID sql.NullString

		err := d.db.QueryRowContext(ctx,
			`SELECT id, sender_id, text, forwarded_sender_id, forwarded_text, forwarded_from_message_id
			 FROM messages WHERE id = $1 AND deleted_at IS NULL`,
			origID,
		).Scan(&orig.ID, &orig.SenderID, &orig.Text, &fwdSenderID, &fwdText, &fwdOrigID)
		if err != nil {
			return nil, fmt.Errorf("original message not found: %w", err)
		}

		// Снимок: если оригинал сам пересланный — берём его снимок
		snapshotSenderID := orig.SenderID
		snapshotText := orig.Text
		snapshotOrigID := &orig.ID
		if fwdSenderID.Valid {
			snapshotSenderID = int(fwdSenderID.Int64)
			snapshotText = fwdText.String
			if fwdOrigID.Valid {
				snapshotOrigID = &fwdOrigID.String
			}
		}

		newID := uuid.NewString()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO messages (id, chat_id, sender_id, text, forwarded_sender_id, forwarded_text, forwarded_from_message_id, created_at)
			 VALUES ($1, $2, $3, '', $4, $5, $6, $7)`,
			newID, toChatID, senderID, snapshotSenderID, snapshotText, snapshotOrigID, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert forwarded message: %w", err)
		}

		// Копируем ссылки на вложения через forwarded_attachments
		attRows, err := d.db.QueryContext(ctx,
			`SELECT id FROM attachments WHERE message_id = $1`, origID,
		)
		if err != nil {
			return nil, err
		}
		for attRows.Next() {
			var attID string
			if err := attRows.Scan(&attID); err != nil {
				attRows.Close()
				return nil, err
			}
			_, err = tx.ExecContext(ctx,
				`INSERT INTO forwarded_attachments (message_id, attachment_id) VALUES ($1, $2)`,
				newID, attID,
			)
			if err != nil {
				attRows.Close()
				return nil, err
			}
		}
		attRows.Close()

		result = append(result, &models.Message{
			ID:        newID,
			ChatID:    toChatID,
			SenderID:  senderID,
			CreatedAt: now,
			ForwardedFrom: &models.ForwardedMeta{
				OriginalMessageID: snapshotOrigID,
				SenderID:          snapshotSenderID,
				Text:              snapshotText,
			},
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return result, nil
}

func (d *Database) EditMessage(ctx context.Context, messageID string, senderID int, newText string) error {
	result, err := d.db.ExecContext(ctx,
		`UPDATE messages SET text = $1, edited_at = NOW()
		 WHERE id = $2 AND sender_id = $3 AND deleted_at IS NULL`,
		newText, messageID, senderID,
	)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found or not yours")
	}
	return nil
}

func (d *Database) DeleteMessage(ctx context.Context, messageID string, senderID int) ([]string, error) {
	// Проверяем авторство
	var ownerID int
	err := d.db.QueryRowContext(ctx,
		`SELECT sender_id FROM messages WHERE id = $1 AND deleted_at IS NULL`,
		messageID,
	).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	if ownerID != senderID {
		return nil, fmt.Errorf("not your message")
	}

	// Получаем store_name вложений для удаления файлов
	rows, err := d.db.QueryContext(ctx,
		`SELECT store_name FROM attachments WHERE message_id = $1`, messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var storeNames []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		storeNames = append(storeNames, s)
	}

	// Физическое удаление — attachments каскадно удалятся
	_, err = d.db.ExecContext(ctx,
		`DELETE FROM messages WHERE id = $1`, messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}

	return storeNames, nil
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
	// Заменить GetUserChats query:
	query := `
		SELECT
			c.id,
			c.type,
			COALESCE(c.name, '') AS name,
			array_agg(cm2.user_id) AS members,
			COALESCE(m.text, '') AS last_message,
			COALESCE(m.created_at, c.created_at) AS updated_at,
			COALESCE((
				SELECT COUNT(*) FROM messages unread
				WHERE unread.chat_id = c.id
				AND unread.sender_id != $1
				AND unread.deleted_at IS NULL
				AND (
					cm_me.last_read_message_id IS NULL
					OR unread.created_at > (
						SELECT created_at FROM messages
						WHERE id = cm_me.last_read_message_id
					)
				)
			), 0) AS unread_count
		FROM chats c
		JOIN chat_members cm_me ON cm_me.chat_id = c.id AND cm_me.user_id = $1
		JOIN chat_members cm2 ON cm2.chat_id = c.id
		LEFT JOIN LATERAL (
			SELECT text, created_at FROM messages
			WHERE chat_id = c.id AND deleted_at IS NULL
			ORDER BY created_at DESC LIMIT 1
		) m ON true
		WHERE c.active = true
		GROUP BY c.id, c.type, c.name, m.text, m.created_at, c.created_at, cm_me.last_read_message_id
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

		if err := rows.Scan(&item.ChatID, &item.Type, &item.Name, &members, &item.LastMessage, &item.UpdatedAt, &item.UnreadCount); err != nil {
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
		`SELECT
			m.id, m.chat_id, m.sender_id, m.text,
			m.reply_to_id, m.edited_at, m.deleted_at, m.created_at,
			r.id, r.sender_id, r.text,
			m.forwarded_sender_id, m.forwarded_text, m.forwarded_from_message_id
		FROM messages m
		LEFT JOIN messages r ON r.id = m.reply_to_id AND r.deleted_at IS NULL
		WHERE m.chat_id = $1
		ORDER BY m.created_at ASC
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
		var replyID, rID, fwdOrigID sql.NullString
		var rSenderID, fwdSenderID sql.NullInt64
		var rText, fwdText sql.NullString
		var editedAt, deletedAt sql.NullTime

		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
			&replyID, &editedAt, &deletedAt, &msg.CreatedAt,
			&rID, &rSenderID, &rText,
			&fwdSenderID, &fwdText, &fwdOrigID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if replyID.Valid {
			msg.ReplyToID = &replyID.String
		}
		if editedAt.Valid {
			msg.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.Time
			// Затираем содержимое удалённого сообщения
			msg.Text = ""
			msg.Attachments = nil
		}
		if rID.Valid && deletedAt.Valid == false {
			msg.ReplyToMessage = &models.ReplyPreview{
				ID:       rID.String,
				SenderID: int(rSenderID.Int64),
				Text:     rText.String,
			}
		}
		if fwdSenderID.Valid {
			origID := (*string)(nil)
			if fwdOrigID.Valid {
				origID = &fwdOrigID.String
			}
			msg.ForwardedFrom = &models.ForwardedMeta{
				OriginalMessageID: origID,
				SenderID:          int(fwdSenderID.Int64),
				Text:              fwdText.String,
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (d *Database) MarkChatRead(ctx context.Context, chatID string, userID int, lastMessageID string) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE chat_members SET last_read_message_id = $1
		 WHERE chat_id = $2 AND user_id = $3`,
		lastMessageID, chatID, userID,
	)
	return err
}

func (d *Database) GetMessagesAroundID(ctx context.Context, chatID string, messageID string, around int) ([]models.Message, error) {
	query := `
		WITH ranked AS (
			SELECT id, chat_id, sender_id, text, reply_to_id, edited_at, deleted_at,
			       forwarded_sender_id, forwarded_text, forwarded_from_message_id,
			       created_at,
			       ROW_NUMBER() OVER (ORDER BY created_at ASC) AS rn
			FROM messages
			WHERE chat_id = $1
		),
		target_rn AS (
			SELECT rn FROM ranked WHERE id = $2
		)
		SELECT id, chat_id, sender_id, text, reply_to_id, edited_at, deleted_at,
		       forwarded_sender_id, forwarded_text, forwarded_from_message_id,
		       created_at
		FROM ranked
		WHERE rn BETWEEN (SELECT rn FROM target_rn) - $3
		              AND (SELECT rn FROM target_rn) + $3
		ORDER BY created_at ASC
	`
	rows, err := d.db.QueryContext(ctx, query, chatID, messageID, around)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages around: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var replyID, fwdOrigID sql.NullString
		var fwdSenderID sql.NullInt64
		var fwdText sql.NullString
		var editedAt, deletedAt sql.NullTime

		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
			&replyID, &editedAt, &deletedAt,
			&fwdSenderID, &fwdText, &fwdOrigID,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if replyID.Valid {
			msg.ReplyToID = &replyID.String
		}
		if editedAt.Valid {
			msg.EditedAt = &editedAt.Time
		}
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.Time
			msg.Text = ""
			msg.Attachments = nil
		}
		if fwdSenderID.Valid {
			origID := (*string)(nil)
			if fwdOrigID.Valid {
				origID = &fwdOrigID.String
			}
			msg.ForwardedFrom = &models.ForwardedMeta{
				OriginalMessageID: origID,
				SenderID:          int(fwdSenderID.Int64),
				Text:              fwdText.String,
			}
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
