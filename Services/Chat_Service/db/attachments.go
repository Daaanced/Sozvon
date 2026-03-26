// Chat_Service/db/attachments.go

package db

import (
	"context"
	"fmt"

	"Chat_Service/models"

	"github.com/google/uuid"
)

// SaveAttachment сохраняет запись о вложении
func (d *Database) SaveAttachment(ctx context.Context, a models.Attachment) error {
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO attachments 
	(id, message_id, file_name, store_name, mime_type, size, width, height)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.ID, a.MessageID, a.FileName, a.StoreName, a.MimeType, a.Size, a.Width, a.Height,
	)
	if err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}
	return nil
}

// GetAttachmentsByMessageIDs возвращает вложения для набора сообщений
func (d *Database) GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []string) (map[string][]models.Attachment, error) {
	if len(messageIDs) == 0 {
		return map[string][]models.Attachment{}, nil
	}

	// Формируем $1,$2,$3...
	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, message_id, file_name, store_name, mime_type, size, width, height
		 FROM attachments WHERE message_id IN (%s)`,
		joinStrings(placeholders),
	)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.Attachment)
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.FileName,
			&a.StoreName, &a.MimeType, &a.Size,
			&a.Width, &a.Height,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		result[a.MessageID] = append(result[a.MessageID], a)
	}

	return result, nil
}

// После основного запроса attachments добавить:
func (d *Database) GetForwardedAttachmentsByMessageIDs(ctx context.Context, messageIDs []string) (map[string][]models.Attachment, error) {
	if len(messageIDs) == 0 {
		return map[string][]models.Attachment{}, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT fa.message_id, a.id, a.file_name, a.store_name, a.mime_type, a.size, a.width, a.height
		 FROM forwarded_attachments fa
		 JOIN attachments a ON a.id = fa.attachment_id
		 WHERE fa.message_id IN (%s)`,
		joinStrings(placeholders),
	)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.Attachment)
	for rows.Next() {
		var msgID string
		var a models.Attachment
		if err := rows.Scan(
			&msgID, // fa.message_id
			&a.ID,  // a.id
			&a.FileName,
			&a.StoreName, &a.MimeType, &a.Size,
			&a.Width, &a.Height,
		); err != nil {
			return nil, err
		}
		result[msgID] = append(result[msgID], a)
	}
	return result, nil
}

// GetAttachmentByID возвращает вложение по ID
func (d *Database) GetAttachmentByID(ctx context.Context, attachmentID string) (*models.Attachment, error) {
	var a models.Attachment
	err := d.db.QueryRowContext(ctx,
		`SELECT id, message_id, file_name, store_name, mime_type, size
		 FROM attachments WHERE id = $1`,
		attachmentID,
	).Scan(&a.ID, &a.MessageID, &a.FileName, &a.StoreName, &a.MimeType, &a.Size)
	if err != nil {
		return nil, fmt.Errorf("attachment not found: %w", err)
	}
	return &a, nil
}

// DeleteAttachmentsByMessageID удаляет все вложения сообщения, возвращает store_name для удаления файлов
func (d *Database) DeleteAttachmentsByMessageID(ctx context.Context, messageID string) ([]string, error) {
	rows, err := d.db.QueryContext(ctx,
		`DELETE FROM attachments WHERE message_id = $1 RETURNING store_name`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to delete attachments: %w", err)
	}
	defer rows.Close()

	var storeNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		storeNames = append(storeNames, name)
	}
	return storeNames, nil
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}

// NewAttachmentID генерирует ID вложения
func NewAttachmentID() string {
	return uuid.NewString()
}
