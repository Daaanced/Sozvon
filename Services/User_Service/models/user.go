// User_Service\models\user.go
package models

type User struct {
	Login   string `json:"login"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Info    string `json:"info"`
	Picture string `json:"picture"`
}
