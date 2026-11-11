package auth

import (
	"testing"
	"time"
)

func TestJWTTokenGenerationAndValidation(t *testing.T) {
	secret := "super-secure-secret-key-12345"

	t.Run("valid token", func(t *testing.T) {
		token, err := GenerateToken("user_123", "alice", "alice@kizuna.com", time.Hour, secret)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		claims, err := ValidateToken(token, secret)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if claims.UserID != "user_123" || claims.Username != "alice" || claims.Email != "alice@kizuna.com" {
			t.Errorf("claims mismatch: %+v", claims)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := GenerateToken("user_456", "bob", "bob@kizuna.com", -10*time.Minute, secret)
		if err != nil {
			t.Fatalf("failed to generate expired token: %v", err)
		}

		_, err = ValidateToken(token, secret)
		if err != ErrExpiredToken {
			t.Errorf("expected ErrExpiredToken, got %v", err)
		}
	})

	t.Run("tampered token signature", func(t *testing.T) {
		token, err := GenerateToken("user_789", "carol", "carol@kizuna.com", time.Hour, secret)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		wrongSecret := "another-secret"
		_, err = ValidateToken(token, wrongSecret)
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got %v", err)
		}
	})
}

func TestAuthService_RegistrationAndLogin(t *testing.T) {
	svc := NewService("test-secret", 2*time.Hour)

	// 1. Register new user
	user, token, err := svc.Register("musician_dan", "dan@band.io", "mypassword123")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if user.Username != "musician_dan" || user.Email != "dan@band.io" {
		t.Errorf("user mismatch: %+v", user)
	}
	if token == "" {
		t.Errorf("expected non-empty token")
	}

	// 2. Duplicate registration returns error
	_, _, err = svc.Register("dan_clone", "dan@band.io", "otherpassword")
	if err != ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// 3. Login with correct password
	loginUser, loginToken, err := svc.Login("dan@band.io", "mypassword123")
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}
	if loginUser.ID != user.ID || loginToken == "" {
		t.Errorf("login failed to return valid user/token")
	}

	// 4. Login with wrong password
	_, _, err = svc.Login("dan@band.io", "wrongpassword")
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}

	// 5. Validate session token
	sessionUser, err := svc.ValidateSession(loginToken)
	if err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}
	if sessionUser.Email != "dan@band.io" {
		t.Errorf("session user email mismatch: %s", sessionUser.Email)
	}
}
