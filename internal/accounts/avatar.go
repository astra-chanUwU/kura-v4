package accounts

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif" // register GIF decoding for image.Decode
	"image/jpeg"
	_ "image/png" // register PNG decoding for image.Decode
	"os/exec"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoding for image.Decode
)

// avatarMaxDim caps a stored profile picture's width and height. Avatars
// render at 24–88px in the UI, so 256px stays sharp on retina displays
// while keeping every served file to a few tens of KB — the uploaded
// original (up to 8 MB, thousands of px) is never stored or served.
const avatarMaxDim = 256

// maxAvatarServeBytes is the largest small JPEG served as-is. A tinier
// picture carrying more than this is mostly embedded metadata, so it is
// re-encoded (which strips it) instead of being passed through.
const maxAvatarServeBytes = 1 << 20

// resizedAvatarJPEG decodes picture bytes and re-encodes them as a JPEG
// no larger than avatarMaxDim in either dimension, mirroring how post
// thumbnails are made. Formats Go cannot decode (AVIF, animated WebP)
// fall back to ffmpeg, which production ships. An error means the bytes
// are not a readable picture at all; callers must not store or serve
// the original as an avatar in that case.
func resizedAvatarJPEG(ctx context.Context, data []byte) ([]byte, error) {
	if out, err := avatarJPEG(data); err == nil {
		return out, nil
	}
	return ffmpegAvatarJPEG(ctx, data)
}

func avatarJPEG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := 1.0
	if w > avatarMaxDim || h > avatarMaxDim {
		if w > h {
			scale = float64(avatarMaxDim) / float64(w)
		} else {
			scale = float64(avatarMaxDim) / float64(h)
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

func ffmpegAvatarJPEG(ctx context.Context, data []byte) ([]byte, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg unavailable: %w", err)
	}
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", avatarMaxDim, avatarMaxDim)
	cmd := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-i", "pipe:0", "-frames:v", "1", "-vf", vf, "-f", "mjpeg", "pipe:1") //nolint:gosec // fixed args, media via stdin
	cmd.Stdin = bytes.NewReader(data)
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

// avatarNeedsResize reports whether stored bytes should be replaced by a
// resized JPEG: anything that is not already a small, reasonably-sized
// JPEG. Undecodable bytes report true (conversion is attempted, and the
// caller falls back to the original if it fails).
func avatarNeedsResize(data []byte, contentType string) bool {
	if contentType != "image/jpeg" || len(data) > maxAvatarServeBytes {
		return true
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return true
	}
	return format != "jpeg" || cfg.Width > avatarMaxDim || cfg.Height > avatarMaxDim
}
