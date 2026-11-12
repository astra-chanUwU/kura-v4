package notifications

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

var ErrForbidden = errors.New("you don't have permission to do that")

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	Create(ctx context.Context, recipientID, actorID, kind, postID, commentID string) error
	List(ctx context.Context, recipientID string, limit, offset int32) ([]model.Notification, error)
	Count(ctx context.Context, recipientID string) (int64, error)
	CountUnread(ctx context.Context, recipientID string) (int64, error)
	MarkRead(ctx context.Context, recipientID string, ids []string) error
	MarkAllRead(ctx context.Context, recipientID string) error
	PostOwner(ctx context.Context, postID string) (string, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// PostOwner is the uploader of the post. A missing post or a post with
// no owner is an empty id, so the caller can skip the notice.
func (s *Service) PostOwner(ctx context.Context, postID string) (string, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return "", nil
	}
	owner, err := s.repo.PostOwner(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return strings.TrimSpace(owner), err
}

// Notify records one notice for recipientID, who is the post's uploader.
// An empty recipient is ignored. The actor is not notified about their
// own post. A favorite carries no comment id; a comment must.
func (s *Service) Notify(ctx context.Context, recipientID, actorID, kind, postID, commentID string) error {
	recipientID = strings.TrimSpace(recipientID)
	actorID = strings.TrimSpace(actorID)
	kind = strings.TrimSpace(kind)
	postID = strings.TrimSpace(postID)
	commentID = strings.TrimSpace(commentID)
	if recipientID == "" {
		return nil
	}
	if actorID != "" && actorID == recipientID {
		return nil
	}
	if postID == "" {
		return &ValidationError{Message: "A notification needs a post."}
	}
	switch kind {
	case model.NotificationFavorite:
		commentID = ""
	case model.NotificationComment:
		if commentID == "" {
			return &ValidationError{Message: "A comment notification needs a comment."}
		}
	default:
		return &ValidationError{Message: "Unknown notification."}
	}
	return s.repo.Create(ctx, recipientID, actorID, kind, postID, commentID)
}

// List returns the recipient's notices, newest first, plus the unread count.
// Only that recipient is queried.
func (s *Service) List(ctx context.Context, recipientID string, limit, offset int) (model.NotificationPage, error) {
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return model.NotificationPage{}, ErrForbidden
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.Count(ctx, recipientID)
	if err != nil {
		return model.NotificationPage{}, err
	}
	unread, err := s.repo.CountUnread(ctx, recipientID)
	if err != nil {
		return model.NotificationPage{}, err
	}
	items, err := s.repo.List(ctx, recipientID, lim, off)
	if err != nil {
		return model.NotificationPage{}, err
	}
	if items == nil {
		items = []model.Notification{}
	}
	return model.NotificationPage{Total: total, Unread: unread, Notifications: items}, nil
}

// MarkRead sets read_at to now for the recipient. An empty id list marks
// every unread notice of that recipient. Other people's rows are not touched.
func (s *Service) MarkRead(ctx context.Context, recipientID string, ids []string) error {
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return ErrForbidden
	}
	if len(ids) == 0 {
		return s.repo.MarkAllRead(ctx, recipientID)
	}
	return s.repo.MarkRead(ctx, recipientID, ids)
}
