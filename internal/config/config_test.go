package config

import (
	"strings"
	"testing"
)

func TestRequiredDatabaseURL(t *testing.T) {
	_, err := Read(func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatal("missing database URL must be actionable")
	}
}
func TestPublicURLRejectsCredentialsWithoutEchoingThem(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://localhost/kura", "KURA_PUBLIC_URL": "https://person:secret@example.com"}
	_, err := Read(func(k string) string { return values[k] })
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("invalid URL must fail without exposing credentials")
	}
}
