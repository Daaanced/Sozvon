package main

import (
	"log"
	"net/http"

	"dbserver/db"
	"dbserver/handlers"
)

func main() {
	//Подключается к базе данных PostgreSQL и сохраняет подключение в глобальную переменную db.DB.
	db.Init()

	//Когда клиент или Server A отправляет запрос на /users, вызывается функция handlers.GetUsers.
	http.HandleFunc("/users", handlers.GetUsers)
	//Регистрирует маршрут для создания пользователя.
	http.HandleFunc("/users/create", handlers.CreateUser)

	log.Println("Data service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
