package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
)

var DB (*sqlx.DB)

func InitDB() {
	var err error
	DB, err = sqlx.Connect("postgres", "user=postgres sslmode=prefer dbname=forum password=IzumiKyouka071534. host=onechandb.chckaa4wm4hz.ap-southeast-1.rds.amazonaws.com")
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
