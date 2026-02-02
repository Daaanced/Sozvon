package storage

import (
	"Chat_Service/models"
	"sync"
)

var (
	Chats = make(map[string]models.Chat)
	Mu    sync.Mutex
)
