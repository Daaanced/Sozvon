//Chat_Service\db\migrate.go

package db

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (d *Database) Migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Используем транзакцию для атомарности
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Создаем таблицы без внешних ключей (или с отложенной проверкой)
	queries := []string{
		// Сначала таблицы, которые ни на что не ссылаются
		`CREATE TABLE IF NOT EXISTS chats (
            id         UUID      PRIMARY KEY,
            active     BOOLEAN   NOT NULL DEFAULT FALSE,
            type       TEXT      NOT NULL DEFAULT 'direct',
            name       TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT NOW()
        );`,

		// messages до chat_members (так как chat_members ссылается на messages)
		`CREATE TABLE IF NOT EXISTS messages (
            id               UUID      PRIMARY KEY,
            chat_id          UUID      REFERENCES chats(id) ON DELETE CASCADE,
            sender_id        INTEGER   NOT NULL,
            text             TEXT      NOT NULL DEFAULT '',
            reply_to_id      UUID      REFERENCES messages(id) ON DELETE SET NULL,
            forwarded_sender_id        INTEGER,
            forwarded_text             TEXT,
            forwarded_from_message_id  UUID,
            edited_at        TIMESTAMP,
            deleted_at       TIMESTAMP,
            created_at       TIMESTAMP NOT NULL DEFAULT NOW()
        );`,

		// Теперь chat_members (messages уже существует)
		`CREATE TABLE IF NOT EXISTS chat_members (
            chat_id             UUID    REFERENCES chats(id) ON DELETE CASCADE,
            user_id             INTEGER NOT NULL,
            last_read_message_id UUID   REFERENCES messages(id) ON DELETE SET NULL,
            PRIMARY KEY (chat_id, user_id)
        );`,

		// attachments
		`CREATE TABLE IF NOT EXISTS attachments (
            id         UUID    PRIMARY KEY,
            message_id UUID    REFERENCES messages(id) ON DELETE CASCADE,
            file_name  TEXT    NOT NULL,
            store_name TEXT    NOT NULL,
            mime_type  TEXT    NOT NULL,
            size       BIGINT  NOT NULL,
            width      INTEGER,
            height     INTEGER
        );`,

		// forwarded_attachments
		`CREATE TABLE IF NOT EXISTS forwarded_attachments (
            message_id    UUID REFERENCES messages(id) ON DELETE CASCADE,
            attachment_id UUID REFERENCES attachments(id) ON DELETE CASCADE,
            PRIMARY KEY (message_id, attachment_id)
        );`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to execute query: %w\nQuery: %s", err, q)
		}
	}

	// 2. Создаем индексы
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created ON messages(chat_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_reply_to ON messages(reply_to_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_forwarded_from ON messages(forwarded_from_message_id);`,
		`CREATE INDEX IF NOT EXISTS idx_chats_active ON chats(active);`,
		`CREATE INDEX IF NOT EXISTS idx_chats_type ON chats(type);`,
		`CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_chat_members_last_read ON chat_members(last_read_message_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at);`,
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			// Индексы не критичны, логируем ошибку
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("✅ Database migration completed successfully")
	return nil
}
