package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestRetry_IAMAdminPathRetries(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) <= 2 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewIAMClient(srv.URL, "ak", "sk", false)
	_, status, err := c.DoSignedRequest(context.Background(), "GetAccount", url.Values{})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if status != 200 {
		t.Fatalf("expected status 200 after retries, got %d", status)
	}
	if got := atomic.LoadInt32(&n); got != 3 {
		t.Fatalf("expected 3 attempts on the IAM admin path, got %d", got)
	}
}

func TestRetry_ThrottleThenSucceeds(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) <= 2 {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`<Error><Code>SlowDown</Code></Error>`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewS3Client(srv.URL, false)
	out, err := c.GetBucketPolicy(context.Background(), "ak", "sk", "b")
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if out == "" {
		t.Fatal("expected a policy body after retry")
	}
	if got := atomic.LoadInt32(&n); got != 3 {
		t.Fatalf("expected 3 attempts (2 retries), got %d", got)
	}
}

func TestRetry_DoesNotRetry500(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code></Error>`))
	}))
	defer srv.Close()

	c := NewS3Client(srv.URL, false)
	if _, err := c.GetBucketPolicy(context.Background(), "ak", "sk", "b"); err == nil {
		t.Fatal("expected an error on 500")
	}
	if got := atomic.LoadInt32(&n); got != 1 {
		t.Fatalf("500 must not be retried; got %d attempts", got)
	}
}

func TestRetry_409OnlyRetryableCodes(t *testing.T) {
	var n1 int32
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n1, 1) <= 1 {
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`<Error><Code>OperationAborted</Code></Error>`))
			return
		}
		w.WriteHeader(200)
	}))
	defer s1.Close()
	_ = NewS3Client(s1.URL, false).PutBucketPolicy(context.Background(), "ak", "sk", "b", "{}")
	if got := atomic.LoadInt32(&n1); got != 2 {
		t.Fatalf("OperationAborted (409) should retry once; got %d attempts", got)
	}

	var n2 int32
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n2, 1)
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`<Error><Code>BucketAlreadyOwnedByYou</Code></Error>`))
	}))
	defer s2.Close()
	_ = NewS3Client(s2.URL, false).PutBucketPolicy(context.Background(), "ak", "sk", "b", "{}")
	if got := atomic.LoadInt32(&n2); got != 1 {
		t.Fatalf("non-retryable 409 must not retry; got %d attempts", got)
	}
}

func TestRetry_Classifier(t *testing.T) {
	if isRetryableStatus(500, nil) {
		t.Error("500 must not retry")
	}
	if !isRetryableStatus(429, nil) || !isRetryableStatus(503, nil) {
		t.Error("429/503 must retry")
	}
	if !isRetryableStatus(409, []byte(`<Error><Code>ConcurrentModification</Code></Error>`)) {
		t.Error("409 ConcurrentModification must retry")
	}
	if isRetryableStatus(409, []byte(`<Error><Code>EntityAlreadyExists</Code></Error>`)) {
		t.Error("409 EntityAlreadyExists must not retry")
	}
	if isRetryableErr(context.Canceled) || isRetryableErr(context.DeadlineExceeded) {
		t.Error("context cancellation must not retry")
	}
}
