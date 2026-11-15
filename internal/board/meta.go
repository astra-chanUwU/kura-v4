package board

import (
	"context"
	"sort"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// SetMeta replaces a post's artist, tags, rating, and source link
// (communal tagging: any signed-in user may edit; the controller enforces sign-in).
func (s *Service) SetMeta(ctx context.Context, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	return s.setMeta(ctx, nil, id, artist, tags, rating, sourceURL)
}

// SetMetaAs is SetMeta with an actor: a changed edit is written to the
// post's change history in the same transaction, so it can be rolled back.
func (s *Service) SetMetaAs(ctx context.Context, actor *model.User, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	return s.setMeta(ctx, actor, id, artist, tags, rating, sourceURL)
}

func (s *Service) setMeta(ctx context.Context, actor *model.User, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	sourceURL, err := normalizeSourceURL(sourceURL)
	if err != nil {
		return model.Post{}, err
	}
	post, err := s.Get(ctx, id)
	if err != nil {
		return model.Post{}, err
	}
	before := model.SnapshotOf(post)
	if rating == "sfw" || rating == "nsfw" {
		post.Rating = model.NormalizeRating(rating)
	}
	artist = strings.TrimSpace(artist)
	if len([]rune(artist)) > 80 {
		artist = string([]rune(artist)[:80])
	}
	previousArtist := post.Artist
	post.Artist = artist
	post.SourceURL = sourceURL
	post.Tags, err = s.applyAliases(ctx, NormalizeTags(tags))
	if err != nil {
		return model.Post{}, err
	}
	var artistTag string
	post.Tags, artistTag, err = s.attachArtist(ctx, previousArtist, post.Tags, post.Artist)
	if err != nil {
		return model.Post{}, err
	}
	post.Tags, artistTag, err = s.rewriteTags(ctx, post.Tags, artistTag)
	if err != nil {
		return model.Post{}, err
	}
	after := model.SnapshotOf(post)
	changed := before.Artist != after.Artist || before.Rating != after.Rating || before.SourceURL != after.SourceURL || !equalTags(before.Tags, after.Tags)
	if actor != nil && changed {
		entry := model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
			Action: "post.meta_updated", Before: model.MarshalPostSnapshot(before), After: model.MarshalPostSnapshot(after)}
		if err := s.repo.ReplacePostMetaAudited(ctx, post, entry); err != nil {
			return model.Post{}, err
		}
		if err := s.ensureArtistPage(ctx, artistTag); err != nil {
			return model.Post{}, err
		}
		return post, nil
	}
	if err := s.repo.ReplacePostMeta(ctx, post); err != nil {
		return model.Post{}, err
	}
	if err := s.ensureArtistPage(ctx, artistTag); err != nil {
		return model.Post{}, err
	}
	return post, nil
}

// rewriteTags is the tag closure for a save. Aliases and implications are
// one relation: an implied name that is itself an alias is stored as the
// canonical tag, and that canonical tag's own implications are included.
// artistTag is the credit attached just before this call. It stays on the
// post even when the walk starts from the member's other tags.
func (s *Service) rewriteTags(ctx context.Context, tags []string, artistTag string) ([]string, string, error) {
	if tags == nil {
		tags = []string{}
	}
	edges, err := s.repo.ListImplications(ctx)
	if err != nil {
		return nil, "", err
	}
	if len(edges) == 0 {
		return tags, artistTag, nil
	}
	names := make([]string, 0, len(tags)+len(edges)*2+1)
	names = append(names, tags...)
	if artistTag != "" {
		names = append(names, artistTag)
	}
	for _, edge := range edges {
		names = append(names, edge.Antecedent, edge.Consequent)
	}
	mapped, err := s.repo.TagAliasTargets(ctx, names)
	if err != nil {
		return nil, "", err
	}
	if artistTag != "" {
		artistTag = aliasOrToken(artistTag, mapped)
	}
	input := tags
	if artistTag != "" && !tagPresent(input, artistTag) {
		input = append(append([]string{}, tags...), artistTag)
	}
	out := closeWithAliases(input, mapped, edges)
	if artistTag != "" && !tagPresent(out, artistTag) {
		out = append(out, artistTag)
		sort.Strings(out)
	}
	if out == nil {
		out = []string{}
	}
	return out, artistTag, nil
}

