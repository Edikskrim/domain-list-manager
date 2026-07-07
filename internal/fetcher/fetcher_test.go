package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "test-etag-123")
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	f := New(
		WithTimeout(5*time.Second),
		WithMaxRetries(2),
		WithUserAgent("TestAgent/1.0"),
	)

	resp, err := f.Fetch(server.URL, "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.ETag != "test-etag-123" {
		t.Errorf("expected etag 'test-etag-123', got '%s'", resp.ETag)
	}

	if resp.LastModified != "Mon, 01 Jan 2024 00:00:00 GMT" {
		t.Errorf("unexpected last-modified: %s", resp.LastModified)
	}

	if string(resp.Body) != "Hello, World!" {
		t.Errorf("expected body 'Hello, World!', got '%s'", string(resp.Body))
	}

	if !resp.IsContentModified() {
		t.Error("expected content to be modified")
	}

	if resp.NeedsRetry() {
		t.Error("should not need retry on success")
	}

	t.Logf("Response: status=%d, etag=%s, body=%s, contentLength=%d",
		resp.StatusCode, resp.ETag, string(resp.Body), resp.ContentLength)
}

func TestFetcher_Fetch_WithEtag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "existing-etag" {
			w.Header().Set("ETag", "new-etag-456")
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", "new-etag-789")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Updated content"))
	}))
	defer server.Close()

	f := New(WithTimeout(5 * time.Second))

	// First fetch
	resp, err := f.Fetch(server.URL, "", "")
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Second fetch with ETag
	resp, err = f.Fetch(server.URL, "existing-etag", "")
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotModified {
		t.Errorf("expected 304, got %d", resp.StatusCode)
	}
	if resp.IsContentModified() {
		t.Error("304 should indicate content NOT modified")
	}

	t.Logf("Etag test: first=%d, second=%d", 200, 304)
}

func TestFetcher_Fetch_Retry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success after retries"))
	}))
	defer server.Close()

	f := New(
		WithTimeout(5*time.Second),
		WithMaxRetries(5),
	)

	resp, err := f.Fetch(server.URL, "", "")
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if string(resp.Body) != "Success after retries" {
		t.Errorf("unexpected body: %s", string(resp.Body))
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	t.Logf("Retry test: %d attempts made", attemptCount)
}

func TestFetcher_Fetch_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	f := New(
		WithTimeout(5*time.Second),
		WithMaxRetries(2),
	)

	resp, err := f.Fetch(server.URL, "", "")
	if err != nil {
		t.Logf("429 test: got error as expected: %v", err)
		// Error is expected for rate limiting
		return
	}

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}

	if resp.NeedsRetry() {
		t.Log("429 correctly indicates retry needed")
	} else {
		t.Error("429 should need retry")
	}
}

func TestFetcher_Fetch_CloneBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test body content"))
	}))
	defer server.Close()

	f := New(WithTimeout(5 * time.Second))
	resp, _ := f.Fetch(server.URL, "", "")

	clone, err := resp.CloneBody()
	if err != nil {
		t.Fatalf("CloneBody failed: %v", err)
	}

	if string(clone) != "test body content" {
		t.Errorf("clone mismatch: got '%s'", string(clone))
	}

	t.Logf("CloneBody test: original=%s, clone=%s", string(resp.Body), string(clone))
}
