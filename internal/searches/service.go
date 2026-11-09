package searches

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const (
	maxSavedSearches = 30
	maxNameRunes     = 60
	maxQueryRunes    = 80
	maxTagLen        = 40
	maxTags          = 30
)

var (
	ErrNotFound = errors.New("not found")

	// Spaces and other separators become underscores, then hyphens do too,
	// so "White Hair" and "white-hair" are the same tag.
	tagCleanRE = regexp.MustCompile(`[^a-z0-9_-]+`)
	tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,39}$`)
)

// ValidationError is a rejected saved search. Controllers map it to 400.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Repository is the saved-search store. Implementations scope every
// call to the signed-in user.
type Repository interface {
	List(ctx context.Context, userID string) ([]model.SavedSearch, error)
	Count(ctx context.Context, userID string) (int64, error)
	Create(ctx context.Context, userID string, in model.SavedSearch) (model.SavedSearch, error)
	Delete(ctx context.Context, userID, id string) (int64, error)
}

// Service stores private board filters for one account.
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string) ([]model.SavedSearch, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []model.SavedSearch{}, nil
	}
	for i := range items {
		items[i].IncludeTags = tagsOrEmpty(items[i].IncludeTags)
		items[i].ExcludeTags = tagsOrEmpty(items[i].ExcludeTags)
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, userID string, in model.SavedSearch) (model.SavedSearch, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return model.SavedSearch{}, &ValidationError{Message: "Name your search first."}
	}
	if len([]rune(name)) > maxNameRunes {
		return model.SavedSearch{}, &ValidationError{Message: "Search names are capped at 60 characters."}
	}
	query := strings.TrimSpace(in.Query)
	if len([]rune(query)) > maxQueryRunes {
		return model.SavedSearch{}, &ValidationError{Message: "Search text is capped at 80 characters."}
	}
	order, err := normalizeSort(in.Sort)
	if err != nil {
		return model.SavedSearch{}, err
	}
	include := normalizeTags(in.IncludeTags)
	exclude := normalizeTags(in.ExcludeTags)

	count, err := s.repo.Count(ctx, userID)
	if err != nil {
		return model.SavedSearch{}, err
	}
	if count >= maxSavedSearches {
		return model.SavedSearch{}, &ValidationError{Message: "You can save up to 30 searches."}
	}
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return model.SavedSearch{}, err
	}
	for _, item := range existing {
		if item.Name == name {
			return model.SavedSearch{}, &ValidationError{Message: "You already saved a search with that name."}
		}
	}

	created, err := s.repo.Create(ctx, userID, model.SavedSearch{
		UserID:      userID,
		Name:        name,
		IncludeTags: include,
		ExcludeTags: exclude,
		Query:       query,
		Sort:        order,
	})
	if isUnique(err) {
		return model.SavedSearch{}, &ValidationError{Message: "You already saved a search with that name."}
	}
	if err != nil {
		return model.SavedSearch{}, err
	}
	created.IncludeTags = tagsOrEmpty(created.IncludeTags)
	created.ExcludeTags = tagsOrEmpty(created.ExcludeTags)
	return created, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	n, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func normalizeSort(sort string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "", "newest":
		return "newest", nil
	case "oldest":
		return "oldest", nil
	case "favorites":
		return "favorites", nil
	default:
		return "", &ValidationError{Message: "Sort must be newest, oldest, or favorites."}
	}
}

// normalizeTags canonicalizes a tag list without importing the board
// package. Commas separate tags. Spaces and hyphens join words inside
// a tag, so "White Hair" and "white-hair" are both white_hair. Empty
// tags are dropped, the rest are deduped and sorted, and each list
// keeps at most 30 tags of 40 characters matching the database pattern.
func normalizeTags(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, raw := range in {
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\t'
		}) {
			tag := tagCleanRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(token)), "_")
			tag = strings.ReplaceAll(tag, "-", "_")
			tag = strings.Trim(tag, "_")
			if tag == "" || len(tag) > maxTagLen || !tagPattern.MatchString(tag) || seen[tag] {
				continue
			}
			seen[tag] = true
			out = append(out, tag)
			if len(out) >= maxTags {
				break
			}
		}
		if len(out) >= maxTags {
			break
		}
	}
	sort.Strings(out)
	return out
}

func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
