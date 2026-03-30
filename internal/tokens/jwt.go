package tokens

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"backend/internal/models"
)

// Manager generates and validates JWT tokens.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager builds a token manager with shared secret and TTL values.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// GenerateAccessToken returns a signed access token for the given user.
func (m *Manager) GenerateAccessToken(user *models.User) (string, error) {
	return m.generateToken(user, m.accessTTL, "access")
}

// GenerateRefreshToken returns a signed refresh token for the given user.
func (m *Manager) GenerateRefreshToken(user *models.User) (string, error) {
	return m.generateToken(user, m.refreshTTL, "refresh")
}

// AccessTTL returns configured access token TTL.
func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

// RefreshTTL returns configured refresh token TTL.
func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *Manager) generateToken(user *models.User, ttl time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":        user.ID.String(),
		"email":      user.Email,
		"role":       string(user.Role),
		"full_name":  user.FullName,
		"token_type": tokenType,
		"iat":        now.Unix(),
		"exp":        now.Add(ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse validates the token signature and returns claims if token is valid.
func (m *Manager) Parse(tokenStr string) (jwt.MapClaims, error) {
	parsed, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// EnsureType verifies that the claim matches the expected token type.
func EnsureType(claims jwt.MapClaims, expected string) error {
	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != expected {
		return errors.New("invalid token type")
	}
	return nil
}
