// Chat_Service\models\message.go
package models

type Message struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}
