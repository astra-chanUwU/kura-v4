package commentsafety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

var ErrForbidden = errors.New("you don't have permission to do that")

// maxResolvers keeps the decided-by list to the people who closed a
// review most recently.
const maxResolvers = 40

// ValidationError is a filter the queue cannot apply.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	List(ctx context.Context, filter model.CommentSafetyFilter, limit, offset int32) ([]model.CommentSafetyEvent, error)
	Count(ctx context.Context, filter model.CommentSafetyFilter) (int64, error)
	Resolvers(ctx context.Context) ([]model.CommentSafetyResolver, error)
	Dismiss(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error)
	Remove(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error)
	Reopen(ctx context.Context, id, actorID string) (model.CommentSafetyEvent, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List is the staff queue. An omitted status is every status; callers
// that want the working queue pass status open. It is not filtered by
// what this moderator has opted in to see.
func (s *Service) List(ctx context.Context, actor model.User, filter model.CommentSafetyFilter, limit, offset int) (model.CommentSafetyPage, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.CommentSafetyPage{}, ErrForbidden
	}
	normalized, err := normalizeFilter(filter)
	if err != nil {
		return model.CommentSafetyPage{}, err
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.Count(ctx, normalized)
	if err != nil {
		return model.CommentSafetyPage{}, err
	}
	events, err := s.repo.List(ctx, normalized, lim, off)
	if err != nil {
		return model.CommentSafetyPage{}, err
	}
	if events == nil {
		events = []model.CommentSafetyEvent{}
	}
	return model.CommentSafetyPage{Total: total, Events: events}, nil
}

// Resolvers lists moderators who have closed a review, newest decision
// first, including what that latest decision was.
func (s *Service) Resolvers(ctx context.Context, actor model.User) ([]model.CommentSafetyResolver, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return nil, ErrForbidden
	}
	people, err := s.repo.Resolvers(ctx)
	if err != nil {
		return nil, err
	}
	if people == nil {
		people = []model.CommentSafetyResolver{}
	}
	if len(people) > maxResolvers {
		people = people[:maxResolvers]
	}
	return people, nil
}

// Dismiss closes one open event and leaves any comment in place.
// An already-closed or unknown event is not found.
func (s *Service) Dismiss(ctx context.Context, actor model.User, id string) (model.CommentSafetyEvent, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.CommentSafetyEvent{}, ErrForbidden
	}
	return s.repo.Dismiss(ctx, id, actor.ID)
}

// Remove closes one open or dismissed event. The repository deletes a
// published comment in the same transaction when the event names one.
// An already-removed or unknown event is not found.
func (s *Service) Remove(ctx context.Context, actor model.User, id string) (model.CommentSafetyEvent, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.CommentSafetyEvent{}, ErrForbidden
	}
	return s.repo.Remove(ctx, id, actor.ID)
}

// Reopen puts a dismissed or removed event back on the open queue.
// When remove deleted a published comment, that comment is published
// again from the stored body. An open or unknown event is not found.
func (s *Service) Reopen(ctx context.Context, actor model.User, id string) (model.CommentSafetyEvent, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.CommentSafetyEvent{}, ErrForbidden
	}
	return s.repo.Reopen(ctx, id, actor.ID)
}

func normalizeFilter(filter model.CommentSafetyFilter) (model.CommentSafetyFilter, error) {
	status := strings.TrimSpace(filter.Status)
	switch status {
	case "", "any":
		status = ""
	case "open", "dismissed", "removed":
	default:
		return model.CommentSafetyFilter{}, &ValidationError{Message: "Status must be open, dismissed, removed, or any."}
	}
	action := strings.TrimSpace(filter.Action)
	switch action {
	case "", "blocked", "flagged":
	default:
		return model.CommentSafetyFilter{}, &ValidationError{Message: "Action must be blocked or flagged."}
	}
	category := strings.TrimSpace(filter.Category)
	if category != "" && !knownCategory(category) {
		return model.CommentSafetyFilter{}, &ValidationError{Message: "That category is not a safety category."}
	}
	query := strings.TrimSpace(filter.Query)
	if utf8.RuneCountInString(query) > 200 {
		return model.CommentSafetyFilter{}, &ValidationError{Message: "Search must be 200 characters or less."}
	}
	resolver := strings.TrimSpace(filter.ResolverID)
	if resolver != "" {
		if _, err := uuid.Parse(resolver); err != nil {
			return model.CommentSafetyFilter{}, &ValidationError{Message: "That moderator is not valid."}
		}
	}
	sort := strings.TrimSpace(filter.Sort)
	switch sort {
	case "", "newest":
		sort = "newest"
	case "oldest", "decided":
	default:
		return model.CommentSafetyFilter{}, &ValidationError{Message: "Sort must be newest, oldest, or decided."}
	}
	return model.CommentSafetyFilter{
		Status: status, Action: action, Category: category,
		Query: query, ResolverID: resolver, Sort: sort,
	}, nil
}

func knownCategory(category string) bool {
	switch category {
	case "phone", "telegram", "whatsapp", "discord", "signal", "snapchat", "kik", "email", "solicitation":
		return true
	default:
		return false
	}
}
