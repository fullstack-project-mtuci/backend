package dto

// RegisterRequest describes incoming payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// LoginRequest describes login payload.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest describes refresh token payload.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
