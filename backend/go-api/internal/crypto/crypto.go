// Package crypto provides helper functions for symmetric encryption
// and decryption using AES-GCM.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// deriveKey generates a valid AES key from a given string.
// It first attempts to decode the keyString as a hex string. If the resulting
// byte slice has a valid AES key length (16, 24, or 32 bytes), it is used directly.
// Otherwise, it falls back to using the SHA-256 hash of the keyString as a 32-byte key.
func deriveKey(keyString string) ([]byte, error) {
	if decoded, err := hex.DecodeString(keyString); err == nil {
		switch len(decoded) {
		case 16, 24, 32:
			return decoded, nil
		}
	}

	// Fallback: if not a valid hex key, derive a 32-byte key from the string.
	hash := sha256.Sum256([]byte(keyString))
	return hash[:], nil
}

// Encrypt encrypts a string using AES-GCM with a given key string.
// The output is a hex-encoded string containing the nonce and the ciphertext.
func Encrypt(stringToEncrypt string, keyString string) (string, error) {
	key, err := deriveKey(keyString)
	if err != nil {
		// This error is currently unreachable as deriveKey always returns a nil error.
		return "", fmt.Errorf("failed to derive key: %w", err)
	}
	plaintext := []byte(stringToEncrypt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	// A nonce is generated randomly for each encryption.
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal encrypts and authenticates the plaintext, prepending the nonce to the ciphertext.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded string that was encrypted using AES-GCM.
// It expects the input string to contain the nonce followed by the ciphertext.
func Decrypt(encryptedString string, keyString string) (string, error) {
	key, err := deriveKey(keyString)
	if err != nil {
		// This error is currently unreachable.
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	enc, err := hex.DecodeString(encryptedString)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(enc) < nonceSize {
		return "", errors.New("ciphertext is too short")
	}

	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	// Open decrypts and authenticates the ciphertext.
	// An error here often means the key is incorrect or the data is corrupted.
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	return string(plaintext), nil
}
