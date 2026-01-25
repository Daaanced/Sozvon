// User_Service\db\db.go
package db

import (
	"database/sql"
	"fmt"
	"log"

	"User_Service/models"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = ""       // пароль к PostgreSQL
	dbname   = "userdb" // отдельная БД для User Service
)

const BackendURL = "http://localhost:8080"
const AvatarsPath = "/static/avatars/"
const DefaultAvatar = "default.png"

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

	log.Println("Connected to PostgreSQL User DB")
}

// CRUD операции

func GetAllUsers() ([]models.User, error) {
	rows, err := DB.Query("SELECT login, name, email, info, picture FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.Login, &u.Name, &u.Email, &u.Info, &u.Picture); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func GetUserByLogin(login string) (models.User, error) {
	var u models.User

	err := DB.QueryRow(
		"SELECT login, name, email, info, picture FROM users WHERE login=$1",
		login,
	).Scan(&u.Login, &u.Name, &u.Email, &u.Info, &u.Picture)

	if err != nil {
		return u, err
	}

	fillAvatar(&u)

	return u, nil
}

func CreateUser(u models.User) error {
	_, err := DB.Exec(
		"INSERT INTO users (login, name, email, info, picture) VALUES ($1,$2,$3,$4,$5)",
		u.Login, u.Name, u.Email, u.Info, u.Picture,
	)
	return err
}

func DeleteUser(login string) error {
	_, err := DB.Exec("DELETE FROM users WHERE login=$1", login)
	return err
}

func UpdateUser(u models.User) error {
	_, err := DB.Exec(
		"UPDATE users SET name=$1, email=$2, info=$3, picture=$4 WHERE login=$5",
		u.Name, u.Email, u.Info, u.Picture, u.Login,
	)
	return err
}

func fillAvatar(u *models.User) {

	if u.Picture == "" {
		u.Picture = BackendURL + AvatarsPath + DefaultAvatar
	} else {
		u.Picture = BackendURL + AvatarsPath + u.Picture
	}
}
