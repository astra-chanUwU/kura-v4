package comments

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const maxCommentRunes = 2000

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("you don't have permission to do that")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	CommentPostAccess(ctx context.Context, postID string) (model.CommentPostAccess, error)
	ListCommentsByPost(ctx context.Context, postID string, limit, offset int32) ([]model.Comment, error)
	CountCommentsByPost(ctx context.Context, postID string) (int64, error)
	CreateComment(ctx context.Context, postID, authorID, body string) (model.Comment, error)
	GetComment(ctx context.Context, id string) (model.Comment, error)
	UpdateCommentBody(ctx context.Context, id, body string) (model.Comment, error)
	DeleteComment(ctx context.Context, id string) (int64, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, viewer *model.User, postID string, limit, offset int) (model.CommentPage, error) {
	if err := s.visiblePost(ctx, viewer, postID); err != nil {
		return model.CommentPage{}, err
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountCommentsByPost(ctx, postID)
	if err != nil {
		return model.CommentPage{}, err
	}
	comments, err := s.repo.ListCommentsByPost(ctx, postID, lim, off)
	if err != nil {
		return model.CommentPage{}, err
	}
	if comments == nil {
		comments = []model.Comment{}
	}
	return model.CommentPage{Total: total, Comments: comments}, nil
}

func (s *Service) Create(ctx context.Context, viewer model.User, postID, body string) (model.Comment, error) {
	if err := s.visiblePost(ctx, &viewer, postID); err != nil {
		return model.Comment{}, err
	}
	if viewer.ID == "" {
		return model.Comment{}, ErrForbidden
	}
	body, err := normalizeBody(body)
	if err != nil {
		return model.Comment{}, err
	}
	return s.repo.CreateComment(ctx, postID, viewer.ID, body)
}

func (s *Service) Update(ctx context.Context, viewer model.User, commentID, body string) (model.Comment, error) {
	current, err := s.visibleComment(ctx, &viewer, commentID)
	if err != nil {
		return model.Comment{}, err
	}
	if viewer.ID == "" || viewer.ID != current.AuthorID {
		return model.Comment{}, ErrForbidden
	}
	body, err = normalizeBody(body)
	if err != nil {
		return model.Comment{}, err
	}
	updated, err := s.repo.UpdateCommentBody(ctx, commentID, body)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	return updated, err
}

func (s *Service) Delete(ctx context.Context, viewer model.User, commentID string) error {
	current, err := s.visibleComment(ctx, &viewer, commentID)
	if err != nil {
		return err
	}
	if viewer.ID == "" || (viewer.ID != current.AuthorID && !viewer.Role.AtLeast(model.RoleModerator)) {
		return ErrForbidden
	}
	n, err := s.repo.DeleteComment(ctx, commentID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) visiblePost(ctx context.Context, viewer *model.User, postID string) error {
	post, err := s.repo.CommentPostAccess(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canSeePost(viewer, post) {
		return ErrNotFound
	}
	return nil
}

func (s *Service) visibleComment(ctx context.Context, viewer *model.User, commentID string) (model.Comment, error) {
	comment, err := s.repo.GetComment(ctx, commentID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	if err != nil {
		return model.Comment{}, err
	}
	if err := s.visiblePost(ctx, viewer, comment.PostID); err != nil {
		return model.Comment{}, err
	}
	return comment, nil
}

// canSeePost matches the board: NSFW needs a signed-in viewer who opted
// in, and a hidden or deleted post is owner-or-moderator only. A miss
// is reported as not found so the post is not confirmed to exist.
func canSeePost(viewer *model.User, post model.CommentPostAccess) bool {
	if post.Rating == "nsfw" && !seesNSFW(viewer) {
		return false
	}
	if !post.Hidden && !post.Deleted {
		return true
	}
	if viewer == nil || viewer.ID == "" {
		return false
	}
	if viewer.Role.AtLeast(model.RoleModerator) {
		return true
	}
	return post.OwnerID != "" && post.OwnerID == viewer.ID
}

func seesNSFW(viewer *model.User) bool {
	return viewer != nil && viewer.ID != "" && viewer.ShowNSFW
}

func normalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", &ValidationError{Message: "Write a comment."}
	}
	if utf8.RuneCountInString(body) > maxCommentRunes {
		return "", &ValidationError{Message: "Comments are capped at 2000 characters."}
	}
	return body, nil
}
