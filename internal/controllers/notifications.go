package controllers

import (
	"bytes"
	"context"
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/notifications"
	"github.com/gofiber/fiber/v3"
)

const notificationTime = "2006-01-02T15:04:05Z07:00"

// Notifier is the optional hook the favorite and comment handlers use to
// tell a post's uploader what happened. The parent sets it from http.go.
// A nil notifier skips the notice. PostOwner is how those handlers learn
// the uploader; they do not already have that id.
type Notifier interface {
	Notify(ctx context.Context, recipientID, actorID, kind, postID, commentID string) error
	PostOwner(ctx context.Context, postID string) (string, error)
}

// notifyUploader tells the post owner about a new favorite or comment.
// The action itself has already succeeded, so a missed notice is not an
// error response. Nothing is sent when the actor is the owner, the post
// has no owner, or no notifier is wired.
func notifyUploader(c fiber.Ctx, n Notifier, actorID, kind, postID, commentID string) {
	if n == nil || actorID == "" || postID == "" {
		return
	}
	owner, err := n.PostOwner(c.Context(), postID)
	if err != nil || owner == "" || owner == actorID {
		return
	}
	_ = n.Notify(c.Context(), owner, actorID, kind, postID, commentID)
}

type Notifications struct{ service *notifications.Service }

func NewNotifications(s *notifications.Service) *Notifications {
	return &Notifications{service: s}
}

func (h *Notifications) RegisterRoutes(api fiber.Router) {
	api.Get("/notifications", h.list)
	api.Post("/notifications/read", h.markRead)
}

func (h *Notifications) list(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.service.List(c.Context(), user.ID, limit, offset)
	if err != nil {
		return writeNotificationError(c, err)
	}
	items := make([]dto.Notification, 0, len(page.Notifications))
	for _, item := range page.Notifications {
		items = append(items, notificationDTO(item))
	}
	return c.JSON(dto.NotificationPage{Total: page.Total, Unread: page.Unread, Notifications: items})
}

func (h *Notifications) markRead(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.MarkNotificationsRead
	if len(bytes.TrimSpace(c.Body())) > 0 {
		if err := c.Bind().Body(&in); err != nil {
			return fiber.NewError(400, "Invalid request.")
		}
	}
	if err := h.service.MarkRead(c.Context(), user.ID, in.IDs); err != nil {
		return writeNotificationError(c, err)
	}
	return c.SendStatus(204)
}

func notificationDTO(item model.Notification) dto.Notification {
	return dto.Notification{
		ID:            item.ID,
		Kind:          item.Kind,
		PostID:        item.PostID,
		CommentID:     item.CommentID,
		ActorUsername: item.ActorUsername,
		Read:          item.Read,
		CreatedAt:     item.CreatedAt.UTC().Format(notificationTime),
	}
}

func writeNotificationError(c fiber.Ctx, err error) error {
	var vErr *notifications.ValidationError
	switch {
	case errors.As(err, &vErr):
		return fiber.NewError(400, vErr.Message)
	case errors.Is(err, notifications.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	default:
		return err
	}
}
