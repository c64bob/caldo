package caldav

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
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

func TestDoCancelsAttemptContextWhenResponseBodyCloses(t *testing.T) {
	t.Parallel()

	transport := &capturingTransport{}
	executor := newRetryExecutor(&http.Client{Transport: transport})

	response, err := executor.do(
		context.Background(),
		operationPolicy{timeout: time.Second, retryEnabled: false},
		func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
		},
	)
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	select {
	case <-transport.requestCtx.Done():
		t.Fatal("request context canceled before body close")
	default:
	}

	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}

	select {
	case <-transport.requestCtx.Done():
	default:
		t.Fatal("request context should be canceled after body close")
	}
}

type capturingTransport struct {
	requestCtx context.Context
}

func (t *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.requestCtx = req.Context()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}, nil
}
