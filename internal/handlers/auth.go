package handlers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"SelektorDisc/internal/auth"
	"SelektorDisc/internal/domain/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	authCookieName          = "auth_token"
	authCookieTTL           = 7 * 24 * time.Hour
	postgresUniqueViolation = "23505"
	minPasswordLength       = 8
)

type loginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type registerRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func (h *Handlers) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username and password are required"})
	}

	user, err := h.usersRepo.GetByUsername(c.Context(), req.Username)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	if !auth.ComparePassword(user.PasswordHash, req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, err := auth.SignToken(user.Id.String(), authTokenSecret(), authCookieTTL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot create session"})
	}

	setAuthCookie(c, token)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handlers) Signup(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username and password are required"})
	}
	if len(req.Password) < minPasswordLength {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("password must be at least %d characters", minPasswordLength),
		})
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot hash password"})
	}

	user := &models.User{
		Id:           uuid.New(),
		Username:     req.Username,
		PasswordHash: passwordHash,
	}
	if _, err := h.usersRepo.CreateUser(c.Context(), user); err != nil {
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "username already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot register user"})
	}

	token, err := auth.SignToken(user.Id.String(), authTokenSecret(), authCookieTTL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "registered, but cannot create session"})
	}

	setAuthCookie(c, token)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handlers) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: "Lax",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

func setAuthCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: "Lax",
		Expires:  time.Now().Add(authCookieTTL),
	})
}

func isUniqueViolation(err error) bool {
	pgErr, ok := err.(*pgconn.PgError)
	return ok && pgErr.Code == postgresUniqueViolation
}

func authTokenSecret() []byte {
	return []byte(strings.TrimSpace(os.Getenv("AUTH_TOKEN_SECRET")))
}
