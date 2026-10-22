package controllers

import (
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/gofiber/fiber/v3"
)

type Site struct{ service *site.Service }

func NewSite(service *site.Service) *Site          { return &Site{service: service} }
func (h *Site) RegisterRoutes(router fiber.Router) { router.Get("/site", h.get) }
func (h *Site) get(c fiber.Ctx) error {
	result, err := h.service.Get(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(dto.Site{Name: result.Name, Description: result.Description})
}
