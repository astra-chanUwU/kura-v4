package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/comments"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

const commentTime = "2006-01-02T15:04:05Z07:00"

type Comments struct{ service *comments.Service }

func NewComments(s *comments.Service) *Comments { return &Comments{service: s} }

func (h *Comments) RegisterRoutes(api fiber.Router) {
	api.Get("/posts/:id/comments", h.list)
	api.Post("/posts/:id/comments", h.create)
	api.Patch("/comments/:id", h.update)
	api.Delete("/comments/:id", h.delete)
}

func (h *Comments) list(c fiber.Ctx) error {
	limit, offset := queryPage(c)
	page, err := h.service.List(c.Context(), currentUser(c), c.Params("id"), limit, offset)
	if err != nil {
		return writeCommentError(c, err)
	}
	return c.JSON(commentPageDTO(page, currentUser(c)))
}

func (h *Comments) create(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreateComment
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	comment, err := h.service.Create(c.Context(), *user, c.Params("id"), in.Body)
	if err != nil {
		return writeCommentError(c, err)
	}
	return c.Status(201).JSON(commentDTO(comment, user))
}

func (h *Comments) update(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdateComment
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	comment, err := h.service.Update(c.Context(), *user, c.Params("id"), in.Body)
	if err != nil {
		return writeCommentError(c, err)
	}
	return c.JSON(commentDTO(comment, user))
}

func (h *Comments) delete(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.service.Delete(c.Context(), *user, c.Params("id")); err != nil {
		return writeCommentError(c, err)
	}
	return c.SendStatus(204)
}

func writeCommentError(c fiber.Ctx, err error) error {
	var vErr *comments.ValidationError
	switch {
	case errors.As(err, &vErr):
		return fiber.NewError(400, vErr.Message)
	case errors.Is(err, comments.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	case errors.Is(err, comments.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	default:
		return writeError(c, err)
	}
}

func commentPageDTO(page model.CommentPage, viewer *model.User) dto.CommentPage {
	out := make([]dto.Comment, 0, len(page.Comments))
	for _, comment := range page.Comments {
		out = append(out, commentDTO(comment, viewer))
	}
	return dto.CommentPage{Total: page.Total, Comments: out}
}

func commentDTO(comment model.Comment, viewer *model.User) dto.Comment {
	canEdit := false
	canDelete := false
	if viewer != nil && viewer.ID != "" {
		if viewer.ID == comment.AuthorID {
			canEdit = true
			canDelete = true
		}
		if viewer.Role.AtLeast(model.RoleModerator) {
			canDelete = true
		}
	}
	return dto.Comment{
		ID: comment.ID, PostID: comment.PostID, AuthorUsername: comment.AuthorUsername,
		AuthorAvatarURL: commentAvatarURL(comment.AuthorID, comment.AuthorAvatarKey),
		Body:            comment.Body,
		CreatedAt:       comment.CreatedAt.Format(commentTime),
		UpdatedAt:       comment.UpdatedAt.Format(commentTime),
		CanEdit:         canEdit, CanDelete: canDelete,
	}
}

func commentAvatarURL(authorID, key string) string {
	if key == "" {
		return ""
	}
	v := key
	if len(v) > 12 {
		v = v[:12]
	}
	return "/media/avatars/" + authorID + "?v=" + v
}