// closeWithAliases adds every implied tag and stores alias names as their
// canonical tags. An implication written on an alias follows the canonical
// tag, so grey_hair -> long_hair still applies after grey_hair becomes gray_hair.
func closeWithAliases(tags []string, aliases map[string]string, edges []model.TagImplication) []string {
	canon := func(tag string) string {
		return aliasOrToken(tag, aliases)
	}
	next := map[string][]string{}
	for _, edge := range edges {
		from := canon(edge.Antecedent)
		to := canon(edge.Consequent)
		if from == "" || to == "" || from == to {
			continue
		}
		next[from] = append(next[from], to)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	var walk func(string)
	walk = func(tag string) {
		tag = canon(tag)
		if tag == "" || seen[tag] {
			return
		}
		seen[tag] = true
		out = append(out, tag)
		for _, child := range next[tag] {
			walk(child)
		}
	}
	for _, tag := range tags {
		walk(tag)
	}
	sort.Strings(out)
	return out
}

// applyAliases replaces alternate names with the canonical tag and drops
// duplicates. Tags that are not aliases stay as they are.
func (s *Service) applyAliases(ctx context.Context, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return tags, nil
	}
	mapped, err := s.repo.TagAliasTargets(ctx, tags)
	if err != nil || len(mapped) == 0 {
		if tags == nil {
			tags = []string{}
		}
		return tags, err
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if next, ok := mapped[tag]; ok && next != "" {
			tag = next
		}
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out, nil
}

// attachArtist stores the artist credit as a tag after alias resolution.
// A moderator alias is rewritten to its canonical tag here, before save.
// previousArtist's token, and that token's canonical alias when it differs,
// leave the set. The returned tag is the one stored, or "" when the credit
// has no token. A blank artist only removes the previous tag.
func (s *Service) attachArtist(ctx context.Context, previousArtist string, tags []string, artist string) ([]string, string, error) {
	prevToken := model.ArtistToken(previousArtist)
	nextToken := model.ArtistToken(artist)
	mapped, err := s.artistAliases(ctx, prevToken, nextToken)
	if err != nil {
		return nil, "", err
	}
	prevCanonical := aliasOrToken(prevToken, mapped)
	nextCanonical := aliasOrToken(nextToken, mapped)
	out := model.ApplyArtistTag(previousArtist, tags, artist)
	if nextCanonical != "" && nextCanonical != nextToken {
		out = exchangeTag(out, nextToken, nextCanonical)
	}
	if prevCanonical != "" && prevCanonical != prevToken && prevCanonical != nextCanonical {
		out = dropTag(out, prevCanonical)
	}
	if nextCanonical != "" && !tagPresent(out, nextCanonical) {
		out = append(out, nextCanonical)
		sort.Strings(out)
	}
	if out == nil {
		out = []string{}
	}
	return out, nextCanonical, nil
}

func (s *Service) artistAliases(ctx context.Context, tokens ...string) (map[string]string, error) {
	lookup := make([]string, 0, len(tokens))
	seen := map[string]bool{}
	for _, token := range tokens {
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		lookup = append(lookup, token)
	}
	if len(lookup) == 0 {
		return map[string]string{}, nil
	}
	return s.repo.TagAliasTargets(ctx, lookup)
}

func aliasOrToken(token string, mapped map[string]string) string {
	if token == "" {
		return ""
	}
	if next, ok := mapped[token]; ok && next != "" {
		return next
	}
	return token
}

func exchangeTag(tags []string, raw, canonical string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		if tag == "" || tag == raw || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	if canonical != "" && !seen[canonical] {
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out
}

func dropTag(tags []string, drop string) []string {
	if drop == "" {
		if tags == nil {
			return []string{}
		}
		return tags
	}
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		if tag == "" || tag == drop || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func tagPresent(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func (s *Service) ensureArtistPage(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}
	return s.repo.EnsureArtistPage(ctx, tag)
}

func equalTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// WithLastTagger sets the username of the latest tag save. A post nobody
// has retagged keeps an empty name. Missing history is not an error.
func (s *Service) WithLastTagger(ctx context.Context, p model.Post) (model.Post, error) {
	name, err := s.repo.LastMetaEditor(ctx, p.ID)
	if err != nil {
		return model.Post{}, err
	}
	if name != "" {
		p.LastTaggerUsername = name
	}
	return p, nil
}
