package maxigobot

import (
	gocontext "context"
	"errors"
	"net/http"
	"testing"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func apiErr(status int, msg string) error {
	return &maxigo.Error{
		Kind:       maxigo.ErrAPI,
		StatusCode: status,
		Message:    msg,
	}
}

// helper configs for tests
var (
	testRateLimitCfg = retryConfig{
		rateLimitIntervals: []time.Duration{time.Millisecond, time.Millisecond},
	}
	testBothCfg = retryConfig{
		rateLimitIntervals:   []time.Duration{time.Millisecond},
		uploadRetryIntervals: []time.Duration{time.Millisecond},
	}
)

func TestIntervalsFor(t *testing.T) {
	cfg := retryConfig{
		rateLimitIntervals:   DefaultRateLimitIntervals,
		uploadRetryIntervals: DefaultUploadRetryIntervals,
	}

	tests := []struct {
		name string
		err  error
		want []time.Duration
	}{
		{"nil", nil, nil},
		{"non-api error", errors.New("boom"), nil},
		{"network error", &maxigo.Error{Kind: maxigo.ErrNetwork}, nil},
		{"500", apiErr(500, "internal"), nil},
		{"400 other", apiErr(400, "bad request"), nil},
		{"429", apiErr(429, "too many requests"), DefaultRateLimitIntervals},
		{"400 not.processed", apiErr(400, "errors.process.attachment.file.not.processed"), DefaultUploadRetryIntervals},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intervalsFor(tt.err, cfg)
			if len(got) != len(tt.want) {
				t.Errorf("intervalsFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithRetry_RateLimitSuccess(t *testing.T) {
	calls := 0
	cfg := retryConfig{rateLimitIntervals: []time.Duration{time.Millisecond}}
	err := withRetry(gocontext.Background(), cfg, func() error {
		calls++
		if calls == 1 {
			return apiErr(http.StatusTooManyRequests, "rate limited")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_UploadSuccess(t *testing.T) {
	calls := 0
	cfg := retryConfig{uploadRetryIntervals: []time.Duration{time.Millisecond}}
	err := withRetry(gocontext.Background(), cfg, func() error {
		calls++
		if calls == 1 {
			return apiErr(http.StatusBadRequest, "errors.process.attachment.file.not.processed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_RateLimitExhausted(t *testing.T) {
	calls := 0
	err := withRetry(gocontext.Background(), testRateLimitCfg, func() error {
		calls++
		return apiErr(http.StatusTooManyRequests, "rate limited")
	})
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	// 1 initial + 2 retries = 3 calls
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_UploadNotRetried_WhenDisabled(t *testing.T) {
	calls := 0
	cfg := retryConfig{rateLimitIntervals: []time.Duration{time.Millisecond}} // no upload intervals
	err := withRetry(gocontext.Background(), cfg, func() error {
		calls++
		return apiErr(http.StatusBadRequest, "errors.process.attachment.file.not.processed")
	})
	if err == nil {
		t.Fatal("expected error with upload retries disabled")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_EmptyConfig(t *testing.T) {
	calls := 0
	err := withRetry(gocontext.Background(), retryConfig{}, func() error {
		calls++
		return apiErr(http.StatusTooManyRequests, "rate limited")
	})
	if err == nil {
		t.Fatal("expected error with no retries configured")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	calls := 0
	err := withRetry(gocontext.Background(), testBothCfg, func() error {
		calls++
		return apiErr(http.StatusBadRequest, "invalid body")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (non-retryable), got %d", calls)
	}
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	calls := 0
	cfg := retryConfig{rateLimitIntervals: []time.Duration{time.Second}}
	err := withRetry(ctx, cfg, func() error {
		calls++
		cancel() // cancel before the retry sleep
		return apiErr(http.StatusTooManyRequests, "rate limited")
	})
	if !errors.Is(err, gocontext.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_ImmediateSuccess(t *testing.T) {
	calls := 0
	cfg := retryConfig{
		rateLimitIntervals:   DefaultRateLimitIntervals,
		uploadRetryIntervals: DefaultUploadRetryIntervals,
	}
	err := withRetry(gocontext.Background(), cfg, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}
