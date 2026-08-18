package utils

import "testing"

func TestMapLogLevelIsCaseInsensitive(t *testing.T) {
	if got := MapLogLevel("debug"); got != -4 {
		t.Fatalf("MapLogLevel(debug) = %d, want -4", got)
	}
	if got := MapLogLevel("WARN"); got != 4 {
		t.Fatalf("MapLogLevel(WARN) = %d, want 4", got)
	}
}
