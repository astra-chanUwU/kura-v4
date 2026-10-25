package board

import "testing"

func TestParseSeedNameTwitter(t *testing.T) {
	meta := ParseSeedName("twitter gif Serraphyll 2108932354891030955.mp4")
	if meta.Artist != "Serraphyll" || meta.Source != "twitter" || meta.SourceID != "2108932354891030955" {
		t.Fatalf("got %+v", meta)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "gif" || meta.Tags[1] != "twitter" {
		t.Fatalf("tags: %v", meta.Tags)
	}
}

func TestParseSeedNameUnknown(t *testing.T) {
	meta := ParseSeedName("HUJzHZma8AALDEC.jpg")
	if meta.Artist != "" || meta.Source != "" || meta.SourceID != "" {
		t.Fatalf("got %+v", meta)
	}
}

func TestNormalizeTags(t *testing.T) {
	got := NormalizeTags([]string{"anime girl white_hair", "yuri, wholesome", "Cute!", "cute"})
	want := []string{"anime", "cute", "girl", "white_hair", "wholesome", "yuri"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
