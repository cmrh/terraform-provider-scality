package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestS3BucketLock_SerializesSameBucket verifies doRequest serializes mutating
// requests per bucket (at most one in flight for a given bucket) while allowing
// different buckets to proceed in parallel.
func TestS3BucketLock_SerializesSameBucket(t *testing.T) {
	var mu sync.Mutex
	inflight := map[string]int{}
	maxPerBucket := map[string]int{}
	globalMax, global := 0, 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := strings.TrimPrefix(r.URL.Path, "/")
		mu.Lock()
		inflight[bucket]++
		global++
		if inflight[bucket] > maxPerBucket[bucket] {
			maxPerBucket[bucket] = inflight[bucket]
		}
		if global > globalMax {
			globalMax = global
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // widen the overlap window

		mu.Lock()
		inflight[bucket]--
		global--
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)

	var wg sync.WaitGroup
	for _, bucket := range []string{"b1", "b2"} {
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func(b string) {
				defer wg.Done()
				_ = c.PutBucketPolicy(context.Background(), "ak", "sk", b, `{"Version":"2012-10-17","Statement":[]}`)
			}(bucket)
		}
	}
	wg.Wait()

	for _, b := range []string{"b1", "b2"} {
		if maxPerBucket[b] != 1 {
			t.Errorf("bucket %s: expected max 1 concurrent request, got %d", b, maxPerBucket[b])
		}
	}
	if globalMax < 2 {
		t.Errorf("expected different buckets to run in parallel (global max >= 2), got %d", globalMax)
	}
}

// TestS3BucketLock_ReadsNotSerialized verifies GET requests are not locked, so
// reads of the same bucket proceed in parallel.
func TestS3BucketLock_ReadsNotSerialized(t *testing.T) {
	var mu sync.Mutex
	inflight, maxInflight := 0, 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inflight++
		if inflight > maxInflight {
			maxInflight = inflight
		}
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		inflight--
		mu.Unlock()
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"Version":"2012-10-17","Statement":[]}`))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.GetBucketPolicy(context.Background(), "ak", "sk", "b1")
		}()
	}
	wg.Wait()

	if maxInflight < 2 {
		t.Errorf("expected concurrent reads of the same bucket (max >= 2), got %d", maxInflight)
	}
}
