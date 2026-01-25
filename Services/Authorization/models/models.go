// Authorization\models\models.go
package models

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"` // в БД будет храниться hash
}
