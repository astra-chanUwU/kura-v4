package moderation

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("you don't have permission to do that")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
func invalid(msg string) error           { return &ValidationError{Message: msg} }

type Repository interface {
	ListUsers(ctx context.Context) ([]model.User, error)
	ListFilteredUsers(ctx context.Context, query, role, sort string, limit, offset int32) ([]model.User, error)
	CountFilteredUsers(ctx context.Context, query, role string) (int64, error)
	CountUsersByRoles(ctx context.Context) (member, moderator, admin, superAdmin int64, err error)
	GetUser(ctx context.Context, id string) (model.User, error)
	GetPost(ctx context.Context, id string) (model.Post, error)
	CountSuperAdmins(ctx context.Context) (int64, error)
	ApplyUserRole(ctx context.Context, userID string, role model.Role, entry model.ChangeEntry) (model.User, error)
	ApplyPostSnapshot(ctx context.Context, p model.Post, entry model.ChangeEntry) error
	EnsureArtistPage(ctx context.Context, tag string) error
	SetPostHidden(ctx context.Context, postID string, hidden, deleted bool, actorID string, entry model.ChangeEntry) error
	ListHiddenPosts(ctx context.Context, kind, query, rating, sort string, limit, offset int32) ([]model.Post, error)
	CountHiddenPosts(ctx context.Context, kind, query, rating string) (int64, error)
	CountRemovedPosts(ctx context.Context) (int64, int64, error)
	GetHistory(ctx context.Context, id string) (model.ChangeEntry, error)
	ListHistoryForEntity(ctx context.Context, entityType, entityID string, limit int32) ([]model.ChangeEntry, error)
	ListRecentHistory(ctx context.Context, limit int32) ([]model.ChangeEntry, error)
	ListHistoryByActor(ctx context.Context, actorID string, limit int32) ([]model.ChangeEntry, error)
	ListHistory(ctx context.Context, actorID, query, action, entity, sort string, limit, offset int32) ([]model.ChangeEntry, error)
	CountHistory(ctx context.Context, actorID, query, action, entity string) (int64, error)
	SearchUsernames(ctx context.Context, query string) ([]model.HistoryActor, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func notFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

// ListUsers is admin+ only (enforced by the caller check here, not just UI).
func (s *Service) ListUsers(ctx context.Context, actor model.User) ([]model.User, error) {
	if !actor.Role.AtLeast(model.RoleAdmin) {
		return nil, ErrForbidden
	}
	users, err := s.repo.ListUsers(ctx)
	if users == nil {
		users = []model.User{}
	}
	return users, err
}

// ListUserPage is one page of the staff people directory (admin+).
// query is a case-insensitive substring of username or display name;
// blank matches everything. role blank is every role. sort blank is name.
// Member, Moderator, Admin, and SuperAdmin are the full role counts,
// ignoring query and role. Total is how many rows match query and role.
func (s *Service) ListUserPage(ctx context.Context, actor model.User, query, role, sort string, limit, offset int) (model.UserPage, error) {
	if !actor.Role.AtLeast(model.RoleAdmin) {
		return model.UserPage{}, ErrForbidden
	}
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 80 {
		return model.UserPage{}, invalid("q must be 80 characters or fewer")
	}
	if role != "" && !model.Role(role).Valid() {
		return model.UserPage{}, invalid("role must be member, moderator, admin, or super_admin")
	}
	switch sort {
	case "", "name":
		sort = "name"
	case "newest", "oldest":
	default:
		return model.UserPage{}, invalid("sort must be name, newest, or oldest")
	}
	lim, off := model.PageBounds(limit, offset)
	members, moderators, admins, superAdmins, err := s.repo.CountUsersByRoles(ctx)
	if err != nil {
		return model.UserPage{}, err
	}
	total, err := s.repo.CountFilteredUsers(ctx, query, role)
	if err != nil {
		return model.UserPage{}, err
	}
	users, err := s.repo.ListFilteredUsers(ctx, query, role, sort, lim, off)
	if err != nil {
		return model.UserPage{}, err
	}
	if users == nil {
		users = []model.User{}
	}
	return model.UserPage{
		Total: total, Member: members, Moderator: moderators, Admin: admins, SuperAdmin: superAdmins, Users: users,
	}, nil
}

// SetRole changes target's role and records it. Hierarchy (model.CanManageRole):
// super_admin manages anyone, admin manages member<->moderator only.
// The last super_admin can never be demoted, and nobody demotes themselves
// out of a super_admin seat they would strand.
func (s *Service) SetRole(ctx context.Context, actor model.User, targetID string, newRole model.Role) (model.User, error) {
	if !newRole.Valid() {
		return model.User{}, invalid("Unknown role.")
	}
	target, err := s.repo.GetUser(ctx, targetID)
	if notFound(err) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, err
	}
	if !model.CanManageRole(actor.Role, target.Role, newRole) {
		return model.User{}, ErrForbidden
	}
	if target.Role == newRole {
		return target, nil
	}
	if target.Role == model.RoleSuperAdmin && newRole != model.RoleSuperAdmin {
		count, err := s.repo.CountSuperAdmins(ctx)
		if err != nil {
			return model.User{}, err
		}
		if count <= 1 {
			return model.User{}, invalid("There must always be at least one super admin.")
		}
	}
	entry := model.ChangeEntry{EntityType: "user", EntityID: target.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
		Action: "user.role_changed", Before: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: target.Role}),
		After: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: newRole})}
	return s.repo.ApplyUserRole(ctx, target.ID, newRole, entry)
}

