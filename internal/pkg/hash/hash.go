// Package hash wraps bcrypt for password hashing (SPEC §7.2). Passwords are
// never stored or compared in plaintext.
package hash

import "golang.org/x/crypto/bcrypt"

// Password hashes a plaintext password with a per-call random salt.
func Password(plaintext string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify reports whether plaintext matches the previously hashed password.
func Verify(hashed, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext)) == nil
}
