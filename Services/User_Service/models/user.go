package models

type User struct {
	Login   string `json:"login"` // идентификатор, совпадает с Auth Service
	Name    string `json:"name"`
	Email   string `json:"email"`
	Age     int    `json:"age"`
	Address string `json:"address"`
}
