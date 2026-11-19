package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/commentsafety"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

// CommentSafety serves the moderator queue of blocked and flagged comments.
type CommentSafety struct{ service *commentsafety.Service }

func NewCommentSafety(s *commentsafety.Service) *CommentSafety {
	return &CommentSafety{service: s}
}

func (h *CommentSafety) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/comment-safety", h.list)
	api.Get("/admin/comment-safety/resolvers", h.resolvers)
	api.Post("/admin/comment-safety/:id/dismiss", h.dismiss)
	api.Post("/admin/comment-safety/:id/remove", h.remove)
	api.Post("/admin/comment-safety/:id/reopen", h.reopen)
}

func (h *CommentSafety) list(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	filter := model.CommentSafetyFilter{
		Status: c.Query("status"), Action: c.Query("action"), Category: c.Query("category"),
		Query: c.Query("q"), Sort: c.Query("sort"),
	}
	if filter.Status == "" {
		filter.Status = "open"
	}
	if c.Query("resolver") == "me" {
		filter.ResolverID = user.ID
	} else {
		filter.ResolverID = c.Query("resolver")
	}
	page, err := h.service.List(c.Context(), *user, filter, limit, offset)
	if err != nil {
		return writeCommentSafetyError(c, err)
	}
	out := make([]dto.CommentSafetyEvent, 0, len(page.Events))
	for _, event := range page.Events {
		out = append(out, commentSafetyDTO(event))
	}
	return c.JSON(dto.CommentSafetyPage{Total: page.Total, Events: out})
}

func (h *CommentSafety) resolvers(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	people, err := h.service.Resolvers(c.Context(), *user)
	if err != nil {
		return writeCommentSafetyError(c, err)
	}
	out := make([]dto.CommentSafetyResolver, 0, len(people))
	for _, person := range people {
		out = append(out, commentSafetyResolverDTO(person))
	}
	return c.JSON(dto.CommentSafetyResolvers{Resolvers: out})
}

func (h *CommentSafety) dismiss(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	event, err := h.service.Dismiss(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeCommentSafetyError(c, err)
	}
	return c.JSON(commentSafetyDTO(event))
}

func (h *CommentSafety) remove(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	event, err := h.service.Remove(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeCommentSafetyError(c, err)
	}
	return c.JSON(commentSafetyDTO(event))
}

func (h *CommentSafety) reopen(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	event, err := h.service.Reopen(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeCommentSafetyError(c, err)
	}
	return c.JSON(commentSafetyDTO(event))
}

func commentSafetyDTO(event model.CommentSafetyEvent) dto.CommentSafetyEvent {
	categories := event.Categories
	if categories == nil {
		categories = []string{}
	}
	out := dto.CommentSafetyEvent{
		ID: event.ID, CommentID: event.CommentID, PostID: event.PostID,
		AuthorUsername: event.AuthorUsername, Action: event.Action, Categories: categories,
		Body: event.Body, Status: event.Status, CreatedAt: event.CreatedAt.UTC().Format(reportTime),
		ResolverID: event.ResolverID, ResolverUsername: event.ResolverUsername,
	}
	if !event.ResolvedAt.IsZero() {
		out.ResolvedAt = event.ResolvedAt.UTC().Format(reportTime)
	}
	out.Decisions = commentSafetyDecisionsDTO(event.Decisions)
	return out
}

func commentSafetyDecisionsDTO(steps []model.CommentSafetyDecision) []dto.CommentSafetyDecision {
	out := make([]dto.CommentSafetyDecision, 0, len(steps))
	for _, step := range steps {
		out = append(out, dto.CommentSafetyDecision{
			ActorID: step.ActorID, ActorUsername: step.ActorUsername, Action: step.Action,
			CreatedAt: step.CreatedAt.UTC().Format(reportTime),
		})
	}
	return out
}

func commentSafetyResolverDTO(person model.CommentSafetyResolver) dto.CommentSafetyResolver {
	categories := person.LastCategories
	if categories == nil {
		categories = []string{}
	}
	out := dto.CommentSafetyResolver{
		ID: person.ID, Username: person.Username, Decisions: person.Decisions,
		LastStatus: person.LastStatus, LastAction: person.LastAction,
		LastCategories: categories, LastBody: person.LastBody,
	}
	if !person.LastAt.IsZero() {
		out.LastAt = person.LastAt.UTC().Format(reportTime)
	}
	return out
}

func writeCommentSafetyError(c fiber.Ctx, err error) error {
	var invalid *commentsafety.ValidationError
	switch {
	case errors.As(err, &invalid):
		return fiber.NewError(400, invalid.Message)
	case errors.Is(err, commentsafety.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	case errors.Is(err, model.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	default:
		return writeError(c, err)
	}
}
