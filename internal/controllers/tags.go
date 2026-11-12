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
	api.Post("/tag/:name/suggestions", h.suggest)
	api.Get("/tag-suggestions", h.listSuggestions)
	api.Post("/tag-suggestions/:id/accept", h.acceptSuggestion)
	api.Post("/tag-suggestions/:id/reject", h.rejectSuggestion)
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

func (h *Tags) suggest(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreateTagSuggestion
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "That suggestion could not be read.")
	}
	item, err := h.service.Suggest(c.Context(), *user, c.Params("name"), in.Kind, in.Other, in.Reason)
	if err != nil {
		return writeTagError(c, err)
	}
	return c.Status(201).JSON(suggestionDTO(item))
}

func (h *Tags) listSuggestions(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	items, err := h.service.ListSuggestions(c.Context(), *user, c.Query("status"))
	if err != nil {
		return writeTagError(c, err)
	}
	out := make([]dto.TagSuggestion, 0, len(items))
	for _, item := range items {
		out = append(out, suggestionDTO(item))
	}
	return c.JSON(out)
}

func (h *Tags) acceptSuggestion(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	item, err := h.service.AcceptSuggestion(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(suggestionDTO(item))
}

func (h *Tags) rejectSuggestion(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	item, err := h.service.RejectSuggestion(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeTagError(c, err)
	}
	return c.JSON(suggestionDTO(item))
}

func suggestionDTO(item model.TagSuggestion) dto.TagSuggestion {
	return dto.TagSuggestion{
		ID: item.ID, Kind: item.Kind, SourceTag: item.SourceTag, TargetTag: item.TargetTag,
		Reason: item.Reason, Status: item.Status, AuthorUsername: item.AuthorUsername,
		CreatedAt: item.CreatedAt.UTC().Format(tagTime),
	}
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
