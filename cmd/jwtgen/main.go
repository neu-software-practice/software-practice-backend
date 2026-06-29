// Command jwtgen generates a JWT token for smoke tests.
// Usage: go run ./cmd/jwtgen <patient_id>
// Reads JWT_SECRET from environment.
package main

import (
	"fmt"
	"os"

	"github.com/neuhis/software-practice-backend/internal/middleware"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: jwtgen <patient_id>\n")
		os.Exit(1)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Fprintf(os.Stderr, "JWT_SECRET environment variable is required\n")
		os.Exit(1)
	}

	token, err := middleware.GenerateToken(os.Args[1], secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(token)
}
