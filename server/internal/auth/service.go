/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid credentials")
)

// User represents a registered Kizuna Cloud user.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Service provides user authentication and session management.
type Service struct {
	mu        sync.RWMutex
	users     map[string]*User // keyed by lowercase email
	jwtSecret string
	tokenTTL  time.Duration
}

// NewService instantiates an AuthService.
func NewService(jwtSecret string, tokenTTL time.Duration) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &Service{
		users:     make(map[string]*User),
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

// Register registers a new user with a hashed password and returns an auth token.
func (s *Service) Register(username, email, password string) (*User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	if email == "" || username == "" || len(password) < 6 {
		return nil, "", fmt.Errorf("invalid registration parameters (password min 6 chars)")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[email]; exists {
		return nil, "", ErrUserAlreadyExists
	}

	salt := generateSalt(16)
	hash := hashPassword(password, salt)

	user := &User{
		ID:           fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Salt:         salt,
		CreatedAt:    time.Now().UTC(),
	}

	s.users[email] = user

	token, err := GenerateToken(user.ID, user.Username, user.Email, s.tokenTTL, s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

// Login verifies user credentials and returns a valid session JWT token.
func (s *Service) Login(email, password string) (*User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	s.mu.RLock()
	user, exists := s.users[email]
	s.mu.RUnlock()

	if !exists {
		return nil, "", ErrUserNotFound
	}

	expectedHash := hashPassword(password, user.Salt)
	if expectedHash != user.PasswordHash {
		return nil, "", ErrInvalidPassword
	}

	token, err := GenerateToken(user.ID, user.Username, user.Email, s.tokenTTL, s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

// ValidateSession verifies a JWT bearer token and returns the corresponding User.
func (s *Service) ValidateSession(tokenStr string) (*User, error) {
	claims, err := ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[claims.Email]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func hashPassword(password, salt string) string {
	combined := append([]byte(password), []byte(salt)...)
	hash := sha256.Sum256(combined)
	return hex.EncodeToString(hash[:])
}

func generateSalt(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
