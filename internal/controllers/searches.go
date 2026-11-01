package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/searches"
	"github.com/gofiber/fiber/v3"
)

// Searches serves an account's private saved board filters.
type Searches struct{ service *searches.Service }

func NewSearches(s *searches.Service) *Searches { return &Searches{service: s} }

func (h *Searches) RegisterRoutes(api fiber.Router) {
	api.Get("/me/searches", h.list)
	api.Post("/me/searches", h.create)
	api.Delete("/me/searches/:id", h.delete)
}

func (h *Searches) list(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	items, err := h.service.List(c.Context(), user.ID)
	if err != nil {
		return searchError(c, err)
	}
	out := make([]dto.SavedSearch, 0, len(items))
	for _, item := range items {
		out = append(out, savedSearchDTO(item))
	}
	return c.JSON(out)
}

func (h *Searches) create(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreateSavedSearch
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	created, err := h.service.Create(c.Context(), user.ID, model.SavedSearch{
		Name:        in.Name,
		IncludeTags: in.IncludeTags,
		ExcludeTags: in.ExcludeTags,
		Query:       in.Q,
		Sort:        in.Sort,
	})
	if err != nil {
		return searchError(c, err)
	}
	return c.Status(201).JSON(savedSearchDTO(created))
}

func (h *Searches) delete(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.Delete(c.Context(), user.ID, c.Params("id")); err != nil {
		return searchError(c, err)
	}
	return c.SendStatus(204)
}

func searchError(c fiber.Ctx, err error) error {
	var vErr *searches.ValidationError
	if errors.As(err, &vErr) {
		return fiber.NewError(400, vErr.Message)
	}
	if errors.Is(err, searches.ErrNotFound) {
		return fiber.NewError(404, "Not found.")
	}
	return writeError(c, err)
}

func savedSearchDTO(s model.SavedSearch) dto.SavedSearch {
	return dto.SavedSearch{
		ID:          s.ID,
		Name:        s.Name,
		IncludeTags: tagsOrEmpty(s.IncludeTags),
		ExcludeTags: tagsOrEmpty(s.ExcludeTags),
		Q:           s.Query,
		Sort:        sortOrNewest(s.Sort),
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func sortOrNewest(sort string) string {
	if sort == "" {
		return "newest"
	}
	return sort
}
