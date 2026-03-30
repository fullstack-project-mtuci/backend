package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/tokens"
)

const userContextKey = "currentUser"

// AuthMiddleware validates JWT access tokens and loads the current user.
type AuthMiddleware struct {
	users        *repositories.UserRepository
	tokenManager *tokens.Manager
}

// NewAuthMiddleware builds an auth middleware.
func NewAuthMiddleware(users *repositories.UserRepository, tokenManager *tokens.Manager) *AuthMiddleware {
	return &AuthMiddleware{
		users:        users,
		tokenManager: tokenManager,
	}
}

// Handle ensures that request contains valid JWT token.
func (m *AuthMiddleware) Handle(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization header")
	}

	claims, err := m.tokenManager.Parse(parts[1])
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	if err := tokens.EnsureType(claims, "access"); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "access token required")
	}

	subject, ok := claims["sub"].(string)
	if !ok || subject == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token payload")
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token payload")
	}

	user, err := m.users.FindByID(userContext(c), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, "user not found")
		}
		return err
	}

	c.Locals(userContextKey, user)
	return c.Next()
}

// RequireRoles checks that current user has one of allowed roles.
func RequireRoles(roles ...models.Role) fiber.Handler {
	roleSet := make(map[models.Role]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		if len(roleSet) == 0 {
			return c.Next()
		}

		if _, ok := roleSet[user.Role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}

		return c.Next()
	}
}

// GetUser returns the authenticated user from context.
func GetUser(c *fiber.Ctx) *models.User {
	if value := c.Locals(userContextKey); value != nil {
		if user, ok := value.(*models.User); ok {
			return user
		}
	}
	return nil
}

// ClaimsFromContext reads JWT claims from context (if stored externally).
func ClaimsFromContext(c *fiber.Ctx) jwt.MapClaims {
	if value := c.Locals("claims"); value != nil {
		if claims, ok := value.(jwt.MapClaims); ok {
			return claims
		}
	}
	return nil
}

func userContext(c *fiber.Ctx) context.Context {
	if ctx := c.UserContext(); ctx != nil {
		return ctx
	}
	return context.Background()
}
