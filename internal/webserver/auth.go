package webserver

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidPassword is returned when a login attempt fails due to wrong password.
var ErrInvalidPassword = errors.New("invalid password")

// HashPassword hashes the plaintext password using bcrypt.
func HashPassword(plaintext string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
}

// CheckPassword returns true if plaintext matches the bcrypt hash.
func CheckPassword(hash []byte, plaintext string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(plaintext)) == nil
}

// AuthManager handles dashboard password authentication and session cookie management.
type AuthManager struct {
	mu           sync.RWMutex
	passwordHash []byte
	sessions     map[string]time.Time
}

// NewAuthManager creates a new AuthManager with an empty sessions map.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		sessions: make(map[string]time.Time),
	}
}

// SetPassword hashes and stores the plaintext password.
func (am *AuthManager) SetPassword(plaintext string) error {
	hash, err := HashPassword(plaintext)
	if err != nil {
		return err
	}
	am.mu.Lock()
	am.passwordHash = hash
	am.mu.Unlock()
	return nil
}

// LoadPasswordHash loads a pre-existing bcrypt hash (e.g. from config file).
func (am *AuthManager) LoadPasswordHash(hash []byte) {
	am.mu.Lock()
	am.passwordHash = hash
	am.mu.Unlock()
}

// IsPasswordSet returns true if a password hash has been stored.
func (am *AuthManager) IsPasswordSet() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.passwordHash) > 0
}

// Login checks the password and, if correct, generates a secure random session cookie
// value, stores it, and returns it. Returns ErrInvalidPassword on wrong password.
func (am *AuthManager) Login(plaintext string) (cookieValue string, err error) {
	am.mu.RLock()
	hash := am.passwordHash
	am.mu.RUnlock()

	if !CheckPassword(hash, plaintext) {
		return "", ErrInvalidPassword
	}

	// Generate a 32-byte cryptographically random session cookie value.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(buf)

	am.mu.Lock()
	am.sessions[value] = time.Now()
	am.mu.Unlock()

	return value, nil
}

// IsAuthenticated returns true if the given cookie value corresponds to an active session.
func (am *AuthManager) IsAuthenticated(cookieValue string) bool {
	if cookieValue == "" {
		return false
	}
	am.mu.RLock()
	_, ok := am.sessions[cookieValue]
	am.mu.RUnlock()
	return ok
}

// MakeSessionCookie constructs an httpOnly, Secure, SameSite=Strict session cookie.
func (am *AuthManager) MakeSessionCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     "agenthub_session",
		Value:    value,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
}
