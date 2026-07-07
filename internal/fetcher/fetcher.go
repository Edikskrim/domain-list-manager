package fetcher

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Response represents the result of a fetch operation.
type Response struct {
	Body           []byte
	ETag           string
	LastModified   string
	StatusCode     int
	ContentLength  int64
}

// Fetcher handles HTTP requests with retry and caching support.
type Fetcher struct {
	client       *http.Client
	timeout      time.Duration
	maxRetries   int
	maxBodySize  int64
	maxRedirects int
	userAgent    string
}

// Option configures the Fetcher.
type Option func(*Fetcher)

// New creates a new Fetcher with the given options.
func New(opts ...Option) *Fetcher {
	f := &Fetcher{
		timeout:      30 * time.Second,
		maxRetries:   3,
		maxBodySize:  50 * 1024 * 1024,
		maxRedirects: 10,
		userAgent:    "DomainListManager/1.0",
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(t time.Duration) Option {
	return func(f *Fetcher) {
		f.timeout = t
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(f *Fetcher) {
		f.maxRetries = n
	}
}

// WithMaxBodySize sets the maximum allowed response body size.
func WithMaxBodySize(size int64) Option {
	return func(f *Fetcher) {
		f.maxBodySize = size
	}
}

// WithMaxRedirects sets the maximum number of redirects.
func WithMaxRedirects(n int) Option {
	return func(f *Fetcher) {
		f.maxRedirects = n
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

// Fetch retrieves the content from the given URL.
// It supports ETag and Last-Modified headers for caching.
func (f *Fetcher) Fetch(url string, etag, lastModified string) (*Response, error) {
	var resp *Response
	var lastErr error

	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		if attempt > 0 {
			retryDelay := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(retryDelay)
		}

		resp, lastErr = f.doFetch(url, etag, lastModified)
		if lastErr == nil {
			if resp.NeedsRetry() {
				continue
			}
			return resp, nil
		}

		// Don't retry on client errors (4xx) except 429 (Too Many Requests)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, lastErr
		}
	}

	if resp != nil {
		return resp, lastErr
	}
	return nil, fmt.Errorf("failed to fetch %q after %d retries: %w", url, f.maxRetries, lastErr)
}

func (f *Fetcher) doFetch(url, etag, lastModified string) (*Response, error) {
	client := &http.Client{
		Timeout: f.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= f.maxRedirects {
				return fmt.Errorf("too many redirects (%d)", f.maxRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	httpResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	// Handle 304 Not Modified
	if httpResp.StatusCode == http.StatusNotModified {
		return &Response{
			StatusCode: 304,
			ETag:       etag,
		}, nil
	}

	// Handle 429 Too Many Requests
	if httpResp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429) for %q", url)
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, f.maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Check if body exceeded limit
	if int64(len(body)) > f.maxBodySize {
		return nil, fmt.Errorf("body size %d exceeds limit %d", len(body), f.maxBodySize)
	}

	return &Response{
		Body:          body,
		ETag:          httpResp.Header.Get("ETag"),
		LastModified:  httpResp.Header.Get("Last-Modified"),
		StatusCode:    httpResp.StatusCode,
		ContentLength: int64(len(body)),
	}, nil
}

// IsContentModified checks if the response indicates content has been modified.
func (r *Response) IsContentModified() bool {
	if r == nil {
		return false
	}
	// 304 means content has NOT been modified
	return r.StatusCode != 304
}

// NeedsRetry checks if the response status code indicates a retryable error.
func (r *Response) NeedsRetry() bool {
	if r == nil {
		return false
	}
	return r.StatusCode >= 500 || r.StatusCode == 429
}

// CloneBody creates a copy of the response body.
func (r *Response) CloneBody() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return bytes.Clone(r.Body), nil
}

// Close releases any resources used by the response.
func (r *Response) Close() {
	// For []byte body, no resources to release
}
