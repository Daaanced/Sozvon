// Chat_Service\storage\inbox.go
package storage

import "sync"

// user_login -> []chatID
var (
	UserInboxes = make(map[string][]string)
	InboxMu     sync.Mutex
)
