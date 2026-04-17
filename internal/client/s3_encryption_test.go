package client

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBucketEncryption_Success(t *testing.T) {
	xmlResp := `<ServerSideEncryptionConfiguration><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>aws:kms</SSEAlgorithm><KMSMasterKeyID>arn:aws:kms:us-east-1:123:key/abc</KMSMasterKeyID></ApplyServerSideEncryptionByDefault></Rule></ServerSideEncryptionConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("expected query to contain encryption, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.SSEAlgorithm != "aws:kms" {
		t.Errorf("expected SSEAlgorithm aws:kms, got %s", cfg.SSEAlgorithm)
	}
	if cfg.KMSMasterKeyID != "arn:aws:kms:us-east-1:123:key/abc" {
		t.Errorf("expected KMSMasterKeyID arn:aws:kms:us-east-1:123:key/abc, got %s", cfg.KMSMasterKeyID)
	}
}

func TestGetBucketEncryption_SSE_S3(t *testing.T) {
	xmlResp := `<ServerSideEncryptionConfiguration><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>AES256</SSEAlgorithm></ApplyServerSideEncryptionByDefault></Rule></ServerSideEncryptionConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.SSEAlgorithm != "AES256" {
		t.Errorf("expected SSEAlgorithm AES256, got %s", cfg.SSEAlgorithm)
	}
	if cfg.KMSMasterKeyID != "" {
		t.Errorf("expected empty KMSMasterKeyID, got %s", cfg.KMSMasterKeyID)
	}
}

func TestGetBucketEncryption_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`<Error><Code>ServerSideEncryptionConfigurationNotFoundError</Code><Message>The server side encryption configuration was not found</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for 404, got %+v", cfg)
	}
}

func TestGetBucketEncryption_NoRules(t *testing.T) {
	xmlResp := `<ServerSideEncryptionConfiguration></ServerSideEncryptionConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for empty rules, got %+v", cfg)
	}
}

func TestPutBucketEncryption_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("expected query to contain encryption, got %s", r.URL.RawQuery)
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
	err := client.PutBucketEncryption(context.Background(), "ak", "sk", "test-bucket", EncryptionConfig{
		SSEAlgorithm:   "aws:kms",
		KMSMasterKeyID: "arn:aws:kms:us-east-1:123:key/abc",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var config serverSideEncryptionConfiguration
	if xmlErr := xml.Unmarshal(receivedBody, &config); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if len(config.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(config.Rules))
	}
	if config.Rules[0].Apply.SSEAlgorithm != "aws:kms" {
		t.Errorf("expected SSEAlgorithm aws:kms, got %s", config.Rules[0].Apply.SSEAlgorithm)
	}
	if config.Rules[0].Apply.KMSMasterKeyID != "arn:aws:kms:us-east-1:123:key/abc" {
		t.Errorf("expected KMSMasterKeyID arn:aws:kms:us-east-1:123:key/abc, got %s", config.Rules[0].Apply.KMSMasterKeyID)
	}
}

func TestDeleteBucketEncryption_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("expected query to contain encryption, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucketEncryption_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketEncryption(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
}
