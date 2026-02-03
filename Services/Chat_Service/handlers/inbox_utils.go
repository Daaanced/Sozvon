// Chat_Service\handlers\inbox_utils.go
package handlers

import "Chat_Service/storage"

func inboxHasChat(user, chatID string) bool {
	storage.InboxMu.Lock()
	defer storage.InboxMu.Unlock()

	for _, id := range storage.UserInboxes[user] {
		if id == chatID {
			return true
		}
	}
	return false
}

func addChatToInbox(user, chatID string) {
	storage.InboxMu.Lock()
	defer storage.InboxMu.Unlock()

	storage.UserInboxes[user] = append(storage.UserInboxes[user], chatID)
}
