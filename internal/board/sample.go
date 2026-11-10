package board

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"os/exec"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"golang.org/x/image/draw"
)

const (
	sampleMaxEdge     = 1280
	sampleJPEGQuality = 82
)

// grabStill reads one video or undecodable frame. Tests replace it.
var grabStill = ffmpegStill

// OpenSample opens the stored medium JPEG. It does not generate one;
// that is the post.derive job. Callers fall back to the original.
func (s *Service) OpenSample(ctx context.Context, p model.Post) (io.ReadCloser, error) {
	if p.SampleKey == "" || s.media == nil {
		return nil, ErrNotFound
	}
	return s.media.Open(ctx, p.SampleKey)
}

func (s *Service) storeSample(ctx context.Context, p model.Post) error {
	if p.SampleKey != "" {
		return nil
	}
	jpegBytes, err := s.renderSample(ctx, p)
	if err != nil {
		return err
	}
	if len(jpegBytes) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Put stores the encoder bytes unchanged under an opaque key.
	key, err := s.media.Put(ctx, bytes.NewReader(jpegBytes))
	if err != nil {
		return err
	}
	return s.repo.SetPostSample(ctx, p.ID, key)
}

// renderSample returns a JPEG, (nil, nil) when no frame can be made,
// or an error when storage failed and the job should retry.
func (s *Service) renderSample(ctx context.Context, p model.Post) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	src, err := s.media.Open(ctx, p.StorageKey)
	if err != nil {
		return nil, err
	}
	defer src.Close() //nolint:errcheck
	if strings.HasPrefix(p.ContentType, "video/") {
		return finishStill(ctx, src)
	}
	img, _, decErr := image.Decode(src)
	if decErr == nil {
		return encodeSampleJPEG(img)
	}
	src2, openErr := s.media.Open(ctx, p.StorageKey)
	if openErr != nil {
		return nil, openErr
	}
	defer src2.Close() //nolint:errcheck
	return finishStill(ctx, src2)
}

func finishStill(ctx context.Context, src io.Reader) ([]byte, error) {
	img, err := grabStill(ctx, src)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || img == nil {
		return nil, nil
	}
	return encodeSampleJPEG(img)
}

func sampleImageJPEG(src io.Reader) ([]byte, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, err
	}
	return encodeSampleJPEG(img)
}

func encodeSampleJPEG(img image.Image) ([]byte, error) {
	b := img.Bounds()
	nw, nh := fitLongestEdge(b.Dx(), b.Dy(), sampleMaxEdge)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: sampleJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fitLongestEdge caps the longer side at maxEdge and does not upscale.
func fitLongestEdge(w, h, maxEdge int) (int, int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if maxEdge < 1 {
		maxEdge = 1
	}
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		nh := int(math.Round(float64(h) * float64(maxEdge) / float64(w)))
		if nh < 1 {
			nh = 1
		}
		if nh > maxEdge {
			nh = maxEdge
		}
		return maxEdge, nh
	}
	nw := int(math.Round(float64(w) * float64(maxEdge) / float64(h)))
	if nw < 1 {
		nw = 1
	}
	if nw > maxEdge {
		nw = maxEdge
	}
	return nw, maxEdge
}

// ffmpegStill grabs one frame the same way thumbnails do: ffmpeg reads
// stdin and writes one mjpeg frame. The scale keeps the longest edge at
// most sampleMaxEdge. The caller re-encodes at the sample quality.
func ffmpegStill(ctx context.Context, src io.Reader) (image.Image, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg unavailable: %w", err)
	}
	vf := fmt.Sprintf("scale='if(gte(iw,ih),min(%d,iw),-2)':'if(gt(ih,iw),min(%d,ih),-2)'", sampleMaxEdge, sampleMaxEdge)
	cmd := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-i", "pipe:0", "-frames:v", "1", "-vf", vf, "-f", "mjpeg", "pipe:1") //nolint:gosec // fixed args, media via stdin
	cmd.Stdin = src
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg produced no frame")
	}
	img, err := jpeg.Decode(&out)
	if err != nil {
		return nil, err
	}
	return img, nil
}
