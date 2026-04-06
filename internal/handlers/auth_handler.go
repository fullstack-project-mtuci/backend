package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"backend/internal/dto"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repositories"
	"backend/internal/tokens"
)

// AuthHandler handles user registration and authentication.
type AuthHandler struct {
	users        *repositories.UserRepository
	tokenManager *tokens.Manager
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(users *repositories.UserRepository, tokenManager *tokens.Manager) *AuthHandler {
	return &AuthHandler{
		users:        users,
		tokenManager: tokenManager,
	}
}

// Register creates a new employee account.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var body dto.RegisterRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	body.FullName = strings.TrimSpace(body.FullName)

	if !strings.Contains(body.Email, "@") {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}

	if len(body.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	if body.FullName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "full name is required")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Email:        body.Email,
		PasswordHash: string(hashed),
		FullName:     body.FullName,
		Role:         models.RoleEmployee,
	}

	ctx := requestContext(c)
	if err := h.users.Create(ctx, user); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fiber.NewError(fiber.StatusBadRequest, "user with this email already exists")
		}
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": sanitizeUser(user),
	})
}

// Login authenticates user credentials and returns JWT tokens.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var body dto.LoginRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	email := strings.TrimSpace(strings.ToLower(body.Email))
	if email == "" || body.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	ctx := requestContext(c)
	user, err := h.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	return h.respondWithTokens(c, user)
}

// Refresh issues a new access token based on a valid refresh token.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var body dto.RefreshTokenRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	if body.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh token is required")
	}

	claims, err := h.tokenManager.Parse(body.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	if err := tokens.EnsureType(claims, "refresh"); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "refresh token required")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token payload")
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token payload")
	}

	ctx := requestContext(c)
	user, err := h.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, "user not found")
		}
		return err
	}

	return h.respondWithTokens(c, user)
}

// Me returns the authenticated user profile.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}

	return c.JSON(fiber.Map{
		"user": sanitizeUser(user),
	})
}

func (h *AuthHandler) respondWithTokens(c *fiber.Ctx, user *models.User) error {
	accessToken, err := h.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return err
	}

	refreshToken, err := h.tokenManager.GenerateRefreshToken(user)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"user":         sanitizeUser(user),
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expiresIn":    int(h.tokenManager.AccessTTL().Seconds()),
	})
}

func sanitizeUser(user *models.User) fiber.Map {
	return fiber.Map{
		"id":            user.ID,
		"email":         user.Email,
		"full_name":     user.FullName,
		"role":          user.Role,
		"department_id": user.DepartmentID,
		"manager_id":    user.ManagerID,
		"is_active":     user.IsActive,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	}
}
