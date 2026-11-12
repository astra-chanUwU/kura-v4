package controllers

import (
	"encoding/json"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/gofiber/fiber/v3"
)

// Admin serves staff endpoints: role management (admin+), the audit feed
// and per-post history with rollback (moderator+ to act, signed-in to view
// a post's history).
type Admin struct {
	mod *moderation.Service
}

func NewAdmin(mod *moderation.Service) *Admin { return &Admin{mod: mod} }

func (h *Admin) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/users", h.listUsers)
	api.Patch("/admin/users/:id/role", h.setRole)
	api.Get("/admin/history-actors", h.historyActors)
	api.Get("/admin/history", h.recentHistory)
	api.Get("/admin/hidden-posts", h.hiddenPosts)
	api.Get("/posts/:id/history", h.postHistory)
	api.Post("/posts/:id/hide", h.hide)
	api.Post("/posts/:id/unhide", h.unhide)
	api.Post("/posts/:id/delete", h.deletePost)
	api.Post("/admin/history/:id/rollback", h.rollback)
}

func historyDTO(e model.ChangeEntry) dto.ChangeEntry {
	decode := func(raw json.RawMessage) map[string]any {
		out := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &out)
		}
		return out
	}
	actorID := ""
	if e.ActorID != nil {
		actorID = *e.ActorID
	}
	thumbnailURL := ""
	if e.EntityType == "post" {
		thumbnailURL = "/media/posts/" + e.EntityID + "/thumb"
	}
	return dto.ChangeEntry{ID: e.ID, EntityType: e.EntityType, EntityID: e.EntityID, ThumbnailURL: thumbnailURL, ActorID: actorID, ActorUsername: e.ActorUsername,
		Action: e.Action, Before: decode(e.Before), After: decode(e.After), RollbackOf: e.RollbackOf,
		CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
}

func historiesDTO(entries []model.ChangeEntry) []dto.ChangeEntry {
	out := make([]dto.ChangeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, historyDTO(e))
	}
	return out
}

func (h *Admin) listUsers(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.mod.ListUserPage(c.Context(), *user, c.Query("q"), c.Query("role"), c.Query("sort"), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	users := make([]dto.User, 0, len(page.Users))
	for _, u := range page.Users {
		users = append(users, userDTO(u))
	}
	return c.JSON(dto.UserPage{
		Total: page.Total, Member: page.Member, Moderator: page.Moderator,
		Admin: page.Admin, SuperAdmin: page.SuperAdmin, Users: users,
	})
}

func (h *Admin) setRole(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.SetRole
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	updated, err := h.mod.SetRole(c.Context(), *user, c.Params("id"), model.Role(in.Role))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(userDTO(updated))
}

func (h *Admin) recentHistory(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.mod.ListHistory(c.Context(), *user, c.Query("actor_id"), c.Query("q"), c.Query("action"), c.Query("entity"), c.Query("sort"), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.HistoryPage{Total: page.Total, Entries: historiesDTO(page.Entries)})
}

func (h *Admin) historyActors(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	actors, err := h.mod.SearchHistoryActors(c.Context(), *user, c.Query("q"))
	if err != nil {
		return writeError(c, err)
	}
	out := make([]dto.HistoryActor, 0, len(actors))
	for _, actor := range actors {
		out = append(out, dto.HistoryActor{ID: actor.ID, Username: actor.Username})
	}
	return c.JSON(out)
}

func (h *Admin) postHistory(c fiber.Ctx) error {
	if _, err := requireUser(c); err != nil {
		return err
	}
	entries, err := h.mod.HistoryForPost(c.Context(), c.Params("id"), 100)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(historiesDTO(entries))
}

func (h *Admin) hide(c fiber.Ctx) error   { return h.setHidden(c, true, false) }
func (h *Admin) unhide(c fiber.Ctx) error { return h.setHidden(c, false, false) }
func (h *Admin) deletePost(c fiber.Ctx) error {
	return h.setHidden(c, true, true)
}

func (h *Admin) setHidden(c fiber.Ctx, hidden, deleted bool) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	post, err := h.mod.SetPostHidden(c.Context(), *user, c.Params("id"), hidden, deleted)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(postDTO(post, map[string]bool{}))
}

func (h *Admin) rollback(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	entry, err := h.mod.Rollback(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(historyDTO(entry))
}

func (h *Admin) hiddenPosts(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.mod.ListHiddenPosts(c.Context(), *user, c.Query("kind"), c.Query("q"), c.Query("rating"), c.Query("sort"), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.HiddenPage{
		Total: page.Total, Hidden: page.Hidden, Deleted: page.Deleted,
		Posts: postsDTO(page.Posts, map[string]bool{}),
	})
}
