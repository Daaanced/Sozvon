//Chat_Service\db\forwarded.go

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Chat_Service/models"

	"github.com/google/uuid"
)

// SaveForwardedMessages пересылает сообщения в другой чат.
// Сохраняет snapshot оригинала (sender + text) и ссылки на вложения.
func (d *Database) SaveForwardedMessages(ctx context.Context, toChatID string, senderID int, originalIDs []string) ([]*models.Message, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	var result []*models.Message

	for _, origID := range originalIDs {
		msg, err := d.forwardOne(ctx, tx, toChatID, senderID, origID, now)
		if err != nil {
			return nil, err
		}
		result = append(result, msg)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return result, nil
}

// forwardOne пересылает одно сообщение внутри транзакции.
func (d *Database) forwardOne(ctx context.Context, tx *sql.Tx, toChatID string, senderID int, origID string, now time.Time) (*models.Message, error) {
	// Читаем оригинал
	var orig models.Message
	var fwdSenderID sql.NullInt64
	var fwdText, fwdOrigID sql.NullString

	err := d.db.QueryRowContext(ctx,
		`SELECT id, sender_id, text, forwarded_sender_id, forwarded_text, forwarded_from_message_id
		 FROM messages WHERE id = $1 AND deleted_at IS NULL`,
		origID,
	).Scan(&orig.ID, &orig.SenderID, &orig.Text, &fwdSenderID, &fwdText, &fwdOrigID)
	if err != nil {
		return nil, fmt.Errorf("original message not found: %w", err)
	}

	// Если оригинал сам пересланный — берём его snapshot, а не обёртку
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

	attIDs, err := d.resolveAttachmentIDs(ctx, origID)
	if err != nil {
		return nil, err
	}

	for _, attID := range attIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO forwarded_attachments (message_id, attachment_id) VALUES ($1, $2)`,
			newID, attID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to link forwarded attachment: %w", err)
		}
	}

	return &models.Message{
		ID:        newID,
		ChatID:    toChatID,
		SenderID:  senderID,
		CreatedAt: now,
		ForwardedFrom: &models.ForwardedMeta{
			OriginalMessageID: snapshotOrigID,
			SenderID:          snapshotSenderID,
			Text:              snapshotText,
		},
	}, nil
}

// resolveAttachmentIDs возвращает id вложений сообщения:
// сначала собственные, затем forwarded (если своих нет).
func (d *Database) resolveAttachmentIDs(ctx context.Context, messageID string) ([]string, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id FROM attachments WHERE message_id = $1`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if len(ids) > 0 {
		return ids, nil
	}

	// Оригинал сам пересланный — берём его forwarded_attachments
	fwdRows, err := d.db.QueryContext(ctx,
		`SELECT attachment_id FROM forwarded_attachments WHERE message_id = $1`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer fwdRows.Close()

	for fwdRows.Next() {
		var id string
		if err := fwdRows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}
