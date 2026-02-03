// Chat_Service\models\chat_list_item.go
package models

type ChatListItem struct {
	ChatID      string   `json:"chatId"`
	Members     []string `json:"members"`
	LastMessage string   `json:"lastMessage"`
}
