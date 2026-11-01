package tags

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const (
	maxDescription  = 2000
	categoryMessage = "Category must be general, artist, character, copyright, or meta."
	tagMessage      = "That tag name is not valid."
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	tagCleanRE   = regexp.MustCompile(`[^a-z0-9_]+`)
)

// ValidationError is a rejected tag page or alias, safe to show to the member.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

// Repository is the tag wiki and alias store.
type Repository interface {
	GetPage(ctx context.Context, tag string) (model.TagWiki, bool, error)
	UpsertPage(ctx context.Context, tag, category, description, userID string) error
	CountPosts(ctx context.Context, tag string, includeNSFW bool) (int64, error)
	RecentPosts(ctx context.Context, tag string, includeNSFW bool) ([]model.Post, error)
	AliasTarget(ctx context.Context, alias string) (string, bool, error)
	AliasesOf(ctx context.Context, target string) ([]string, error)
	IsAliasTarget(ctx context.Context, tag string) (bool, error)
	CreateAlias(ctx context.Context, alias, target, userID string) error
	DeleteAlias(ctx context.Context, alias, target string) (int64, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Get returns the wiki and recent posts for a tag. An alias name returns
// the canonical tag's page and sets AliasOf.
func (s *Service) Get(ctx context.Context, viewer *model.User, raw string) (model.TagPage, error) {
	tag, err := oneTag(raw)
	if err != nil {
		return model.TagPage{}, err
	}
	return s.page(ctx, viewer, tag)
}

// Update replaces the category and description. Any signed-in member may
// edit. Editing an alias updates the canonical tag.
func (s *Service) Update(ctx context.Context, viewer model.User, raw, category, description string) (model.TagPage, error) {
	tag, err := oneTag(raw)
	if err != nil {
		return model.TagPage{}, err
	}
	canonical, _, err := s.canonical(ctx, tag)
	if err != nil {
		return model.TagPage{}, err
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if !validCategory(category) {
		return model.TagPage{}, invalid(categoryMessage)
	}
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > maxDescription {
		return model.TagPage{}, invalid("Description must be 2000 characters or fewer.")
	}
	if err := s.repo.UpsertPage(ctx, canonical, category, description, viewer.ID); err != nil {
		return model.TagPage{}, err
	}
	return s.page(ctx, &viewer, tag)
}

// AddAlias points alias at target and moves existing posts onto the target.
// Moderators and above may do this. Chains are rejected: a target cannot
// itself be an alias, and an alias cannot already be someone else's target.
func (s *Service) AddAlias(ctx context.Context, actor model.User, targetRaw, aliasRaw string) (model.TagPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.TagPage{}, ErrForbidden
	}
	target, err := oneTag(targetRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	alias, err := oneTag(aliasRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	if alias == target {
		return model.TagPage{}, invalid("A tag can't be an alias of itself.")
	}
	if to, ok, err := s.repo.AliasTarget(ctx, target); err != nil {
		return model.TagPage{}, err
	} else if ok {
		return model.TagPage{}, invalid(target + " is an alias of " + to + ". Point the new name at that tag.")
	}
	if _, ok, err := s.repo.AliasTarget(ctx, alias); err != nil {
		return model.TagPage{}, err
	} else if ok {
		return model.TagPage{}, invalid("That name is already an alias.")
	}
	if taken, err := s.repo.IsAliasTarget(ctx, alias); err != nil {
		return model.TagPage{}, err
	} else if taken {
		return model.TagPage{}, invalid("Other tags already point at that name.")
	}
	if err := s.repo.CreateAlias(ctx, alias, target, actor.ID); err != nil {
		return model.TagPage{}, err
	}
	return s.page(ctx, &actor, target)
}

// RemoveAlias drops one alternate name. Posts stay on the canonical tag.
func (s *Service) RemoveAlias(ctx context.Context, actor model.User, targetRaw, aliasRaw string) (model.TagPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.TagPage{}, ErrForbidden
	}
	target, err := oneTag(targetRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	alias, err := oneTag(aliasRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	n, err := s.repo.DeleteAlias(ctx, alias, target)
	if err != nil {
		return model.TagPage{}, err
	}
	if n == 0 {
		return model.TagPage{}, ErrNotFound
	}
	return s.page(ctx, &actor, target)
}

func (s *Service) canonical(ctx context.Context, tag string) (string, bool, error) {
	target, ok, err := s.repo.AliasTarget(ctx, tag)
	if err != nil || !ok {
		return tag, false, err
	}
	return target, true, nil
}

func (s *Service) page(ctx context.Context, viewer *model.User, tag string) (model.TagPage, error) {
	canonical, isAlias, err := s.canonical(ctx, tag)
	if err != nil {
		return model.TagPage{}, err
	}
	wiki, found, err := s.repo.GetPage(ctx, canonical)
	if err != nil {
		return model.TagPage{}, err
	}
	includeNSFW := viewer != nil && viewer.ShowNSFW
	count, err := s.repo.CountPosts(ctx, canonical, includeNSFW)
	if err != nil {
		return model.TagPage{}, err
	}
	aliases, err := s.repo.AliasesOf(ctx, canonical)
	if err != nil {
		return model.TagPage{}, err
	}
	if aliases == nil {
		aliases = []string{}
	}
	if !found && count == 0 && !isAlias && len(aliases) == 0 {
		return model.TagPage{}, ErrNotFound
	}
	posts, err := s.repo.RecentPosts(ctx, canonical, includeNSFW)
	if err != nil {
		return model.TagPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	page := model.TagPage{
		Tag: tag, Aliases: aliases, Category: "general", PostCount: count, Posts: posts,
	}
	if isAlias {
		page.AliasOf = canonical
	}
	if found {
		page.Category = wiki.Category
		page.Description = wiki.Description
		page.UpdatedByUsername = wiki.UpdatedByUsername
		page.UpdatedAt = wiki.UpdatedAt
	}
	return page, nil
}

func oneTag(raw string) (string, error) {
	tags := normalizeTags([]string{raw})
	if len(tags) != 1 {
		return "", invalid(tagMessage)
	}
	return tags[0], nil
}

func validCategory(category string) bool {
	switch category {
	case "general", "artist", "character", "copyright", "meta":
		return true
	default:
		return false
	}
}

// normalizeTags matches the board: spaces and commas separate tags,
// hyphens become underscores, and the result is lowercase and unique.
func normalizeTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range in {
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ',' || r == '\n'
		}) {
			tag := tagCleanRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(token)), "_")
			tag = strings.Trim(tag, "_")
			if tag == "" || len(tag) > 40 || seen[tag] {
				continue
			}
			seen[tag] = true
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}