// SetPostHidden hides (moderator+) or deletes (admin+, same soft-remove:
// the post leaves the board but its bytes and history stay, so it can be
// rolled back) a post, or unhides it (moderator+).
func (s *Service) SetPostHidden(ctx context.Context, actor model.User, postID string, hidden bool, deleted bool) (model.Post, error) {
	if deleted && hidden {
		if !actor.Role.AtLeast(model.RoleAdmin) {
			return model.Post{}, ErrForbidden
		}
	} else if !actor.Role.AtLeast(model.RoleModerator) {
		return model.Post{}, ErrForbidden
	}
	post, err := s.repo.GetPost(ctx, postID)
	if notFound(err) {
		return model.Post{}, ErrNotFound
	}
	if err != nil {
		return model.Post{}, err
	}
	if post.Hidden == hidden && post.Deleted == (hidden && deleted) {
		return post, nil
	}
	before := model.SnapshotOf(post)
	post.Hidden = hidden
	post.Deleted = hidden && deleted
	action := "post.unhidden"
	if hidden && deleted {
		action = "post.deleted"
	} else if hidden {
		action = "post.hidden"
	}
	entry := model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
		Action: action, Before: model.MarshalPostSnapshot(before), After: model.MarshalPostSnapshot(model.SnapshotOf(post))}
	if err := s.repo.SetPostHidden(ctx, post.ID, hidden, deleted, actor.ID, entry); err != nil {
		return model.Post{}, err
	}
	return post, nil
}

// HistoryForPost lists a post's changes, newest first. Any signed-in user
// may view it (transparency); rolling back needs moderator+ (see Rollback).
func (s *Service) HistoryForPost(ctx context.Context, postID string, limit int32) ([]model.ChangeEntry, error) {
	if _, err := s.repo.GetPost(ctx, postID); notFound(err) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListHistoryForEntity(ctx, "post", postID, clampLimit(limit))
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return entries, err
}

// RecentHistory is the staff audit feed (moderator+).
func (s *Service) RecentHistory(ctx context.Context, actor model.User, limit int32) ([]model.ChangeEntry, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return nil, ErrForbidden
	}
	entries, err := s.repo.ListRecentHistory(ctx, clampLimit(limit))
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return entries, err
}

// HistoryByActor lists everything one account changed — the rogue-staff
// view: every row can then be rolled back individually (moderator+ for
// post rows, admin rules for role rows, enforced in Rollback).
func (s *Service) HistoryByActor(ctx context.Context, actor model.User, actorID string, limit int32) ([]model.ChangeEntry, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return nil, ErrForbidden
	}
	entries, err := s.repo.ListHistoryByActor(ctx, actorID, clampLimit(limit))
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return entries, err
}

