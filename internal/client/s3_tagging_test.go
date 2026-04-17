package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBucketTagging_Success(t *testing.T) {
	taggingXML := `<Tagging><TagSet><Tag><Key>env</Key><Value>prod</Value></Tag><Tag><Key>team</Key><Value>platform</Value></Tag></TagSet></Tagging>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "tagging" {
			t.Errorf("expected query 'tagging', got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(taggingXML))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	tags, err := client.GetBucketTagging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags["env"] != "prod" {
		t.Errorf("expected tag 'env'='prod', got %q", tags["env"])
	}
	if tags["team"] != "platform" {
		t.Errorf("expected tag 'team'='platform', got %q", tags["team"])
	}
}

func TestGetBucketTagging_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`<Error><Code>NoSuchTagSet</Code><Message>The TagSet does not exist</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	tags, err := client.GetBucketTagging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tags != nil {
		t.Errorf("expected nil tags, got %v", tags)
	}
}

func TestGetBucketTagging_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`<Error><Code>InternalError</Code><Message>Internal server error</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	_, err := client.GetBucketTagging(context.Background(), "ak", "sk", "test-bucket")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestPutBucketTagging_Success(t *testing.T) {
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.RawQuery != "tagging" {
			t.Errorf("expected query 'tagging', got %s", r.URL.RawQuery)
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		gotBody = string(bodyBytes)
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	tags := map[string]string{"env": "prod", "team": "platform"}
	err := client.PutBucketTagging(context.Background(), "ak", "sk", "test-bucket", tags)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(gotBody, "<Tag>") {
		t.Errorf("expected request body to contain <Tag> elements, got %q", gotBody)
	}
	if !strings.Contains(gotBody, "<Key>") {
		t.Errorf("expected request body to contain <Key> elements, got %q", gotBody)
	}
}

func TestDeleteBucketTagging_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "tagging" {
			t.Errorf("expected query 'tagging', got %s", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketTagging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucketTagging_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketTagging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404 (idempotent delete), got %v", err)
	}
}
