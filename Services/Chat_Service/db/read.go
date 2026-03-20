//Chat_Service\db\read.go

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Chat_Service/models"
)

// MarkChatRead обновляет курсор последнего прочитанного сообщения для пользователя.
func (d *Database) MarkChatRead(ctx context.Context, chatID string, userID int, lastMessageID string) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE chat_members SET last_read_message_id = $1
		 WHERE chat_id = $2 AND user_id = $3`,
		lastMessageID, chatID, userID,
	)
	return err
}

// GetMessagesAroundID возвращает сообщения вокруг указанного id (±around штук).
// Используется для перехода к цитируемому или найденному сообщению.
func (d *Database) GetMessagesAroundID(ctx context.Context, chatID string, messageID string, around int) ([]models.Message, error) {
	query := `
		WITH ranked AS (
			SELECT
				id, chat_id, sender_id, text, reply_to_id,
				edited_at, deleted_at,
				forwarded_sender_id, forwarded_text, forwarded_from_message_id,
				created_at,
				ROW_NUMBER() OVER (ORDER BY created_at ASC) AS rn
			FROM messages
			WHERE chat_id = $1
		),
		target_rn AS (
			SELECT rn FROM ranked WHERE id = $2
		)
		SELECT
			id, chat_id, sender_id, text, reply_to_id,
			edited_at, deleted_at,
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
		msg, err := scanAroundMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (d *Database) GetMessagesFromUnread(ctx context.Context, chatID string, userID int, around int) (*models.UnreadResult, error) {
	// Находим last_read_message_id пользователя
	var lastReadID sql.NullString
	err := d.db.QueryRowContext(ctx,
		`SELECT last_read_message_id FROM chat_members
         WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID,
	).Scan(&lastReadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last read: %w", err)
	}

	// Если всё прочитано — отдаём последние N сообщений обычным способом
	if !lastReadID.Valid {
		return nil, nil
	}

	// Находим первое непрочитанное
	var firstUnreadID string
	err = d.db.QueryRowContext(ctx,
		`SELECT id FROM messages
         WHERE chat_id = $1
           AND deleted_at IS NULL
           AND created_at > (
               SELECT created_at FROM messages WHERE id = $2
           )
         ORDER BY created_at ASC
         LIMIT 1`,
		chatID, lastReadID.String,
	).Scan(&firstUnreadID)
	if err == sql.ErrNoRows {
		// Нет непрочитанных
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find first unread: %w", err)
	}

	// Считаем количество непрочитанных
	var totalUnread int
	err = d.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages
         WHERE chat_id = $1
           AND sender_id != $2
           AND deleted_at IS NULL
           AND created_at > (
               SELECT created_at FROM messages WHERE id = $3
           )`,
		chatID, userID, lastReadID.String,
	).Scan(&totalUnread)
	if err != nil {
		return nil, fmt.Errorf("failed to count unread: %w", err)
	}

	// Грузим сообщения вокруг первого непрочитанного
	messages, err := d.GetMessagesAroundID(ctx, chatID, firstUnreadID, around)
	if err != nil {
		return nil, err
	}

	return &models.UnreadResult{
		Messages:      messages,
		FirstUnreadID: firstUnreadID,
		TotalUnread:   totalUnread,
	}, nil
}

// scanAroundMessage читает одну строку из запроса GetMessagesAroundID.
// Отличается от scanMessage набором колонок (нет JOIN на reply).
func scanAroundMessage(rows *sql.Rows) (models.Message, error) {
	var msg models.Message
	var replyID, fwdOrigID sql.NullString
	var fwdSenderID sql.NullInt64
	var fwdText sql.NullString

	var createdAt time.Time
	var editedAt, deletedAt sql.NullTime

	if err := rows.Scan(
		&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
		&replyID, &editedAt, &deletedAt,
		&fwdSenderID, &fwdText, &fwdOrigID,
		&createdAt,
	); err != nil {
		return msg, fmt.Errorf("failed to scan message: %w", err)
	}

	msg.CreatedAt = models.UTCTime{Time: createdAt.UTC()}

	if replyID.Valid {
		msg.ReplyToID = &replyID.String
	}
	if editedAt.Valid {
		t := models.UTCTime{Time: editedAt.Time.UTC()}
		msg.EditedAt = &t
	}
	if deletedAt.Valid {
		t := models.UTCTime{Time: deletedAt.Time.UTC()}
		msg.DeletedAt = &t
		msg.Text = ""
		msg.Attachments = nil
	}
	if fwdSenderID.Valid {
		var origID *string
		if fwdOrigID.Valid {
			origID = &fwdOrigID.String
		}
		msg.ForwardedFrom = &models.ForwardedMeta{
			OriginalMessageID: origID,
			SenderID:          int(fwdSenderID.Int64),
			Text:              fwdText.String,
		}
	}

	return msg, nil
}
