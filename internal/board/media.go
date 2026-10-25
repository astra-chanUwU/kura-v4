package board

import (
	"net/http"
	"path/filepath"
	"strings"
)

// SniffContentType identifies an upload/seed file. http.DetectContentType
// alone misses some MP4 files (and WebM entirely), which used to leave
// Twitter "GIFs" — really MP4s — stored as application/octet-stream and
// rendered as broken images. Container magic wins, then the extension.
func SniffContentType(head []byte, filename string) string {
	ct := http.DetectContentType(head)
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = ct[:idx]
	}
	if ct != "application/octet-stream" {
		return ct
	}
	if len(head) >= 12 && string(head[4:8]) == "ftyp" {
		if brand := string(head[8:12]); brand == "avif" || brand == "avis" {
			return "image/avif"
		}
		return "video/mp4"
	}
	if len(head) >= 4 && head[0] == 0x1a && head[1] == 0x45 && head[2] == 0xdf && head[3] == 0xa3 {
		return "video/webm"
	}
	if len(head) >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WEBP" {
		return "image/webp"
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	}
	return ct
}

var allowedMedia = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true, "image/avif": true,
	"video/mp4": true, "video/webm": true, "video/quicktime": true,
}

// IsAllowedMedia reports whether a sniffed type can be posted.
func IsAllowedMedia(contentType string) bool { return allowedMedia[contentType] }
