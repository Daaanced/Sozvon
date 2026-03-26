//Chat_Service\db\messages.go

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Chat_Service/models"

	"github.com/google/uuid"
)

// SaveMessage сохраняет новое сообщение и возвращает его модель.
func (d *Database) SaveMessage(ctx context.Context, chatID string, senderID int, text string, replyToID *string) (*models.Message, error) {
	messageID := uuid.NewString()
	now := time.Now().UTC()

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
		CreatedAt: models.UTCTime{Time: now},
	}, nil
}

// EditMessage обновляет текст сообщения. Только автор может редактировать.
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

// DeleteMessage удаляет сообщение и возвращает store_name вложений для удаления файлов.
// Только автор может удалить своё сообщение.
func (d *Database) DeleteMessage(ctx context.Context, messageID string, senderID int) ([]string, error) {
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

	rows, err := d.db.QueryContext(ctx,
		`SELECT store_name FROM attachments WHERE message_id = $1`,
		messageID,
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

	// attachments каскадно удалятся вместе с сообщением
	_, err = d.db.ExecContext(ctx,
		`DELETE FROM messages WHERE id = $1`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to delete message: %w", err)
	}

	return storeNames, nil
}

// GetChatMessages возвращает сообщения чата с пагинацией (новые → старые, потом реверсируются).
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
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3`,
		chatID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Разворачиваем: в БД читали DESC, отдаём хронологически
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// scanMessage читает одну строку из запроса GetChatMessages.
func scanMessage(rows *sql.Rows) (models.Message, error) {
	var msg models.Message
	var replyID, rID, fwdOrigID sql.NullString
	var rSenderID, fwdSenderID sql.NullInt64
	var rText, fwdText sql.NullString

	var createdAt time.Time
	var editedAt, deletedAt sql.NullTime

	if err := rows.Scan(
		&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Text,
		&replyID, &editedAt, &deletedAt, &createdAt,
		&rID, &rSenderID, &rText,
		&fwdSenderID, &fwdText, &fwdOrigID,
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
	if rID.Valid && !deletedAt.Valid {
		msg.ReplyToMessage = &models.ReplyPreview{
			ID:       rID.String,
			SenderID: int(rSenderID.Int64),
			Text:     rText.String,
		}
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

func (d *Database) GetMessagesAfterID(ctx context.Context, chatID, messageID string, limit int) ([]models.Message, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT
            m.id, m.chat_id, m.sender_id, m.text,
            m.reply_to_id, m.edited_at, m.deleted_at, m.created_at,
            r.id, r.sender_id, r.text,
            m.forwarded_sender_id, m.forwarded_text, m.forwarded_from_message_id
        FROM messages m
        LEFT JOIN messages r ON r.id = m.reply_to_id AND r.deleted_at IS NULL
        WHERE m.chat_id = $1
          AND m.created_at > (SELECT created_at FROM messages WHERE id = $2)
        ORDER BY m.created_at ASC
        LIMIT $3`,
		chatID, messageID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages after: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (d *Database) GetMessagesBeforeID(ctx context.Context, chatID, messageID string, limit int) ([]models.Message, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT
            m.id, m.chat_id, m.sender_id, m.text,
            m.reply_to_id, m.edited_at, m.deleted_at, m.created_at,
            r.id, r.sender_id, r.text,
            m.forwarded_sender_id, m.forwarded_text, m.forwarded_from_message_id
        FROM messages m
        LEFT JOIN messages r ON r.id = m.reply_to_id AND r.deleted_at IS NULL
        WHERE m.chat_id = $1
          AND m.created_at < (SELECT created_at FROM messages WHERE id = $2)
        ORDER BY m.created_at DESC
        LIMIT $3`,
		chatID, messageID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages before: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// разворачиваем — читали DESC, отдаём хронологически
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
