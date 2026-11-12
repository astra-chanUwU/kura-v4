package dao

import "testing"

func TestSubstringLike(t *testing.T) {
	if got := substringLike(""); got != "" {
		t.Fatalf("empty query = %q", got)
	}
	if got := substringLike("ada"); got != "%ada%" {
		t.Fatalf("plain query = %q", got)
	}
	if got := substringLike(`100%_sure`); got != `%100\%\_sure%` {
		t.Fatalf("metacharacters = %q", got)
	}
	if got := substringLike(`a\b`); got != `%a\\b%` {
		t.Fatalf("backslash = %q", got)
	}
}
