package fraud

import (
	"testing"
	"time"
)

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) returned error: %v", value, err)
	}

	return parsed
}
