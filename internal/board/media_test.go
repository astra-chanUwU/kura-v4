package board

import "testing"

func TestSniffContentTypeMP4MislabeledAsGIF(t *testing.T) {
	// Twitter "GIF": an MP4 (ftypisom) that net/http reports generically.
	head := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0x00, 0x00, 0x00, 0x01}
	if got := SniffContentType(head, "twitter gif 2107863409937244558.gif"); got != "video/mp4" {
		t.Fatalf("got %q, want video/mp4", got)
	}
}

func TestSniffContentTypeWebM(t *testing.T) {
	head := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x93, 0x42, 0x82, 0x88}
	if got := SniffContentType(head, "clip.webm"); got != "video/webm" {
		t.Fatalf("got %q, want video/webm", got)
	}
}

func TestSniffContentTypeStillDetectsJPEG(t *testing.T) {
	head := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if got := SniffContentType(head, "photo.bin"); got != "image/jpeg" {
		t.Fatalf("got %q, want image/jpeg", got)
	}
}
