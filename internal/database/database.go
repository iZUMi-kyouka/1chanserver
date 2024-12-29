package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
)

var DB (*sqlx.DB)

func InitDB() {
	var err error
	DB, err = sqlx.Connect("postgres", "user=izumikyouka001 dbname=forum sslmode=disable password=izumikyouka001 host=localhost")
	DB = DB.Unsafe()

	if err != nil {

		log.Fatal(err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Successfully connected to database.")
	}
}
