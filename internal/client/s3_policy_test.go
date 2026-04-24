package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBucketPolicy_Success(t *testing.T) {
	policyJSON := `{"Version":"2012-10-17","Statement":[]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "policy" {
			t.Errorf("expected query 'policy', got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(policyJSON))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	result, err := client.GetBucketPolicy(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != policyJSON {
		t.Errorf("expected policy %q, got %q", policyJSON, result)
	}
}

func TestGetBucketPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<Error><Code>NoSuchBucketPolicy</Code><Message>The bucket policy does not exist</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	result, err := client.GetBucketPolicy(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestGetBucketPolicy_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>Internal server error</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	_, err := client.GetBucketPolicy(context.Background(), "ak", "sk", "test-bucket")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestPutBucketPolicy_Success(t *testing.T) {
	policyJSON := `{"Version":"2012-10-17","Statement":[]}`
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.RawQuery != "policy" {
			t.Errorf("expected query 'policy', got %s", r.URL.RawQuery)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketPolicy(context.Background(), "ak", "sk", "test-bucket", policyJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(gotBody, policyJSON) {
		t.Errorf("expected request body to contain %q, got %q", policyJSON, gotBody)
	}
}

func TestPutBucketPolicy_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketPolicy(context.Background(), "ak", "sk", "test-bucket", `{"Version":"2012-10-17"}`)
	if err != nil {
		t.Fatalf("expected no error for 204, got %v", err)
	}
}

func TestDeleteBucketPolicy_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "policy" {
			t.Errorf("expected query 'policy', got %s", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketPolicy(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucketPolicy_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketPolicy(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404 (idempotent delete), got %v", err)
	}
}
