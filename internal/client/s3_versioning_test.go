package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBucketVersioning_Enabled(t *testing.T) {
	xmlResp := `<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "versioning") {
			t.Errorf("expected query to contain versioning, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	status, err := client.GetBucketVersioning(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != "Enabled" {
		t.Errorf("expected Enabled, got %s", status)
	}
}

func TestGetBucketVersioning_Suspended(t *testing.T) {
	xmlResp := `<VersioningConfiguration><Status>Suspended</Status></VersioningConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	status, err := client.GetBucketVersioning(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != "Suspended" {
		t.Errorf("expected Suspended, got %s", status)
	}
}

func TestGetBucketVersioning_Unset(t *testing.T) {
	xmlResp := `<VersioningConfiguration></VersioningConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	status, err := client.GetBucketVersioning(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != "" {
		t.Errorf("expected empty string, got %s", status)
	}
}

func TestGetBucketVersioning_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>Internal server error</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	_, err := client.GetBucketVersioning(context.Background(), "ak", "sk", "test-bucket")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestPutBucketVersioning_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "versioning") {
			t.Errorf("expected query to contain versioning, got %s", r.URL.RawQuery)
		}
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketVersioning(context.Background(), "ak", "sk", "test-bucket", "Enabled")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	body := string(receivedBody)
	if !strings.Contains(body, "<Status>Enabled</Status>") {
		t.Errorf("expected request body to contain <Status>Enabled</Status>, got %s", body)
	}
}

func TestPutBucketVersioning_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>Internal server error</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketVersioning(context.Background(), "ak", "sk", "test-bucket", "Enabled")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}
