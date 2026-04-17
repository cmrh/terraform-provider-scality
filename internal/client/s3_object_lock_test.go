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

func TestPutObjectLockConfiguration_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/lock-bucket" {
			t.Errorf("expected path /lock-bucket, got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "object-lock") {
			t.Errorf("expected query to contain object-lock, got %s", r.URL.RawQuery)
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
	cfg := ObjectLockConfig{
		Enabled:       true,
		RetentionMode: "COMPLIANCE",
		RetentionDays: 365,
	}
	err := client.PutObjectLockConfiguration(context.Background(), "ak", "sk", "lock-bucket", cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the XML body sent to the server.
	var config objectLockConfiguration
	if xmlErr := xml.Unmarshal(receivedBody, &config); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if config.ObjectLockEnabled != "Enabled" {
		t.Errorf("expected ObjectLockEnabled to be Enabled, got %s", config.ObjectLockEnabled)
	}
	if config.Rule == nil {
		t.Fatal("expected Rule to be present")
	}
	if config.Rule.DefaultRetention.Mode != "COMPLIANCE" {
		t.Errorf("expected retention mode COMPLIANCE, got %s", config.Rule.DefaultRetention.Mode)
	}
	if config.Rule.DefaultRetention.Days != 365 {
		t.Errorf("expected retention days 365, got %d", config.Rule.DefaultRetention.Days)
	}
}

func TestGetObjectLockConfiguration_Success(t *testing.T) {
	xmlResp := `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Days>365</Days></DefaultRetention></Rule></ObjectLockConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "object-lock") {
			t.Errorf("expected query to contain object-lock, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetObjectLockConfiguration(context.Background(), "ak", "sk", "lock-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.RetentionMode != "COMPLIANCE" {
		t.Errorf("expected RetentionMode COMPLIANCE, got %s", cfg.RetentionMode)
	}
	if cfg.RetentionDays != 365 {
		t.Errorf("expected RetentionDays 365, got %d", cfg.RetentionDays)
	}
	if cfg.RetentionYears != 0 {
		t.Errorf("expected RetentionYears 0, got %d", cfg.RetentionYears)
	}
}

func TestGetObjectLockConfiguration_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<Error><Code>ObjectLockConfigurationNotFoundError</Code><Message>Object Lock configuration does not exist for this bucket</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetObjectLockConfiguration(context.Background(), "ak", "sk", "no-lock-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for 404, got %+v", cfg)
	}
}

func TestGetObjectLockConfiguration_WithoutRule(t *testing.T) {
	xmlResp := `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetObjectLockConfiguration(context.Background(), "ak", "sk", "enabled-only-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.RetentionMode != "" {
		t.Errorf("expected empty RetentionMode, got %s", cfg.RetentionMode)
	}
	if cfg.RetentionDays != 0 {
		t.Errorf("expected RetentionDays 0, got %d", cfg.RetentionDays)
	}
	if cfg.RetentionYears != 0 {
		t.Errorf("expected RetentionYears 0, got %d", cfg.RetentionYears)
	}
}

func TestGetObjectLockConfiguration_WithYears(t *testing.T) {
	xmlResp := `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>GOVERNANCE</Mode><Years>3</Years></DefaultRetention></Rule></ObjectLockConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	cfg, err := client.GetObjectLockConfiguration(context.Background(), "ak", "sk", "years-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.RetentionMode != "GOVERNANCE" {
		t.Errorf("expected RetentionMode GOVERNANCE, got %s", cfg.RetentionMode)
	}
	if cfg.RetentionDays != 0 {
		t.Errorf("expected RetentionDays 0, got %d", cfg.RetentionDays)
	}
	if cfg.RetentionYears != 3 {
		t.Errorf("expected RetentionYears 3, got %d", cfg.RetentionYears)
	}
}
