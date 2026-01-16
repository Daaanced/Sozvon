package db

import (
	"database/sql"
	"fmt"
	"log"

	"Authorization/models"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = ""       // поставь свой пароль
	dbname   = "authdb" // отдельная БД для Auth Service
)

func Init() {
	psqlInfo := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname,
	)

	var err error
	DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	log.Println("Connected to PostgreSQL Auth DB")
}

// CreateUser сохраняет нового пользователя с хешем пароля
func CreateUser(u models.User) error {
	_, err := DB.Exec(
		"INSERT INTO users (login, password) VALUES ($1, $2)",
		u.Login, u.Password,
	)
	return err
}

// GetUserByLogin возвращает пользователя по логину
func GetUserByLogin(login string) (models.User, error) {
	var u models.User
	err := DB.QueryRow(
		"SELECT login, password FROM users WHERE login=$1",
		login,
	).Scan(&u.Login, &u.Password)
	return u, err
}
