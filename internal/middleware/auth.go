package middleware

import (
	"strings"

	"SelektorDisc/internal/auth"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultTokenCookieName = "auth_token"
)

type AuthMiddleware struct {
	secret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	trimmed := strings.TrimSpace(secret)
	return &AuthMiddleware{secret: []byte(trimmed)}
}

func (m *AuthMiddleware) RequireAuth(c *fiber.Ctx) error {
	token := extractToken(c)
	if token == "" {
		return unauthorized(c)
	}

	userID, err := auth.VerifyToken(token, m.secret)
	if err != nil || userID == "" {
		return unauthorized(c)
	}

	c.Locals("user_uuid", userID)
	return c.Next()
}

func (m *AuthMiddleware) Close() error {
	return nil
}

func extractToken(c *fiber.Ctx) string {
	authorization := strings.TrimSpace(c.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		token := strings.TrimSpace(authorization[len("Bearer "):])
		if token != "" {
			return token
		}
	}

	if token := strings.TrimSpace(c.Cookies(defaultTokenCookieName)); token != "" {
		return token
	}

	if token := strings.TrimSpace(c.Query("token")); token != "" {
		return token
	}

	return ""
}

func unauthorized(c *fiber.Ctx) error {
	if strings.EqualFold(c.Get("Upgrade"), "websocket") {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}
	return c.Redirect("/")
}
