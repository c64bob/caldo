package shutdown

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

type stubScheduler struct {
	stop func(ctx context.Context) error
}

func (s *stubScheduler) Stop(ctx context.Context) error {
	if s.stop != nil {
		return s.stop(ctx)
	}
	return nil
}

func TestHandleRegistersSIGINTAndSIGTERM(t *testing.T) {
	t.Parallel()

	coordinator := NewCoordinator(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)), nil, time.Second)

	baseCtx, cancelSignal := context.WithCancel(context.Background())
	t.Cleanup(cancelSignal)

	var gotSignals []any
	coordinator.notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		gotSignals = append(gotSignals, any(signals[0]), any(signals[1]))
		return baseCtx, func() {}
	}

	finished := make(chan int, 1)
	go func() {
		finished <- coordinator.Handle(context.Background(), nil)
	}()

	cancelSignal()

	if got := <-finished; got != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", got, ExitCodeSuccess)
	}

	if len(gotSignals) != 2 {
		t.Fatalf("unexpected signal registrations: got %d want 2", len(gotSignals))
	}
	if gotSignals[0] != syscall.SIGTERM || gotSignals[1] != syscall.SIGINT {
		t.Fatalf("unexpected signals: got %v", gotSignals)
	}
}

func TestHandleGracefulShutdownReturnsZero(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	requestStarted := make(chan struct{})
	allowFinish := make(chan struct{})

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-allowFinish
		_, _ = w.Write([]byte("ok"))
	})}

	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		_ = listener.Close()
	})

	requestErr := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			requestErr <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			requestErr <- errors.New("unexpected status code")
			return
		}
		requestErr <- nil
	}()

	<-requestStarted

	coordinator := NewCoordinator(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)), &stubScheduler{}, time.Second)
	signalCtx, cancelSignal := context.WithCancel(context.Background())
	coordinator.notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		return signalCtx, func() {}
	}

	result := make(chan int, 1)
	go func() {
		result <- coordinator.Handle(context.Background(), server)
	}()

	cancelSignal()
	close(allowFinish)

	if err := <-requestErr; err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if code := <-result; code != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", code, ExitCodeSuccess)
	}

	if err := <-serveDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("serve returned unexpected error: %v", err)
	}
}

func TestHandleTimeoutReturnsOne(t *testing.T) {
	t.Parallel()

	coordinator := NewCoordinator(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)), &stubScheduler{
		stop: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}, 50*time.Millisecond)

	signalCtx, cancelSignal := context.WithCancel(context.Background())
	coordinator.notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		return signalCtx, func() {}
	}

	result := make(chan int, 1)
	go func() {
		result <- coordinator.Handle(context.Background(), nil)
	}()

	cancelSignal()

	if code := <-result; code != ExitCodeFailure {
		t.Fatalf("unexpected exit code: got %d want %d", code, ExitCodeFailure)
	}
}

func TestHandleCallsHTTPShutdownBeforeScheduler(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		steps []string
	)
	httpShutdownCalled := make(chan struct{})

	scheduler := &stubScheduler{stop: func(ctx context.Context) error {
		select {
		case <-httpShutdownCalled:
		case <-ctx.Done():
			return ctx.Err()
		}

		mu.Lock()
		steps = append(steps, "scheduler")
		mu.Unlock()
		return nil
	}}

	server := &http.Server{}
	server.RegisterOnShutdown(func() {
		mu.Lock()
		steps = append(steps, "http")
		mu.Unlock()
		close(httpShutdownCalled)
	})

	coordinator := NewCoordinator(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)), scheduler, time.Second)
	signalCtx, cancelSignal := context.WithCancel(context.Background())
	coordinator.notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		return signalCtx, func() {}
	}

	cancelSignal()
	if code := coordinator.Handle(context.Background(), server); code != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", code, ExitCodeSuccess)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(steps) != 2 || steps[0] != "http" || steps[1] != "scheduler" {
		t.Fatalf("unexpected shutdown order: %v", steps)
	}
}
