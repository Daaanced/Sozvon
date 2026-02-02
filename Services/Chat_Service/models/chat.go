package models

type Chat struct {
	ID      string   `json:"id"`
	Members []string `json:"members"`
}
