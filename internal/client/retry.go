package client

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// Conservative retry policy: retry only requests the server provably did not
// process — transport failures (no response), throttling (429, 503), and the
// retryable 409 codes. Never retry a response that may have been processed
// (e.g. 500), so a non-idempotent create is never duplicated.
const (
	maxRetries    = 3
	retryBaseWait = 200 * time.Millisecond
	retryMaxWait  = 5 * time.Second
)

// isRetryableErr reports whether a transport-level error (no HTTP response was
// received) is safe to retry. A genuine network failure means the request did
// not complete; context cancellation is never retried.
func isRetryableErr(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// isRetryableStatus retries only responses the server did not process:
// throttling (429, 503) and the two retryable 409 conflict codes.
func isRetryableStatus(status int, body []byte) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	case http.StatusConflict:
		code := errorCode(body)
		return code == "OperationAborted" || code == "ConcurrentModification"
	}
	return false
}

// errorCode pulls <Code> out of an S3/IAM XML error envelope.
func errorCode(body []byte) string {
	var e struct {
		XMLName xml.Name
		Code    string `xml:"Code"`
	}
	if xml.Unmarshal(body, &e) == nil {
		return e.Code
	}
	return ""
}

// retrySleep waits with exponential backoff + jitter, or returns false if the
// context is done.
func retrySleep(ctx context.Context, attempt int) bool {
	wait := retryBaseWait << attempt
	if wait > retryMaxWait {
		wait = retryMaxWait
	}
	wait += time.Duration(rand.Int63n(int64(wait)/2 + 1)) // #nosec G404 -- jitter, not security-sensitive
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// doWithRetry calls send (which must build a FRESH *http.Request each call, so
// the body can be re-read) and retries transient failures per the policy above.
// It reads and closes the response body and returns (body, statusCode, err).
func doWithRetry(ctx context.Context, send func() (*http.Response, error)) ([]byte, int, error) {
	for attempt := 0; ; attempt++ {
		resp, err := send()
		if err != nil {
			if attempt < maxRetries && isRetryableErr(err) && retrySleep(ctx, attempt) {
				continue
			}
			return nil, 0, err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, 0, readErr
		}
		if attempt < maxRetries && isRetryableStatus(resp.StatusCode, body) && retrySleep(ctx, attempt) {
			continue
		}
		return body, resp.StatusCode, nil
	}
}
