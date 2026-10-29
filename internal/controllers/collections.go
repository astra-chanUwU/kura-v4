package controllers

import (
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

type Collections struct{ service *collections.Service }

func NewCollections(s *collections.Service) *Collections { return &Collections{service: s} }

func (h *Collections) RegisterRoutes(api fiber.Router) {
	api.Get("/me/favorites", h.favorites)
	api.Post("/me/favorites/:postId", h.favorite)
	api.Delete("/me/favorites/:postId", h.unfavorite)
	api.Get("/me/lists", h.myLists)
	api.Post("/lists", h.createList)
	api.Get("/lists/:id", h.getList)
	api.Patch("/lists/:id", h.updateList)
	api.Delete("/lists/:id", h.deleteList)
	api.Post("/lists/:id/items", h.addItem)
	api.Delete("/lists/:id/items/:postId", h.removeItem)
	api.Put("/lists/:id/reorder", h.reorder)
}

func (h *Collections) favorites(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.service.Favorites(c.Context(), user.ID, viewerSeesNSFW(c), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	favs := map[string]bool{}
	for _, p := range page.Posts {
		favs[p.ID] = true
	}
	return c.JSON(dto.SearchPage{Total: page.Total, Posts: postsDTO(page.Posts, favs)})
}

// visiblePosts hides NSFW posts from viewers who have not opted in.
func visiblePosts(c fiber.Ctx, posts []model.Post) []model.Post {
	if viewerSeesNSFW(c) {
		return posts
	}
	out := make([]model.Post, 0, len(posts))
	for _, p := range posts {
		if p.Rating != "nsfw" {
			out = append(out, p)
		}
	}
	return out
}

func (h *Collections) favorite(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.Favorite(c.Context(), user.ID, c.Params("postId")); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Collections) unfavorite(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.Unfavorite(c.Context(), user.ID, c.Params("postId")); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Collections) myLists(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	lists, err := h.service.MyLists(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(listsDTO(lists))
}

func (h *Collections) createList(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.ListInput
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	list, err := h.service.CreateList(c.Context(), user.ID, in.Title, in.Description, in.Visibility)
	if err != nil {
		return writeError(c, err)
	}
	list.OwnerUsername = user.Username
	return c.Status(201).JSON(listDTO(list))
}

func (h *Collections) getList(c fiber.Ctx) error {
	var viewerID *string
	if u := currentUser(c); u != nil {
		viewerID = &u.ID
	}
	limit, offset := queryPage(c)
	detail, err := h.service.GetList(c.Context(), viewerID, c.Params("id"), viewerSeesNSFW(c), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	favs := map[string]bool{}
	if u := currentUser(c); u != nil {
		if ids, err := h.service.FavoriteIDs(c.Context(), u.ID); err == nil {
			favs = ids
		}
	}
	return c.JSON(dto.ListDetail{List: listDTO(detail.List), Items: postsDTO(detail.Items, favs), Total: detail.Total})
}

func (h *Collections) updateList(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.ListInput
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	list, err := h.service.UpdateList(c.Context(), user.ID, c.Params("id"), in.Title, in.Description, in.Visibility)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(listDTO(list))
}

func (h *Collections) deleteList(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.DeleteList(c.Context(), user.ID, c.Params("id")); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

type postIDInput struct {
	PostID string `json:"post_id"`
}

func (h *Collections) addItem(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in postIDInput
	if err := c.Bind().Body(&in); err != nil || in.PostID == "" {
		return fiber.NewError(400, "Choose a post to add.")
	}
	if err := h.service.AddItem(c.Context(), user.ID, c.Params("id"), in.PostID); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Collections) removeItem(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.RemoveItem(c.Context(), user.ID, c.Params("id"), c.Params("postId")); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

func (h *Collections) reorder(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.ReorderInput
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	if in.PostID != "" {
		if err := h.service.MoveItem(c.Context(), user.ID, c.Params("id"), in.PostID, in.AfterID); err != nil {
			return writeError(c, err)
		}
		return c.SendStatus(204)
	}
	if err := h.service.Reorder(c.Context(), user.ID, c.Params("id"), in.PostIDs); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}
