package caldav

import (
	"math/rand"
	"net/http"
	"testing"
)

func TestBackoffUsesJitterWithinExpectedRange(t *testing.T) {
	t.Parallel()

	executor := newRetryExecutor(&http.Client{})
	executor.rng = rand.New(rand.NewSource(42))

	got := executor.backoff(2)
	max := retryBaseDelay * 4
	if got < 0 || got > max {
		t.Fatalf("backoff out of range: got %v max %v", got, max)
	}
	if got == max {
		t.Fatal("backoff should include jitter and not always hit max")
	}
}
