package board

import (
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// probeDimensions reads a post's pixel size: image headers directly,
// ffprobe (via a temp file, so non-faststart MP4s work) for the rest.
func (s *Service) probeDimensions(ctx context.Context, p model.Post) (int, int, error) {
	src, err := s.media.Open(ctx, p.StorageKey)
	if err != nil {
		return 0, 0, err
	}
	defer src.Close() //nolint:errcheck
	if !strings.HasPrefix(p.ContentType, "video/") {
		if cfg, _, err := image.DecodeConfig(src); err == nil && cfg.Width > 0 {
			return cfg.Width, cfg.Height, nil
		}
		// Header sniff failed (e.g. AVIF): reopen and let ffprobe try.
		src2, err := s.media.Open(ctx, p.StorageKey)
		if err != nil {
			return 0, 0, err
		}
		defer src2.Close() //nolint:errcheck
		return ffprobeDimensions(ctx, src2)
	}
	return ffprobeDimensions(ctx, src)
}

func ffprobeDimensions(ctx context.Context, src io.Reader) (int, int, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe unavailable: %w", err)
	}
	tmp, err := os.CreateTemp("", "kura-probe-*")
	if err != nil {
		return 0, 0, err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck
	if _, err := io.Copy(tmp, io.LimitReader(src, 256<<20)); err != nil {
		tmp.Close() //nolint:errcheck
		return 0, 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, 0, err
	}
	out, err := exec.CommandContext(ctx, ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", tmp.Name()).Output() //nolint:gosec // fixed args, temp path
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output")
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("unexpected ffprobe dimensions")
	}
	return w, h, nil
}

// ensureDimensions probes and persists the size the first time a post
// that lacks it is viewed. Best-effort: failures leave 0 ("unknown").
func (s *Service) ensureDimensions(ctx context.Context, p model.Post) model.Post {
	if p.Width > 0 && p.Height > 0 || s.media == nil {
		return p
	}
	if w, h, err := s.probeDimensions(ctx, p); err == nil {
		p.Width, p.Height = w, h
		_ = s.repo.SetPostDimensions(ctx, p.ID, w, h)
	}
	return p
}
