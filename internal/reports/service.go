package reports

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const maxNoteRunes = 500

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("you don't have permission to do that")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	ReportPostAccess(ctx context.Context, postID string) (model.ReportPostAccess, error)
	GetOpenPostReport(ctx context.Context, postID, reporterID string) (bool, error)
	CreatePostReport(ctx context.Context, postID, reporterID, reason, note string) (model.PostReport, error)
	ListOpenPostReports(ctx context.Context, limit, offset int32) ([]model.PostReport, error)
	CountOpenPostReports(ctx context.Context) (int64, error)
	DismissPostReport(ctx context.Context, id, resolverID string) (model.PostReport, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Create files one open report. A second open report from the same
// member on the same post is a validation error. The post is not hidden.
func (s *Service) Create(ctx context.Context, viewer model.User, postID, reason, note string) (model.PostReport, error) {
	if err := s.visiblePost(ctx, &viewer, postID); err != nil {
		return model.PostReport{}, err
	}
	if viewer.ID == "" {
		return model.PostReport{}, ErrForbidden
	}
	reason, err := normalizeReason(reason)
	if err != nil {
		return model.PostReport{}, err
	}
	note, err = normalizeNote(note)
	if err != nil {
		return model.PostReport{}, err
	}
	open, err := s.repo.GetOpenPostReport(ctx, postID, viewer.ID)
	if err != nil {
		return model.PostReport{}, err
	}
	if open {
		return model.PostReport{}, alreadyOpen()
	}
	created, err := s.repo.CreatePostReport(ctx, postID, viewer.ID, reason, note)
	if isUnique(err) {
		return model.PostReport{}, alreadyOpen()
	}
	return created, err
}

// ListOpen is the staff queue: newest open report first. It is not
// filtered by what this moderator has opted in to see. Dismissing a
// report does not hide the post.
func (s *Service) ListOpen(ctx context.Context, actor model.User, limit, offset int) (model.PostReportPage, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.PostReportPage{}, ErrForbidden
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountOpenPostReports(ctx)
	if err != nil {
		return model.PostReportPage{}, err
	}
	items, err := s.repo.ListOpenPostReports(ctx, lim, off)
	if err != nil {
		return model.PostReportPage{}, err
	}
	if items == nil {
		items = []model.PostReport{}
	}
	return model.PostReportPage{Total: total, Reports: items}, nil
}

// Dismiss closes one open report and records the moderator and the time.
// An already-closed or unknown report is not found. The post is unchanged.
func (s *Service) Dismiss(ctx context.Context, actor model.User, reportID string) (model.PostReport, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.PostReport{}, ErrForbidden
	}
	report, err := s.repo.DismissPostReport(ctx, reportID, actor.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.PostReport{}, ErrNotFound
	}
	return report, err
}

func (s *Service) visiblePost(ctx context.Context, viewer *model.User, postID string) error {
	post, err := s.repo.ReportPostAccess(ctx, postID)
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

// canSeePost matches the board: NSFW needs a signed-in viewer who opted
// in, and a hidden or deleted post is owner-or-moderator only. A miss
// is reported as not found so the post is not confirmed to exist.
func canSeePost(viewer *model.User, post model.ReportPostAccess) bool {
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

func normalizeReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	switch reason {
	case "illegal", "underage", "spam", "wrong_rating", "other":
		return reason, nil
	default:
		return "", &ValidationError{Message: "Reason must be illegal, underage, spam, wrong_rating, or other."}
	}
}

func normalizeNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNoteRunes {
		return "", &ValidationError{Message: "Notes are capped at 500 characters."}
	}
	return note, nil
}

func alreadyOpen() error {
	return &ValidationError{Message: "You already have an open report on this post."}
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
