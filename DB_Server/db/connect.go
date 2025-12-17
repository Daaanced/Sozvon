package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() {
	const (
		host     = "localhost"
		port     = 5432
		user     = "postgres"
		password = ""
		dbname   = "appdb"
	)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	log.Println("Connected to PostgreSQL!")
}
