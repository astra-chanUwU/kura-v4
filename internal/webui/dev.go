//go:build dev

package webui

import "io/fs"

// Vite serves the UI in development; backend tests need no frontend build.
func Files() fs.FS { return nil }
