package caldav

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

const (
	maxRetryAttempts = 3

	timeoutPROPFIND = 10 * time.Second
	timeoutREPORT   = 20 * time.Second
	timeoutGET      = 15 * time.Second
	timeoutPUT      = 15 * time.Second
	timeoutDELETE   = 10 * time.Second
	timeoutMKCAL    = 15 * time.Second
	timeoutFullScan = 30 * time.Second

	retryBaseDelay = 150 * time.Millisecond
)

var (
	// ErrPreconditionFailed indicates a CalDAV If-Match precondition conflict.
	ErrPreconditionFailed = errors.New("caldav precondition failed")
)

type operationPolicy struct {
	timeout      time.Duration
	retryEnabled bool
}

type retryExecutor struct {
	httpClient *http.Client
	rng        *rand.Rand
	sleep      func(time.Duration)
}

func newRetryExecutor(httpClient *http.Client) *retryExecutor {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &retryExecutor{
		httpClient: httpClient,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
		sleep:      time.Sleep,
	}
}

func (e *retryExecutor) do(ctx context.Context, policy operationPolicy, buildRequest func(context.Context) (*http.Request, error)) (*http.Response, error) {
	attempts := 1
	if policy.retryEnabled {
		attempts = maxRetryAttempts
	}

	for attempt := 0; attempt < attempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, policy.timeout)
		request, err := buildRequest(attemptCtx)
		if err != nil {
			cancel()
			return nil, err
		}

		response, doErr := e.httpClient.Do(request)
		if doErr == nil {
			if response.StatusCode == http.StatusPreconditionFailed {
				cancel()
				response.Body.Close()
				return nil, ErrPreconditionFailed
			}
			if shouldRetryStatus(response.StatusCode) && attempt < attempts-1 {
				cancel()
				response.Body.Close()
				e.sleep(e.backoff(attempt))
				continue
			}
			cancel()
			return response, nil
		}

		cancel()
		if !policy.retryEnabled || attempt >= attempts-1 || !isRetriableError(doErr) {
			return nil, doErr
		}
		e.sleep(e.backoff(attempt))
	}

	return nil, fmt.Errorf("request failed after retries")
}

func isRetriableError(err error) bool {
	return !errors.Is(err, context.Canceled)
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode >= http.StatusInternalServerError || statusCode == http.StatusTooManyRequests
}

func (e *retryExecutor) backoff(attempt int) time.Duration {
	maxDelay := retryBaseDelay * time.Duration(1<<attempt)
	if maxDelay <= 0 {
		return retryBaseDelay
	}
	return time.Duration(e.rng.Int63n(maxDelay.Nanoseconds() + 1))
}
