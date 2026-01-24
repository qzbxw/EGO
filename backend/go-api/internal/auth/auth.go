// Package auth provides services for user authentication, including
// password hashing, JWT generation, and validation.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
)

const (
	// accessTokenDuration defines the validity period for an access token.
	accessTokenDuration = 24 * time.Hour
	// refreshTokenDuration defines the validity period for a refresh token.
	refreshTokenDuration = 30 * 24 * time.Hour
	// bcryptCost is the cost factor for hashing passwords. A higher value is more secure
	// but also slower. 14 is a strong and recommended value.
	bcryptCost = 14
)

// AuthService provides methods for handling JWT-based authentication.
type AuthService struct {
	jwtSecret []byte
}

// GooglePayload holds the essential claims extracted from a Google ID token.
type GooglePayload struct {
	Email   string
	Subject string
}

// NewAuthService creates and returns a new AuthService instance.
// It requires a non-empty JWT secret key.
func NewAuthService(secret string) (*AuthService, error) {
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}
	return &AuthService{jwtSecret: []byte(secret)}, nil
}

// HashPassword generates a bcrypt hash from a given password string.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a plaintext password with a bcrypt hash.
// It returns true if the password matches the hash, and false otherwise.
// It safely handles cases where the hash pointer is nil.
func CheckPasswordHash(password string, hash *string) bool {
	if hash == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password))
	return err == nil
}

// CreateAccessToken generates a new JWT access token for a given user and role.
func (s *AuthService) CreateAccessToken(username, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  username,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(accessTokenDuration).Unix(),
		"role": role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// CreateRefreshToken generates a new JWT refresh token for a given user.
func (s *AuthService) CreateRefreshToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(refreshTokenDuration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateJWT parses and validates a JWT token string.
// If the token is valid, it returns the username (subject) stored within the token.
func (s *AuthService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure that the signing method is HMAC, as we expect.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if username, ok := claims["sub"].(string); ok {
			return username, nil
		}
	}

	return "", errors.New("invalid token")
}

// ValidateGoogleJWT validates a Google-issued ID token against a specific client ID (audience).
// If the token is valid, it extracts and returns the user's email and Google subject ID.
func (s *AuthService) ValidateGoogleJWT(googleToken, audience string) (*GooglePayload, error) {
	payload, err := idtoken.Validate(context.Background(), googleToken, audience)
	if err != nil {
		return nil, fmt.Errorf("google token validation failed: %w", err)
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return nil, errors.New("email claim is missing or empty in the Google token")
	}

	return &GooglePayload{
		Email:   email,
		Subject: payload.Subject,
	}, nil
}