// ListHistory is one filtered page of the staff change feed (moderator+).
// query matches an actor name or an artist, role, rating, source link, or
// tag list on either snapshot; blank matches everything. actorID blank is
// every actor. A non-empty actorID must already be a user id and is not
// looked up. action and entity blank match every value. sort blank is newest.
func (s *Service) ListHistory(ctx context.Context, actor model.User, actorID, query, action, entity, sort string, limit, offset int) (model.HistoryPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.HistoryPage{}, ErrForbidden
	}
	query, err := historyQuery(query)
	if err != nil {
		return model.HistoryPage{}, err
	}
	if actorID != "" {
		id, err := uuid.Parse(actorID)
		if err != nil {
			return model.HistoryPage{}, invalid("actor_id must be a user id")
		}
		actorID = id.String()
	}
	if err := historyAction(action); err != nil {
		return model.HistoryPage{}, err
	}
	switch entity {
	case "", "post", "user":
	default:
		return model.HistoryPage{}, invalid("entity must be post or user")
	}
	switch sort {
	case "", "newest", "oldest":
	default:
		return model.HistoryPage{}, invalid("sort must be newest or oldest")
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountHistory(ctx, actorID, query, action, entity)
	if err != nil {
		return model.HistoryPage{}, err
	}
	entries, err := s.repo.ListHistory(ctx, actorID, query, action, entity, sort, lim, off)
	if err != nil {
		return model.HistoryPage{}, err
	}
	if entries == nil {
		entries = []model.ChangeEntry{}
	}
	return model.HistoryPage{Total: total, Entries: entries}, nil
}

// SearchHistoryActors returns up to eight usernames for the history actor
// picker (moderator+). A blank query returns nothing and does not query.
func (s *Service) SearchHistoryActors(ctx context.Context, actor model.User, query string) ([]model.HistoryActor, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return nil, ErrForbidden
	}
	query, err := historyQuery(query)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return []model.HistoryActor{}, nil
	}
	actors, err := s.repo.SearchUsernames(ctx, query)
	if err != nil {
		return nil, err
	}
	if actors == nil {
		actors = []model.HistoryActor{}
	}
	return actors, nil
}

func historyQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > 80 {
		return "", invalid("q must be 80 characters or fewer")
	}
	return q, nil
}

func historyAction(action string) error {
	switch action {
	case "", "post.created", "post.meta_updated", "post.hidden", "post.unhidden", "post.deleted", "user.role_changed", "rollback":
		return nil
	default:
		return invalid("action is not a history action")
	}
}

// Rollback restores the before_state of one history row and records the
// rollback as a new row (the log is append-only, never rewritten). This
// covers both rogue-staff cleanup and changing your mind after a change.
// Post rows need moderator+; role rows re-check CanManageRole against the
// target's current role and the role being restored.
func (s *Service) Rollback(ctx context.Context, actor model.User, historyID string) (model.ChangeEntry, error) {
	entry, err := s.repo.GetHistory(ctx, historyID)
	if notFound(err) {
		return model.ChangeEntry{}, ErrNotFound
	}
	if err != nil {
		return model.ChangeEntry{}, err
	}
	switch entry.EntityType {
	case "post":
		return s.rollbackPost(ctx, actor, entry)
	case "user":
		return s.rollbackUser(ctx, actor, entry)
	default:
		return model.ChangeEntry{}, ErrNotFound
	}
}

