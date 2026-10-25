package board

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif" // register GIF decoding for image.Decode
	"image/jpeg"
	_ "image/png" // register PNG decoding for image.Decode
	"io"
	"os/exec"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoding for image.Decode
)

const thumbMaxDim = 480

// Thumbnail returns a small JPEG preview for a post, generating and
// caching it in media storage on first use. When generation is not
// possible (e.g. AVIF, or ffmpeg missing for a video), it falls back to
// the original bytes so the tile still renders.
func (s *Service) Thumbnail(ctx context.Context, p model.Post) (io.ReadCloser, string, error) {
	if p.ThumbnailKey != "" {
		if file, err := s.media.Open(ctx, p.ThumbnailKey); err == nil {
			return file, "image/jpeg", nil
		}
	}
	if key, err := s.MakeThumbnail(ctx, p); err == nil && key != "" {
		if file, err := s.media.Open(ctx, key); err == nil {
			return file, "image/jpeg", nil
		}
	}
	file, err := s.media.Open(ctx, p.StorageKey)
	if err != nil {
		return nil, "", err
	}
	return file, p.ContentType, nil
}

// MakeThumbnail generates the preview now and records its storage key.
// It returns "" (and no error the caller must surface) when the format
// cannot be thumbnailed; previews are best-effort.
func (s *Service) MakeThumbnail(ctx context.Context, p model.Post) (string, error) {
	if p.ThumbnailKey != "" {
		return p.ThumbnailKey, nil
	}
	src, err := s.media.Open(ctx, p.StorageKey)
	if err != nil {
		return "", err
	}
	defer src.Close() //nolint:errcheck
	var jpegBytes []byte
	if strings.HasPrefix(p.ContentType, "video/") {
		jpegBytes, err = ffmpegFrameJPEG(ctx, src)
	} else {
		jpegBytes, err = imageThumbnailJPEG(src)
		if err != nil {
			// Formats Go cannot decode (animated WebP, AVIF, damaged
			// GIFs) may still yield a frame via ffmpeg.
			if src2, openErr := s.media.Open(ctx, p.StorageKey); openErr == nil {
				defer src2.Close() //nolint:errcheck
				jpegBytes, err = ffmpegFrameJPEG(ctx, src2)
			}
		}
	}
	if err != nil || len(jpegBytes) == 0 {
		return "", nil // best-effort: caller falls back to the original
	}
	key, err := s.media.Put(ctx, bytes.NewReader(jpegBytes))
	if err != nil {
		return "", err
	}
	if err := s.repo.SetPostThumbnail(ctx, p.ID, key); err != nil {
		return "", err
	}
	return key, nil
}

func imageThumbnailJPEG(src io.Reader) ([]byte, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := 1.0
	if w > thumbMaxDim || h > thumbMaxDim {
		if w > h {
			scale = float64(thumbMaxDim) / float64(w)
		} else {
			scale = float64(thumbMaxDim) / float64(h)
		}
	}
	nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ffmpegFrameJPEG grabs the first frame with ffmpeg when it is available.
func ffmpegFrameJPEG(ctx context.Context, src io.Reader) ([]byte, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg unavailable: %w", err)
	}
	cmd := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-i", "pipe:0", "-frames:v", "1", "-vf", "scale=480:-2", "-f", "mjpeg", "pipe:1") //nolint:gosec // fixed args, media via stdin
	cmd.Stdin = src
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg produced no frame")
	}
	return out.Bytes(), nil
}
