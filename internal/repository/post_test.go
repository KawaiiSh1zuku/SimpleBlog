package repository

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	if got, want := escapeLikePattern(`100%_match\path`), `100\%\_match\\path`; got != want {
		t.Fatalf("escapeLikePattern() = %q, want %q", got, want)
	}
}
