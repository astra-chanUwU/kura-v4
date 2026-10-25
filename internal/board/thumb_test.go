package board

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

func TestImageThumbnailJPEGFromGIFAndPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 960, 540))
	for y := 0; y < 540; y++ {
		for x := 0; x < 960; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 80, B: 200, A: 255})
		}
	}
	var gifBuf, pngBuf bytes.Buffer
	if err := gif.Encode(&gifBuf, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string][]byte{"gif": gifBuf.Bytes(), "png": pngBuf.Bytes()} {
		out, err := imageThumbnailJPEG(bytes.NewReader(src))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(out) < 2 || out[0] != 0xff || out[1] != 0xd8 {
			t.Fatalf("%s: output is not JPEG (%d bytes)", name, len(out))
		}
		decoded, _, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s: redecode: %v", name, err)
		}
		if decoded.Bounds().Dx() != thumbMaxDim {
			t.Fatalf("%s: width %d, want %d", name, decoded.Bounds().Dx(), thumbMaxDim)
		}
	}
}
