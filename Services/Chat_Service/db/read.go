package db

import (
	"context"
	"database/sql"
	"fmt"

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

// scanAroundMessage читает одну строку из запроса GetMessagesAroundID.
// Отличается от scanMessage набором колонок (нет JOIN на reply).
func scanAroundMessage(rows *sql.Rows) (models.Message, error) {
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
		return msg, fmt.Errorf("failed to scan message: %w", err)
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
