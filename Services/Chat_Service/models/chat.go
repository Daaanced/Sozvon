// Chat_Service\models\chat.go
package models

type Chat struct {
	ID      string   `json:"id"`
	Members []string `json:"members"`
	Active  bool     `json:"active"`
}
