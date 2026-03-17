//Chat_Service\db\members.go

package db

import (
	"context"
	"fmt"
)

// GetChatMembers возвращает список user_id участников чата.
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

// IsMember проверяет, является ли пользователь участником чата.
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

// DeleteChatMembersByUserID удаляет все записи участника по user_id.
// Используется при удалении пользователя из системы.
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
