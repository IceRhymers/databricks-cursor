package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// fakeTokenFetcher implements tokencache.TokenFetcher for testing.
type fakeTokenFetcher struct{}

func (f *fakeTokenFetcher) FetchToken(_ context.Context) (string, time.Time, error) {
	return "fake-token", time.Now().Add(1 * time.Hour), nil
}

func TestNewProxy_ReturnsHandler(t *testing.T) {
	// Create a fake upstream server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify token was injected.
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("expected Authorization header, got empty")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	fetcher := &staticFetcher{token: "test-proxy-token"}
	tp := tokencache.NewTokenProvider(fetcher)

	handler := newProxy(upstream.URL, upstream.URL, tp, false)
	if handler == nil {
		t.Fatal("newProxy returned nil handler")
	}

	// Make a request through the proxy.
	req := httptest.NewRequest("GET", "/v1/models", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// staticFetcher is a simple tokencache.TokenFetcher for tests.
type staticFetcher struct {
	token string
}

func (f *staticFetcher) FetchToken(_ context.Context) (string, time.Time, error) {
	return f.token, time.Now().Add(1 * time.Hour), nil
}
