package board

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

// KindPostDerive is the background job that fills a new post's
// thumbnail, dimensions, and medium sample.
const KindPostDerive = "post.derive"

// JobQueue accepts work after a post is stored. The board does not
// import the jobs package; app wires the repository.
type JobQueue interface {
	Enqueue(ctx context.Context, kind string, payload json.RawMessage) (int64, error)
}

type derivePayload struct {
	PostID string `json:"post_id"`
}

// SetJobQueue enables post.derive after Create. A nil queue leaves
// creates without background work, which tests and audit rely on.
func (s *Service) SetJobQueue(q JobQueue, log *slog.Logger) {
	s.jobs = q
	if log != nil {
		s.log = log
	}
}

func (s *Service) enqueueDerive(ctx context.Context, postID string) {
	if s.jobs == nil || postID == "" {
		return
	}
	payload, err := json.Marshal(derivePayload{PostID: postID})
	if err != nil {
		s.logEnqueue(postID, err)
		return
	}
	if _, err := s.jobs.Enqueue(ctx, KindPostDerive, payload); err != nil {
		s.logEnqueue(postID, err)
	}
}

func (s *Service) logEnqueue(postID string, err error) {
	log := s.log
	if log == nil {
		log = slog.Default()
	}
	log.Error("enqueue post.derive", "post_id", postID, "error", err)
}

// DerivePost builds derivatives for one post. It is safe to run more
// than once: a thumbnail, sample, or size that is already stored is
// left alone. Storage errors are returned so the job retries. A frame
// that cannot be decoded leaves sample_key empty and returns nil so
// the attempt is not burned again. ctx is honored between steps.
// Thumbnail() and ensureDimensions() stay in place for posts this job
// has not finished.
func (s *Service) DerivePost(ctx context.Context, payload json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var body derivePayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return fmt.Errorf("post.derive payload: %w", err)
	}
	if _, err := uuid.Parse(body.PostID); err != nil {
		return fmt.Errorf("post.derive post_id: %w", err)
	}
	p, err := s.repo.GetPost(ctx, body.PostID)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if s.media == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.ThumbnailKey == "" {
		if _, err := s.MakeThumbnail(ctx, p); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.Width <= 0 || p.Height <= 0 {
		if err := s.persistDimensions(ctx, p); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.storeSample(ctx, p)
}

func (s *Service) persistDimensions(ctx context.Context, p model.Post) error {
	w, h, err := s.probeDimensions(ctx, p)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	return s.repo.SetPostDimensions(ctx, p.ID, w, h)
}
