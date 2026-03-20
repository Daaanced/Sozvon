// Chat_Service\db\chats.go

package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"Chat_Service/models"

	"github.com/google/uuid"
)

// FindExistingChat ищет прямой чат между двумя пользователями.
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

// CreateChat создаёт новый чат и добавляет участников.
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

// ActivateChat активирует чат при первом сообщении.
// Возвращает true, если чат был только что активирован.
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

// ChatExists проверяет существование чата по id.
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

// GetUserChats возвращает список активных чатов пользователя,
// отсортированных по дате последнего сообщения.
func (d *Database) GetUserChats(ctx context.Context, userID int) ([]models.ChatListItem, error) {
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
		JOIN chat_members cm2   ON cm2.chat_id = c.id
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

	var updatedAt time.Time
	var chats []models.ChatListItem
	for rows.Next() {
		var item models.ChatListItem
		var members pqIntArray

		if err := rows.Scan(
			&item.ChatID, &item.Type, &item.Name,
			&members,
			&item.LastMessage, &updatedAt, &item.UnreadCount,
		); err != nil {
			log.Printf("scan chat error: %v", err)
			return nil, fmt.Errorf("failed to scan chat: %w", err)
		}

		item.UpdatedAt = models.UTCTime{Time: updatedAt.UTC()}
		item.Members = []int(members)
		chats = append(chats, item)
	}

	return chats, nil
}
