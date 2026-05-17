package honeycomb

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError represents a non-2xx response from the Honeycomb API.
type APIError struct {
	StatusCode int
	Body       string
	RetryAfter *time.Time // non-nil if Retry-After header was present
}

func (e *APIError) Error() string {
	if e.RetryAfter != nil {
		return fmt.Sprintf("honeycomb API error %d: %s (retry after %s)", e.StatusCode, e.Body, e.RetryAfter.Format(time.RFC1123))
	}
	return fmt.Sprintf("honeycomb API error %d: %s", e.StatusCode, e.Body)
}

// IsRateLimit returns true if the error is a 429 Too Many Requests.
func IsRateLimit(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusTooManyRequests
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}

// IsServerError returns true for 5xx errors.
func IsServerError(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode >= 500
}

// parseRetryAfter parses the Retry-After header value into a time.Time.
// Per Honeycomb docs, the header (when present) uses a GMT timestamp.
// The Query Data API does NOT include Retry-After on 429; callers must
// handle the nil case.
func parseRetryAfter(header string) *time.Time {
	if header == "" {
		return nil
	}
	// Try HTTP-date format first (RFC1123)
	if t, err := http.ParseTime(header); err == nil {
		return &t
	}
	// Try integer seconds
	if secs, err := strconv.ParseInt(strings.TrimSpace(header), 10, 64); err == nil {
		t := time.Now().Add(time.Duration(secs) * time.Second)
		return &t
	}
	return nil
}

// ParseRateLimitHeader parses the standard RateLimit response header:
// "limit=N, remaining=M, reset=S"
func ParseRateLimitHeader(header string) RateLimitInfo {
	info := RateLimitInfo{}
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			continue
		}
		switch key {
		case "limit":
			info.Limit = val
		case "remaining":
			info.Remaining = val
		case "reset":
			info.Reset = time.Duration(val) * time.Second
		}
	}
	return info
}