func (s *Service) rollbackPost(ctx context.Context, actor model.User, entry model.ChangeEntry) (model.ChangeEntry, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.ChangeEntry{}, ErrForbidden
	}
	before, err := entry.PostBefore()
	if err != nil {
		return model.ChangeEntry{}, err
	}
	post, err := s.repo.GetPost(ctx, entry.EntityID)
	if notFound(err) {
		return model.ChangeEntry{}, ErrNotFound
	}
	if err != nil {
		return model.ChangeEntry{}, err
	}
	current := model.SnapshotOf(post)
	post.Artist, post.Tags, post.Rating, post.Hidden, post.Deleted = before.Artist, before.Tags, before.Rating, before.Hidden, before.Deleted
	if before.HasSourceURL {
		post.SourceURL = before.SourceURL
	} else {
		// The row predates source links. Leave the current link on the post,
		// and record it on this rollback so rolling the rollback back does not clear it.
		before.SourceURL = post.SourceURL
	}
	if post.Rating != "sfw" && post.Rating != "nsfw" {
		post.Rating = "sfw"
	}
	post.Tags = model.ApplyArtistTag("", post.Tags, post.Artist)
	before.Tags = post.Tags
	rb := model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
		Action: "rollback", Before: model.MarshalPostSnapshot(current), After: model.MarshalPostSnapshot(before), RollbackOf: &entry.ID}
	if err := s.repo.ApplyPostSnapshot(ctx, post, rb); err != nil {
		return model.ChangeEntry{}, err
	}
	if token := model.ArtistToken(post.Artist); token != "" && tagPresent(post.Tags, token) {
		if err := s.repo.EnsureArtistPage(ctx, token); err != nil {
			return model.ChangeEntry{}, err
		}
	}
	return rb, nil
}

func (s *Service) rollbackUser(ctx context.Context, actor model.User, entry model.ChangeEntry) (model.ChangeEntry, error) {
	before, err := entry.RoleBefore()
	if err != nil {
		return model.ChangeEntry{}, err
	}
	target, err := s.repo.GetUser(ctx, entry.EntityID)
	if notFound(err) {
		return model.ChangeEntry{}, ErrNotFound
	}
	if err != nil {
		return model.ChangeEntry{}, err
	}
	if !model.CanManageRole(actor.Role, target.Role, before.Role) {
		return model.ChangeEntry{}, ErrForbidden
	}
	if target.Role == before.Role {
		return entry, nil
	}
	if target.Role == model.RoleSuperAdmin && before.Role != model.RoleSuperAdmin {
		count, err := s.repo.CountSuperAdmins(ctx)
		if err != nil {
			return model.ChangeEntry{}, err
		}
		if count <= 1 {
			return model.ChangeEntry{}, invalid("There must always be at least one super admin.")
		}
	}
	rb := model.ChangeEntry{EntityType: "user", EntityID: target.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
		Action: "rollback", Before: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: target.Role}),
		After: model.MarshalRoleSnapshot(before), RollbackOf: &entry.ID}
	if _, err := s.repo.ApplyUserRole(ctx, target.ID, before.Role, rb); err != nil {
		return model.ChangeEntry{}, err
	}
	return rb, nil
}

// ListHiddenPosts is one page of the staff review queue (moderator+).
// kind is all, hidden, or deleted. query is a case-insensitive substring
// of the artist, uploader, remover, or any tag; blank matches everything.
// rating is sfw, nsfw, or blank. sort is newest or oldest removal time,
// and blank is newest. Hidden and Deleted are the full queue counts.
// Total is how many rows match kind, query, and rating.
func (s *Service) ListHiddenPosts(ctx context.Context, actor model.User, kind, query, rating, sort string, limit, offset int) (model.HiddenPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.HiddenPage{}, ErrForbidden
	}
	switch kind {
	case "", "all":
		kind = "all"
	case "hidden", "deleted":
	default:
		return model.HiddenPage{}, invalid("kind must be all, hidden, or deleted")
	}
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 80 {
		return model.HiddenPage{}, invalid("q must be 80 characters or fewer")
	}
	switch rating {
	case "", "sfw", "nsfw":
	default:
		return model.HiddenPage{}, invalid("rating must be sfw or nsfw")
	}
	switch sort {
	case "", "newest", "oldest":
	default:
		return model.HiddenPage{}, invalid("sort must be newest or oldest")
	}
	lim, off := model.PageBounds(limit, offset)
	hidden, deleted, err := s.repo.CountRemovedPosts(ctx)
	if err != nil {
		return model.HiddenPage{}, err
	}
	total, err := s.repo.CountHiddenPosts(ctx, kind, query, rating)
	if err != nil {
		return model.HiddenPage{}, err
	}
	posts, err := s.repo.ListHiddenPosts(ctx, kind, query, rating, sort, lim, off)
	if err != nil {
		return model.HiddenPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return model.HiddenPage{Total: total, Hidden: hidden, Deleted: deleted, Posts: posts}, nil
}

func tagPresent(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
