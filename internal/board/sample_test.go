package board

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSampleJPEGScalingBounds(t *testing.T) {
	cases := []struct {
		name  string
		w, h  int
		wantW int
		wantH int
	}{
		{name: "landscape", w: 2000, h: 1000, wantW: 1280, wantH: 640},
		{name: "portrait", w: 1000, h: 2000, wantW: 640, wantH: 1280},
		{name: "square", w: 2000, h: 2000, wantW: 1280, wantH: 1280},
		{name: "already small", w: 100, h: 50, wantW: 100, wantH: 50},
		{name: "already at the cap", w: 1280, h: 720, wantW: 1280, wantH: 720},
		{name: "wide strip", w: 5000, h: 1, wantW: 1280, wantH: 1},
		{name: "tall strip", w: 1, h: 5000, wantW: 1, wantH: 1280},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := sampleImageJPEG(bytes.NewReader(pngBytes(tc.w, tc.h)))
			require.NoError(t, err)
			require.True(t, bytes.HasPrefix(out, []byte{0xff, 0xd8}), "sample is a JPEG")
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
			require.NoError(t, err)
			require.Equal(t, tc.wantW, cfg.Width)
			require.Equal(t, tc.wantH, cfg.Height)
			require.LessOrEqual(t, cfg.Width, sampleMaxEdge)
			require.LessOrEqual(t, cfg.Height, sampleMaxEdge)
			require.GreaterOrEqual(t, cfg.Width, 1)
			require.GreaterOrEqual(t, cfg.Height, 1)
			if tc.w <= sampleMaxEdge && tc.h <= sampleMaxEdge {
				require.Equal(t, tc.w, cfg.Width)
				require.Equal(t, tc.h, cfg.Height)
			}
		})
	}
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	if w > 1 && h > 1 {
		img.Set(w-1, h-1, color.RGBA{G: 180, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
