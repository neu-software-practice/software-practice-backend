// Command migrate applies or reverts the embedded database migrations.
//
//	go run ./cmd/migrate -dir up
//	go run ./cmd/migrate -dir down
package main

import (
	"flag"
	"log"

	"github.com/neu-software-practice/software-practice-backend/internal/config"
	"github.com/neu-software-practice/software-practice-backend/internal/migrate"
)

func main() {
	dir := flag.String("dir", "up", "migration direction: up | down")
	flag.Parse()

	dsn, err := config.DatabaseDSN()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	switch *dir {
	case "up":
		if err := migrate.Up(dsn); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		log.Println("migrations applied")
	case "down":
		if err := migrate.Down(dsn); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("migrations reverted")
	default:
		log.Fatalf("unknown -dir %q (use up|down)", *dir)
	}
}
