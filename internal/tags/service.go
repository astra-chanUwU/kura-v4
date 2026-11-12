package tags

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const (
	maxDescription      = 2000
	maxSuggestionReason = 300
	categoryMessage     = "Category must be general, artist, character, copyright, or meta."
	tagMessage          = "That tag name is not valid."
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
	CountPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) (int64, error)
	RecentPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) ([]model.Post, error)
	AliasTarget(ctx context.Context, alias string) (string, bool, error)
	AliasesOf(ctx context.Context, target string) ([]string, error)
	IsAliasTarget(ctx context.Context, tag string) (bool, error)
	CreateAlias(ctx context.Context, alias, target, userID string) error
	DeleteAlias(ctx context.Context, alias, target string) (int64, error)
	ListImplications(ctx context.Context) ([]model.TagImplication, error)
	// CreateImplication stores the edge and adds each backfill tag to posts
	// that already have the antecedent. The caller chooses those tags.
	CreateImplication(ctx context.Context, antecedent, consequent, userID string, backfill []string) error
	DeleteImplication(ctx context.Context, antecedent, consequent string) (int64, error)
	CreateSuggestion(ctx context.Context, kind, sourceTag, targetTag, reason, authorID string) (model.TagSuggestion, error)
	GetSuggestion(ctx context.Context, id string) (model.TagSuggestion, error)
	ListSuggestions(ctx context.Context, status string) ([]model.TagSuggestion, error)
	// ResolveSuggestion sets status when the row is still fromStatus.
	// A row that no longer has that status is sql.ErrNoRows.
	ResolveSuggestion(ctx context.Context, id, resolverID, status, fromStatus string) (model.TagSuggestion, error)
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
	exclude := viewerExclude(viewer)
	count, err := s.repo.CountPosts(ctx, canonical, includeNSFW, exclude)
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
	edges, err := s.repo.ListImplications(ctx)
	if err != nil {
		return model.TagPage{}, err
	}
	implies, impliedBy := implicationSides(canonical, edges)
	if !found && count == 0 && !isAlias && len(aliases) == 0 && len(implies) == 0 && len(impliedBy) == 0 {
		return model.TagPage{}, ErrNotFound
	}
	posts, err := s.repo.RecentPosts(ctx, canonical, includeNSFW, exclude)
	if err != nil {
		return model.TagPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	page := model.TagPage{
		Tag: tag, Aliases: aliases, Implies: implies, ImpliedBy: impliedBy,
		Category: "general", PostCount: count, Posts: posts,
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

// viewerExclude is the signed-in user's saved blacklist. A nil viewer
// and a nil tag list both become an empty slice.
func viewerExclude(viewer *model.User) []string {
	if viewer == nil || len(viewer.DefaultExcludeTags) == 0 {
		return []string{}
	}
	return viewer.DefaultExcludeTags
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

// AddImplication records that antecedent brings consequent along, and
// backfills posts that already have the antecedent, including every tag
// the consequent implies. Moderators and above.
func (s *Service) AddImplication(ctx context.Context, actor model.User, antecedentRaw, consequentRaw string) (model.TagPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.TagPage{}, ErrForbidden
	}
	antecedent, consequent, err := s.implicationPair(ctx, antecedentRaw, consequentRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	edges, err := s.repo.ListImplications(ctx)
	if err != nil {
		return model.TagPage{}, err
	}
	if implicationExists(edges, antecedent, consequent) {
		return model.TagPage{}, invalid("That implication already exists.")
	}
	// The edge is included before it is stored. Posts that have the
	// antecedent gain the consequent and every tag that consequent implies.
	pending := make([]model.TagImplication, len(edges)+1)
	copy(pending, edges)
	pending[len(edges)] = model.TagImplication{Antecedent: antecedent, Consequent: consequent}
	if implicationCycle(pending) {
		return model.TagPage{}, invalid("That implication would loop.")
	}
	backfill := make([]string, 0)
	for _, tag := range model.CloseImplications([]string{consequent}, pending) {
		if tag != antecedent {
			backfill = append(backfill, tag)
		}
	}
	if err := s.repo.CreateImplication(ctx, antecedent, consequent, actor.ID, backfill); err != nil {
		if strings.Contains(err.Error(), "23505") {
			return model.TagPage{}, invalid("That implication already exists.")
		}
		return model.TagPage{}, err
	}
	return s.page(ctx, &actor, antecedent)
}

// RemoveImplication drops the rule. Posts that already received the
// consequent keep it.
func (s *Service) RemoveImplication(ctx context.Context, actor model.User, antecedentRaw, consequentRaw string) (model.TagPage, error) {
	if !actor.Role.AtLeast(model.RoleModerator) {
		return model.TagPage{}, ErrForbidden
	}
	antecedent, consequent, err := s.implicationPair(ctx, antecedentRaw, consequentRaw)
	if err != nil {
		return model.TagPage{}, err
	}
	n, err := s.repo.DeleteImplication(ctx, antecedent, consequent)
	if err != nil {
		return model.TagPage{}, err
	}
	if n == 0 {
		return model.TagPage{}, ErrNotFound
	}
	return s.page(ctx, &actor, antecedent)
}

// Suggest records a proposal from any signed-in member. anchor is the tag
// in the URL. For an alias, anchor is the canonical name and other is the
// alternate. For an implication, anchor is the antecedent and other is the
// consequent. A second pending proposal for the same pair is rejected.
func (s *Service) Suggest(ctx context.Context, actor model.User, anchorRaw, kind, otherRaw, reason string) (model.TagSuggestion, error) {
	if actor.ID == "" {
		return model.TagSuggestion{}, ErrForbidden
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "alias" && kind != "implication" {
		return model.TagSuggestion{}, invalid("Kind must be alias or implication.")
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > maxSuggestionReason {
		return model.TagSuggestion{}, invalid("Reason must be 300 characters or fewer.")
	}
	anchor, err := oneTag(anchorRaw)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	other, err := oneTag(otherRaw)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	source, target := other, anchor
	if kind == "implication" {
		source, target = anchor, other
	}
	if source == target {
		if kind == "alias" {
			return model.TagSuggestion{}, invalid("A tag can't be an alias of itself.")
		}
		return model.TagSuggestion{}, invalid("A tag can't imply itself.")
	}
	created, err := s.repo.CreateSuggestion(ctx, kind, source, target, reason, actor.ID)
	if isUnique(err) {
		return model.TagSuggestion{}, invalid("That suggestion is already pending.")
	}
	if err != nil {
		return model.TagSuggestion{}, err
	}
	if created.AuthorUsername == "" {
		created.AuthorUsername = actor.Username
	}
	return created, nil
}

// ListSuggestions is the moderator queue. An empty status means pending.
func (s *Service) ListSuggestions(ctx context.Context, actor model.User, status string) ([]model.TagSuggestion, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return nil, ErrForbidden
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		status = "pending"
	}
	switch status {
	case "pending", "accepted", "rejected":
	default:
		return nil, invalid("Status must be pending, accepted, or rejected.")
	}
	items, err := s.repo.ListSuggestions(ctx, status)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.TagSuggestion{}
	}
	return items, nil
}

// AcceptSuggestion applies an alias or implication, then records the
// moderator. A pending or rejected row can be accepted. Accepting a row
// that is already accepted leaves it as it is. When the rule is already
// in place and points the same way, it is left alone. The tag rules and
// post updates match adding it directly.
func (s *Service) AcceptSuggestion(ctx context.Context, actor model.User, id string) (model.TagSuggestion, error) {
	current, err := s.suggestionForModerator(ctx, actor, id)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	if current.Status == "accepted" {
		return current, nil
	}
	if err := s.applySuggestion(ctx, actor, current); err != nil {
		return model.TagSuggestion{}, err
	}
	return s.recordSuggestion(ctx, actor, current, "accepted")
}

// RejectSuggestion records a no. A pending row is closed without changing
// tags. An accepted row is undone first: the alias or implication is
// removed, and posts that already received the tag keep it. Rejecting a
// row that is already rejected leaves it as it is.
func (s *Service) RejectSuggestion(ctx context.Context, actor model.User, id string) (model.TagSuggestion, error) {
	current, err := s.suggestionForModerator(ctx, actor, id)
	if err != nil {
		return model.TagSuggestion{}, err
	}
	if current.Status == "rejected" {
		return current, nil
	}
	if current.Status == "accepted" {
		if err := s.undoSuggestion(ctx, actor, current); err != nil {
			return model.TagSuggestion{}, err
		}
	}
	return s.recordSuggestion(ctx, actor, current, "rejected")
}

func (s *Service) suggestionForModerator(ctx context.Context, actor model.User, id string) (model.TagSuggestion, error) {
	if actor.ID == "" || !actor.Role.AtLeast(model.RoleModerator) {
		return model.TagSuggestion{}, ErrForbidden
	}
	current, err := s.repo.GetSuggestion(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.TagSuggestion{}, ErrNotFound
	}
	if err != nil {
		return model.TagSuggestion{}, err
	}
	return current, nil
}

func (s *Service) recordSuggestion(ctx context.Context, actor model.User, current model.TagSuggestion, status string) (model.TagSuggestion, error) {
	resolved, err := s.repo.ResolveSuggestion(ctx, current.ID, actor.ID, status, current.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return model.TagSuggestion{}, invalid("That suggestion changed. Refresh and try again.")
	}
	return resolved, err
}

// applySuggestion installs the proposed rule. A rule that already matches
// this suggestion is left in place so a changed decision can be recorded
// without failing.
func (s *Service) applySuggestion(ctx context.Context, actor model.User, item model.TagSuggestion) error {
	switch item.Kind {
	case "alias":
		target, ok, err := s.repo.AliasTarget(ctx, item.SourceTag)
		if err != nil {
			return err
		}
		if ok && target == item.TargetTag {
			return nil
		}
		_, err = s.AddAlias(ctx, actor, item.TargetTag, item.SourceTag)
		return err
	case "implication":
		edges, err := s.repo.ListImplications(ctx)
		if err != nil {
			return err
		}
		if implicationExists(edges, item.SourceTag, item.TargetTag) {
			return nil
		}
		_, err = s.AddImplication(ctx, actor, item.SourceTag, item.TargetTag)
		return err
	default:
		return invalid("Kind must be alias or implication.")
	}
}

// undoSuggestion drops the exact rule this suggestion applied. The stored
// names are used as they were accepted, and a rule that is already gone
// still counts as undone. Posts keep tags they already have.
func (s *Service) undoSuggestion(ctx context.Context, _ model.User, item model.TagSuggestion) error {
	switch item.Kind {
	case "alias":
		_, err := s.repo.DeleteAlias(ctx, item.SourceTag, item.TargetTag)
		return err
	case "implication":
		_, err := s.repo.DeleteImplication(ctx, item.SourceTag, item.TargetTag)
		return err
	default:
		return invalid("Kind must be alias or implication.")
	}
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func (s *Service) implicationPair(ctx context.Context, antecedentRaw, consequentRaw string) (string, string, error) {
	antecedent, err := oneTag(antecedentRaw)
	if err != nil {
		return "", "", err
	}
	consequent, err := oneTag(consequentRaw)
	if err != nil {
		return "", "", err
	}
	antecedent, _, err = s.canonical(ctx, antecedent)
	if err != nil {
		return "", "", err
	}
	consequent, _, err = s.canonical(ctx, consequent)
	if err != nil {
		return "", "", err
	}
	if antecedent == "" || consequent == "" {
		return "", "", invalid(tagMessage)
	}
	if antecedent == consequent {
		return "", "", invalid("A tag can't imply itself.")
	}
	return antecedent, consequent, nil
}

func implicationExists(edges []model.TagImplication, antecedent, consequent string) bool {
	for _, edge := range edges {
		if edge.Antecedent == antecedent && edge.Consequent == consequent {
			return true
		}
	}
	return false
}

func implicationSides(tag string, edges []model.TagImplication) (implies, impliedBy []string) {
	implies, impliedBy = []string{}, []string{}
	for _, edge := range edges {
		if edge.Antecedent == tag {
			implies = append(implies, edge.Consequent)
		}
		if edge.Consequent == tag {
			impliedBy = append(impliedBy, edge.Antecedent)
		}
	}
	sort.Strings(implies)
	sort.Strings(impliedBy)
	return implies, impliedBy
}

func implicationCycle(edges []model.TagImplication) bool {
	next := map[string][]string{}
	for _, edge := range edges {
		if edge.Antecedent == edge.Consequent {
			return true
		}
		next[edge.Antecedent] = append(next[edge.Antecedent], edge.Consequent)
	}
	color := map[string]int{}
	var visit func(string) bool
	visit = func(tag string) bool {
		switch color[tag] {
		case 1:
			return true
		case 2:
			return false
		}
		color[tag] = 1
		for _, child := range next[tag] {
			if visit(child) {
				return true
			}
		}
		color[tag] = 2
		return false
	}
	for tag := range next {
		if visit(tag) {
			return true
		}
	}
	return false
}
