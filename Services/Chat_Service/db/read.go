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
         WHERE chat_id = $2 AND user_id = $3
           AND (
               last_read_message_id IS NULL
               OR (
                   SELECT created_at FROM messages WHERE id = $1
               ) > (
                   SELECT created_at FROM messages WHERE id = last_read_message_id
               )
           )`,
		lastMessageID, chatID, userID,
	)
	return err
}

// GetMessagesAroundID возвращает сообщения вокруг указанного id (±around штук).
// Используется для перехода к цитируемому или найденному сообщению.
func (d *Database) GetMessagesAroundID(ctx context.Context, chatID string, messageID string, around int) ([]models.Message, error) {
	before, err := d.GetMessagesBeforeID(ctx, chatID, messageID, around)
	if err != nil {
		return nil, err
	}

	after, err := d.GetMessagesAfterID(ctx, chatID, messageID, around)
	if err != nil {
		return nil, err
	}

	// Берём само целевое сообщение через GetMessagesAfterID с limit=1 не подойдёт,
	// поэтому делаем отдельный запрос с тем же SELECT что в scanMessage
	rows, err := d.db.QueryContext(ctx,
		`SELECT
			m.id, m.chat_id, m.sender_id, m.text,
			m.reply_to_id, m.edited_at, m.deleted_at, m.created_at,
			r.id, r.sender_id, r.text,
			m.forwarded_sender_id, m.forwarded_text, m.forwarded_from_message_id
		FROM messages m
		LEFT JOIN messages r ON r.id = m.reply_to_id AND r.deleted_at IS NULL
		WHERE m.id = $1`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get target message: %w", err)
	}
	defer rows.Close()

	var target *models.Message
	if rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		target = &msg
	}

	result := before
	if target != nil {
		result = append(result, *target)
	}
	result = append(result, after...)
	return result, nil
}

func (d *Database) GetMessagesFromUnread(ctx context.Context, chatID string, userID int, around int) (*models.UnreadResult, error) {
	var lastReadID sql.NullString
	err := d.db.QueryRowContext(ctx,
		`SELECT last_read_message_id FROM chat_members
         WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID,
	).Scan(&lastReadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last read: %w", err)
	}

	if !lastReadID.Valid {
		return nil, nil
	}

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
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find first unread: %w", err)
	}

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

	messages, err := d.GetMessagesAroundID(ctx, chatID, firstUnreadID, around)
	if err != nil {
		return nil, err
	}

	var hasMoreTop bool
	if len(messages) > 0 {
		firstLoaded := messages[0]
		err = d.db.QueryRowContext(ctx,
			`SELECT EXISTS(
            SELECT 1 FROM messages
            WHERE chat_id = $1
              AND created_at < $2
              AND deleted_at IS NULL
        )`,
			chatID, firstLoaded.CreatedAt.Time,
		).Scan(&hasMoreTop)
		if err != nil {
			return nil, fmt.Errorf("failed to check hasMoreTop: %w", err)
		}
	}
	// Проверяем есть ли сообщения после последнего загруженного
	var hasMoreBottom bool
	if len(messages) > 0 {
		lastLoaded := messages[len(messages)-1]
		err = d.db.QueryRowContext(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM messages
				WHERE chat_id = $1
				  AND created_at > $2
				  AND deleted_at IS NULL
			)`,
			chatID, lastLoaded.CreatedAt.Time,
		).Scan(&hasMoreBottom)
		if err != nil {
			return nil, fmt.Errorf("failed to check hasMoreBottom: %w", err)
		}
	}

	return &models.UnreadResult{
		Messages:      messages,
		FirstUnreadID: firstUnreadID,
		TotalUnread:   totalUnread,
		HasMoreBottom: hasMoreBottom,
		HasMoreTop:    hasMoreTop,
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
