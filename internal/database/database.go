package database

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var DB (*sqlx.DB)

func InitDB() {
	var err error
	log.Println("Connecting to database...")

	deploymentEnv := os.Getenv("DEPLOYMENT_ENV")
	if deploymentEnv == "cloud" {
		RDSPassword := os.Getenv("RDS_PASSWORD")
		RDSHost := os.Getenv("RDS_HOST")
		DB, err = sqlx.Connect("postgres", fmt.Sprintf("user=postgres dbname=forum password=%s host=%s", RDSPassword, RDSHost))
	} else if deploymentEnv == "local" {
		DB, err = sqlx.Connect("postgres", "user=postgres sslmode=disable dbname=forum password=postgres host=db")
	} else {
		panic("invalid DEPLOYMENT_ENV environment variable")
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}
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
