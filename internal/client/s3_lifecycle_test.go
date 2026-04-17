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

func TestGetBucketLifecycle_Success(t *testing.T) {
	xmlResp := `<LifecycleConfiguration><Rule><ID>expire-logs</ID><Status>Enabled</Status><Filter><Prefix>logs/</Prefix></Filter><Expiration><Days>30</Days></Expiration></Rule></LifecycleConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "lifecycle") {
			t.Errorf("expected query to contain lifecycle, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	rules, err := client.GetBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "expire-logs" {
		t.Errorf("expected ID expire-logs, got %s", rules[0].ID)
	}
	if rules[0].Status != "Enabled" {
		t.Errorf("expected Status Enabled, got %s", rules[0].Status)
	}
	if rules[0].Prefix != "logs/" {
		t.Errorf("expected Prefix logs/, got %s", rules[0].Prefix)
	}
	if rules[0].ExpirationDays != 30 {
		t.Errorf("expected ExpirationDays 30, got %d", rules[0].ExpirationDays)
	}
}

func TestGetBucketLifecycle_MultipleRules(t *testing.T) {
	xmlResp := `<LifecycleConfiguration><Rule><ID>cleanup</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><NoncurrentVersionExpiration><NoncurrentDays>90</NoncurrentDays></NoncurrentVersionExpiration></Rule><Rule><ID>abort-uploads</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><AbortIncompleteMultipartUpload><DaysAfterInitiation>7</DaysAfterInitiation></AbortIncompleteMultipartUpload></Rule></LifecycleConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	rules, err := client.GetBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].ID != "cleanup" {
		t.Errorf("expected first rule ID cleanup, got %s", rules[0].ID)
	}
	if rules[0].NoncurrentVersionExpirationDays != 90 {
		t.Errorf("expected NoncurrentVersionExpirationDays 90, got %d", rules[0].NoncurrentVersionExpirationDays)
	}
	if rules[1].ID != "abort-uploads" {
		t.Errorf("expected second rule ID abort-uploads, got %s", rules[1].ID)
	}
	if rules[1].AbortIncompleteMultipartUploadDays != 7 {
		t.Errorf("expected AbortIncompleteMultipartUploadDays 7, got %d", rules[1].AbortIncompleteMultipartUploadDays)
	}
}

func TestGetBucketLifecycle_ExpirationDate(t *testing.T) {
	xmlResp := `<LifecycleConfiguration><Rule><ID>date-expire</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Expiration><Date>2025-12-31T00:00:00Z</Date></Expiration></Rule></LifecycleConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	rules, err := client.GetBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ExpirationDate != "2025-12-31T00:00:00Z" {
		t.Errorf("expected ExpirationDate 2025-12-31T00:00:00Z, got %s", rules[0].ExpirationDate)
	}
}

func TestGetBucketLifecycle_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`<Error><Code>NoSuchLifecycleConfiguration</Code><Message>The lifecycle configuration does not exist</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	rules, err := client.GetBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
	if rules != nil {
		t.Errorf("expected nil rules for 404, got %+v", rules)
	}
}

func TestPutBucketLifecycle_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "lifecycle") {
			t.Errorf("expected query to contain lifecycle, got %s", r.URL.RawQuery)
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
	err := client.PutBucketLifecycle(context.Background(), "ak", "sk", "test-bucket", []LifecycleRule{
		{
			ID:             "expire-logs",
			Status:         "Enabled",
			Prefix:         "logs/",
			ExpirationDays: 30,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var config lifecycleConfiguration
	if xmlErr := xml.Unmarshal(receivedBody, &config); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if len(config.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(config.Rules))
	}
	if config.Rules[0].ID != "expire-logs" {
		t.Errorf("expected rule ID expire-logs, got %s", config.Rules[0].ID)
	}
	if config.Rules[0].Expiration == nil {
		t.Fatal("expected Expiration to be present")
	}
	if config.Rules[0].Expiration.Days != 30 {
		t.Errorf("expected Expiration Days 30, got %d", config.Rules[0].Expiration.Days)
	}
}

func TestDeleteBucketLifecycle_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "lifecycle") {
			t.Errorf("expected query to contain lifecycle, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucketLifecycle_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketLifecycle(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
}
