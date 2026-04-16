package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateBucket_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.CreateBucket(context.Background(), "ak", "sk", "test-bucket", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreateBucket_ObjectLockEnabled(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Amz-Bucket-Object-Lock-Enabled")
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.CreateBucket(context.Background(), "ak", "sk", "lock-bucket", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotHeader != "true" {
		t.Errorf("expected x-amz-bucket-object-lock-enabled header to be \"true\", got %q", gotHeader)
	}
}

func TestCreateBucket_AlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(`<Error><Code>BucketAlreadyOwnedByYou</Code><Message>Your previous request to create the named bucket succeeded and you already own it.</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.CreateBucket(context.Background(), "ak", "sk", "existing-bucket", false)
	if err == nil {
		t.Fatal("expected error for 409 conflict")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error to contain \"already exists\", got %q", err.Error())
	}
}

func TestHeadBucket_Exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	exists, err := client.HeadBucket(context.Background(), "ak", "sk", "my-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected exists to be true")
	}
}

func TestHeadBucket_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	exists, err := client.HeadBucket(context.Background(), "ak", "sk", "missing-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected exists to be false")
	}
}

func TestHeadBucket_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	exists, err := client.HeadBucket(context.Background(), "ak", "sk", "forbidden-bucket")
	if exists {
		t.Error("expected exists to be false")
	}
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("expected error to contain \"access denied\", got %q", err.Error())
	}
}

func TestDeleteBucket_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/del-bucket" {
			t.Errorf("expected path /del-bucket, got %s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucket(context.Background(), "ak", "sk", "del-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucket_NotFound_Idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucket(context.Background(), "ak", "sk", "gone-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404 (idempotent delete), got %v", err)
	}
}

func TestDeleteBucket_OtherError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`<Error><Code>InternalError</Code><Message>Internal server error</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucket(context.Background(), "ak", "sk", "err-bucket")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}
