package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/tags"
	"github.com/gofiber/fiber/v3"
)

const tagTime = "2006-01-02T15:04:05Z07:00"

// Tags serves tag wiki pages and the aliases that point at them.
// Suggestions stay on GET /tags.
type Tags struct{ service *tags.Service }

func NewTags(s *tags.Service) *Tags { return &Tags{service: s} }

func (h *Tags) RegisterRoutes(api fiber.Router) {
	api.Get("/tag/:name", h.get)
	api.Put("/tag/:name", h.update)
	api.Post("/tag/:name/aliases", h.addAlias)
	api.Delete("/tag/:name/aliases/:alias", h.removeAlias)
	api.Post("/tag/:name/implications", h.addImplication)
	api.Delete("/tag/:name/implications/:consequent", h.removeImplication)
}

func (h *Tags) get(c fiber.Ctx) error {
	page, err := h.service.Get(c.Context(), currentUser(c), c.Params("name"))
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(tagPageDTO(page))
}

func (h *Tags) update(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdateTagPage
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "That tag page could not be read.")
	}
	page, err := h.service.Update(c.Context(), *user, c.Params("name"), in.Category, in.Description)
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(tagPageDTO(page))
}

func (h *Tags) addAlias(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreateTagAlias
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "That alias could not be read.")
	}
	page, err := h.service.AddAlias(c.Context(), *user, c.Params("name"), in.Alias)
	if err != nil {
		return writeTagError(c, err)
	}
	return c.Status(201).JSON(tagPageDTO(page))
}

func (h *Tags) removeAlias(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	page, err := h.service.RemoveAlias(c.Context(), *user, c.Params("name"), c.Params("alias"))
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(tagPageDTO(page))
}

func (h *Tags) addImplication(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreateTagImplication
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "That implication could not be read.")
	}
	page, err := h.service.AddImplication(c.Context(), *user, c.Params("name"), in.Consequent)
	if err != nil {
		return writeTagError(c, err)
	}
	return c.Status(201).JSON(tagPageDTO(page))
}

func (h *Tags) removeImplication(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	page, err := h.service.RemoveImplication(c.Context(), *user, c.Params("name"), c.Params("consequent"))
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(tagPageDTO(page))
}

func writeTagError(c fiber.Ctx, err error) error {
	var vErr *tags.ValidationError
	switch {
	case errors.As(err, &vErr):
		return fiber.NewError(400, vErr.Message)
	case errors.Is(err, tags.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	case errors.Is(err, tags.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	default:
		return writeError(c, err)
	}
}

func tagPageDTO(page model.TagPage) dto.TagPage {
	aliases := page.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	implies := page.Implies
	if implies == nil {
		implies = []string{}
	}
	impliedBy := page.ImpliedBy
	if impliedBy == nil {
		impliedBy = []string{}
	}
	updated := ""
	if !page.UpdatedAt.IsZero() {
		updated = page.UpdatedAt.Format(tagTime)
	}
	return dto.TagPage{
		Tag: page.Tag, AliasOf: page.AliasOf, Aliases: aliases, Implies: implies, ImpliedBy: impliedBy, Category: page.Category,
		Description: page.Description, PostCount: page.PostCount,
		UpdatedByUsername: page.UpdatedByUsername, UpdatedAt: updated,
		Posts: postsDTO(page.Posts, map[string]bool{}),
	}
}
