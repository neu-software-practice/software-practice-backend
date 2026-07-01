package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAccessToken(t *testing.T) {
	secret := "test-secret-key-at-least-32-bytes-long!!"
	token, err := GenerateAccessToken("user1", "p001", "13800138000", secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	// Verify the token can be parsed
	claims, err := ParseJWT(token, secret)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims["sub"] != "user1" {
		t.Errorf("sub = %v, want user1", claims["sub"])
	}
	if claims["patientId"] != "p001" {
		t.Errorf("patientId = %v, want p001", claims["patientId"])
	}
	if claims["phone"] != "13800138000" {
		t.Errorf("phone = %v, want 13800138000", claims["phone"])
	}
}

func TestGenerateAdminAccessToken(t *testing.T) {
	secret := "test-secret-key-at-least-32-bytes-long!!"
	token, err := GenerateAdminAccessToken("admin1", "super_admin", secret, 900)
	if err != nil {
		t.Fatalf("GenerateAdminAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := ParseJWT(token, secret)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims["sub"] != "admin1" {
		t.Errorf("sub = %v, want admin1", claims["sub"])
	}
	if claims["role"] != "super_admin" {
		t.Errorf("role = %v, want super_admin", claims["role"])
	}
}

func TestParseJWT_InvalidToken(t *testing.T) {
	secret := "test-secret-key-at-least-32-bytes-long!!"

	t.Run("empty token", func(t *testing.T) {
		_, err := ParseJWT("", secret)
		if err == nil {
			t.Error("expected error for empty token")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := ParseJWT("not.a.jwt", secret)
		if err == nil {
			t.Error("expected error for malformed token")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, _ := GenerateAccessToken("u1", "p1", "", secret)
		_, err := ParseJWT(token, "wrong-secret-key-at-least-32-bytes!!")
		if err == nil {
			t.Error("expected error for wrong secret")
		}
	})

	t.Run("wrong signing method", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "test"})
		ts, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		_, err := ParseJWT(ts, secret)
		if err == nil {
			t.Error("expected error for none signing method")
		}
	})
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-at-least-32-bytes-long!!"
	// Create an already-expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u1",
		"exp": -1, // definitely in the past
	})
	expiredToken, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = ParseJWT(expiredToken, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
