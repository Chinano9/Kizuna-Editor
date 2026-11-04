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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid authentication token")
	ErrExpiredToken = errors.New("authentication token has expired")
)

// Claims represents standard JWT payload claims for Kizuna Cloud.
type Claims struct {
	UserID    string `json:"sub"`
	Username  string `json:"name"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// GenerateToken generates a signed JWT token with standard HMAC-SHA256.
func GenerateToken(userID, username, email string, duration time.Duration, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret cannot be empty")
	}

	h := header{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(h)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(duration).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	unsignedToken := headerB64 + "." + claimsB64
	sig := computeSignature(unsignedToken, secret)

	return unsignedToken + "." + sig, nil
}

// ValidateToken parses, validates signature, and verifies expiration of a JWT token.
func ValidateToken(tokenStr, secret string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsignedToken := parts[0] + "." + parts[1]
	expectedSig := computeSignature(unsignedToken, secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().UTC().Unix() > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func computeSignature(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
