// Command seed loads demonstration base data into the database (SPEC §7.3).
//
//	go run ./cmd/seed
package main

import (
	"log"
	"os"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/database"
	"github.com/neu-software-practice/software-practice-backend/internal/seed"
)

func main() {
	dsn, err := config.DatabaseDSN()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	password := os.Getenv("SEED_DEFAULT_PASSWORD")
	if password == "" {
		password = "Passw0rd!"
	}

	db, err := database.NewMySQL(dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := seed.Run(db, password); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seed completed: 6 role accounts + base data ready")
}
