package handlers

import "github.com/gofiber/fiber/v2"

func (h *Handlers) Welcome(c *fiber.Ctx) error {
	return c.Render("login", nil)

}

func (h *Handlers) Register(c *fiber.Ctx) error {
	return c.Render("registration", nil)

}
