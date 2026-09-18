package client

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const notImplementedBody = `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NotImplemented</Code><Message>A header you provided implies functionality that is not implemented.</Message></Error>`

func TestGetBucketLogging_Configured(t *testing.T) {
	xmlResp := `<BucketLoggingStatus xmlns="http://doc.s3.amazonaws.com/2006-03-01"><LoggingEnabled><TargetBucket>my-logs</TargetBucket><TargetPrefix>src/</TargetPrefix></LoggingEnabled></BucketLoggingStatus>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "logging") {
			t.Errorf("expected query to contain logging, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	cfg, err := c.GetBucketLogging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.TargetBucket != "my-logs" || cfg.TargetPrefix != "src/" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestGetBucketLogging_NotConfigured(t *testing.T) {
	xmlResp := `<BucketLoggingStatus xmlns="http://doc.s3.amazonaws.com/2006-03-01" />`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	cfg, err := c.GetBucketLogging(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config when not configured, got %+v", cfg)
	}
}

func TestGetBucketLogging_FeatureDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(501)
		_, _ = w.Write([]byte(notImplementedBody))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	_, err := c.GetBucketLogging(context.Background(), "ak", "sk", "test-bucket")
	if !errors.Is(err, ErrServerAccessLoggingDisabled) {
		t.Fatalf("expected ErrServerAccessLoggingDisabled, got %v", err)
	}
}

func TestPutBucketLogging_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "logging") {
			t.Errorf("expected query to contain logging, got %s", r.URL.RawQuery)
		}
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	err := c.PutBucketLogging(context.Background(), "ak", "sk", "test-bucket", LoggingConfig{
		TargetBucket: "my-logs",
		TargetPrefix: "src/",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var status bucketLoggingStatus
	if xmlErr := xml.Unmarshal(receivedBody, &status); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if status.Enabled == nil {
		t.Fatal("expected LoggingEnabled in request body")
	}
	if status.Enabled.TargetBucket != "my-logs" || status.Enabled.TargetPrefix != "src/" {
		t.Errorf("unexpected request body: %+v", status.Enabled)
	}
}

func TestPutBucketLogging_FeatureDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(501)
		_, _ = w.Write([]byte(notImplementedBody))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	err := c.PutBucketLogging(context.Background(), "ak", "sk", "test-bucket", LoggingConfig{TargetBucket: "my-logs", TargetPrefix: ""})
	if !errors.Is(err, ErrServerAccessLoggingDisabled) {
		t.Fatalf("expected ErrServerAccessLoggingDisabled, got %v", err)
	}
}

func TestDeleteBucketLogging_SendsEmptyStatus(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT (disable is a PUT of an empty status), got %s", r.Method)
		}
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	if err := c.DeleteBucketLogging(context.Background(), "ak", "sk", "test-bucket"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var status bucketLoggingStatus
	if xmlErr := xml.Unmarshal(receivedBody, &status); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if status.Enabled != nil {
		t.Errorf("expected empty BucketLoggingStatus (no LoggingEnabled), got %+v", status.Enabled)
	}
}

func TestDeleteBucketLogging_FeatureDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(501)
		_, _ = w.Write([]byte(notImplementedBody))
	}))
	defer server.Close()

	c := NewS3Client(server.URL, false)
	err := c.DeleteBucketLogging(context.Background(), "ak", "sk", "test-bucket")
	if !errors.Is(err, ErrServerAccessLoggingDisabled) {
		t.Fatalf("expected ErrServerAccessLoggingDisabled, got %v", err)
	}
}
