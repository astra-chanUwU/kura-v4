package board

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const sourceURLMessage = "Source link must be an http or https URL."

func invalid(msg string) error { return &ValidationError{Message: msg} }

// Upload stores an original under its content hash (identical bytes
// are stored once) and returns the storage key and SHA-256.
func (s *Service) Upload(ctx context.Context, r io.Reader) (string, string, error) {
	return s.media.PutHashed(ctx, r)
}

// Discard removes a stored original, e.g. after a failed post create.
func (s *Service) Discard(ctx context.Context, key string) error {
	return s.media.Delete(ctx, key)
}

// OpenMedia opens the original bytes for a post.
func (s *Service) OpenMedia(ctx context.Context, p model.Post) (io.ReadCloser, error) {
	return s.media.Open(ctx, p.StorageKey)
}

func (s *Service) Create(ctx context.Context, p model.Post, sha string) (model.Post, error) {
	sourceURL, err := normalizeSourceURL(p.SourceURL)
	if err != nil {
		return model.Post{}, err
	}
	p.SourceURL = sourceURL
	p.Tags, err = s.applyAliases(ctx, NormalizeTags(p.Tags))
	if err != nil {
		return model.Post{}, err
	}
	p.Rating = model.NormalizeRating(p.Rating)
	var artistTag string
	p.Tags, artistTag, err = s.attachArtist(ctx, "", p.Tags, p.Artist)
	if err != nil {
		return model.Post{}, err
	}
	p.Tags, artistTag, err = s.rewriteTags(ctx, p.Tags, artistTag)
	if err != nil {
		return model.Post{}, err
	}
	if (p.Width <= 0 || p.Height <= 0) && s.media != nil && p.StorageKey != "" {
		if w, h, err := s.probeDimensions(ctx, p); err == nil {
			p.Width, p.Height = w, h
		}
	}
	// CreatePost stores the row and its tags in one transaction.
	created, err := s.repo.CreatePost(ctx, p, sha)
	if err != nil {
		return model.Post{}, err
	}
	// The post is already stored. Queue derivatives before the artist page
	// so a wiki-row failure cannot leave the file without a derive job.
	s.enqueueDerive(ctx, created.ID)
	if err := s.ensureArtistPage(ctx, artistTag); err != nil {
		return model.Post{}, err
	}
	return created, nil
}

// GetBySHA finds the existing post for identical bytes, if any.
func (s *Service) GetBySHA(ctx context.Context, sha string) (model.Post, error) {
	return s.repo.GetPostBySHA(ctx, sha)
}

// RecordCreation writes the post.created history row for a new post.
func (s *Service) RecordCreation(ctx context.Context, actor model.User, post model.Post) {
	empty := model.PostSnapshot{Artist: "", Tags: []string{}, Rating: "sfw", Hidden: false}
	_, err := s.repo.InsertHistory(ctx, model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID,
		ActorUsername: actor.Username, Action: "post.created", Before: model.MarshalPostSnapshot(empty), After: model.MarshalPostSnapshot(model.SnapshotOf(post))})
	if err != nil {
		log := s.log
		if log == nil {
			log = slog.Default()
		}
		log.Error("record post.created", "post_id", post.ID, "error", err)
	}
}

func normalizeSourceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if utf8.RuneCountInString(raw) > 300 {
		return "", invalid(sourceURLMessage)
	}
	parsed, err := url.Parse(raw)
	if err != nil || !httpURL(parsed) {
		return "", invalid(sourceURLMessage)
	}
	return raw, nil
}

func httpURL(u *url.URL) bool {
	if u == nil || u.Host == "" || u.Opaque != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}
