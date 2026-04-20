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

func TestPutBucketReplication_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/src-bucket" {
			t.Errorf("expected path /src-bucket, got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "replication") {
			t.Errorf("expected query to contain replication, got %s", r.URL.RawQuery)
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
	rules := []ReplicationRule{
		{
			ID:                "rule1",
			Status:            "Enabled",
			Prefix:            "",
			DestinationBucket: "arn:aws:s3:::dest-bucket",
		},
	}
	err := client.PutBucketReplication(context.Background(), "ak", "sk", "src-bucket", "arn:aws:iam::123:role/repl-role", rules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the XML body sent to the server.
	var config replicationConfiguration
	if xmlErr := xml.Unmarshal(receivedBody, &config); xmlErr != nil {
		t.Fatalf("failed to unmarshal request body: %v", xmlErr)
	}
	if config.Role != "arn:aws:iam::123:role/repl-role" {
		t.Errorf("expected role arn:aws:iam::123:role/repl-role, got %s", config.Role)
	}
	if len(config.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(config.Rules))
	}
	if config.Rules[0].ID != "rule1" {
		t.Errorf("expected rule ID rule1, got %s", config.Rules[0].ID)
	}
	if config.Rules[0].Status != "Enabled" {
		t.Errorf("expected rule Status Enabled, got %s", config.Rules[0].Status)
	}
	if config.Rules[0].Destination.Bucket != "arn:aws:s3:::dest-bucket" {
		t.Errorf("expected destination bucket arn:aws:s3:::dest-bucket, got %s", config.Rules[0].Destination.Bucket)
	}
}

func TestGetBucketReplication_Success(t *testing.T) {
	xmlResp := `<ReplicationConfiguration><Role>arn:aws:iam::123:role/repl-role</Role><Rule><ID>rule1</ID><Status>Enabled</Status><Prefix></Prefix><Destination><Bucket>arn:aws:s3:::dest-bucket</Bucket></Destination></Rule></ReplicationConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "replication") {
			t.Errorf("expected query to contain replication, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	role, rules, err := client.GetBucketReplication(context.Background(), "ak", "sk", "src-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if role != "arn:aws:iam::123:role/repl-role" {
		t.Errorf("expected role arn:aws:iam::123:role/repl-role, got %s", role)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "rule1" {
		t.Errorf("expected rule ID rule1, got %s", rules[0].ID)
	}
	if rules[0].Status != "Enabled" {
		t.Errorf("expected rule Status Enabled, got %s", rules[0].Status)
	}
	if rules[0].Prefix != "" {
		t.Errorf("expected empty prefix, got %s", rules[0].Prefix)
	}
	if rules[0].DestinationBucket != "arn:aws:s3:::dest-bucket" {
		t.Errorf("expected destination bucket arn:aws:s3:::dest-bucket, got %s", rules[0].DestinationBucket)
	}
}

func TestGetBucketReplication_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<Error><Code>ReplicationConfigurationNotFoundError</Code><Message>The replication configuration was not found</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	role, rules, err := client.GetBucketReplication(context.Background(), "ak", "sk", "no-repl-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
	if role != "" {
		t.Errorf("expected empty role, got %s", role)
	}
	if rules != nil {
		t.Errorf("expected nil rules, got %v", rules)
	}
}

func TestDeleteBucketReplication_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "replication") {
			t.Errorf("expected query to contain replication, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketReplication(context.Background(), "ak", "sk", "src-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteBucketReplication_NotFound_Idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketReplication(context.Background(), "ak", "sk", "gone-bucket")
	if err != nil {
		t.Fatalf("expected no error for 404, got %v", err)
	}
}

func TestDeleteBucketReplication_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>Something went wrong</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.DeleteBucketReplication(context.Background(), "ak", "sk", "err-bucket")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestGetBucketReplication_MultipleRules(t *testing.T) {
	xmlResp := `<ReplicationConfiguration>
		<Role>arn:aws:iam::456:role/multi-role</Role>
		<Rule>
			<ID>rule-a</ID>
			<Status>Enabled</Status>
			<Prefix>logs/</Prefix>
			<Destination><Bucket>arn:aws:s3:::dest-a</Bucket></Destination>
		</Rule>
		<Rule>
			<ID>rule-b</ID>
			<Status>Disabled</Status>
			<Prefix>data/</Prefix>
			<Destination><Bucket>arn:aws:s3:::dest-b</Bucket></Destination>
		</Rule>
		<Rule>
			<ID>rule-c</ID>
			<Status>Enabled</Status>
			<Prefix></Prefix>
			<Destination><Bucket>arn:aws:s3:::dest-c</Bucket></Destination>
		</Rule>
	</ReplicationConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	role, rules, err := client.GetBucketReplication(context.Background(), "ak", "sk", "multi-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if role != "arn:aws:iam::456:role/multi-role" {
		t.Errorf("expected role arn:aws:iam::456:role/multi-role, got %s", role)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	expected := []struct {
		id     string
		status string
		prefix string
		dest   string
	}{
		{"rule-a", "Enabled", "logs/", "arn:aws:s3:::dest-a"},
		{"rule-b", "Disabled", "data/", "arn:aws:s3:::dest-b"},
		{"rule-c", "Enabled", "", "arn:aws:s3:::dest-c"},
	}

	for i, exp := range expected {
		if rules[i].ID != exp.id {
			t.Errorf("rule %d: expected ID %s, got %s", i, exp.id, rules[i].ID)
		}
		if rules[i].Status != exp.status {
			t.Errorf("rule %d: expected Status %s, got %s", i, exp.status, rules[i].Status)
		}
		if rules[i].Prefix != exp.prefix {
			t.Errorf("rule %d: expected Prefix %q, got %q", i, exp.prefix, rules[i].Prefix)
		}
		if rules[i].DestinationBucket != exp.dest {
			t.Errorf("rule %d: expected DestinationBucket %s, got %s", i, exp.dest, rules[i].DestinationBucket)
		}
	}
}

func TestGetBucketReplication_WithStorageClass(t *testing.T) {
	xmlResp := `<ReplicationConfiguration>
		<Role>arn:aws:iam::789:role/sc-role</Role>
		<Rule>
			<ID>sc-rule</ID>
			<Status>Enabled</Status>
			<Prefix></Prefix>
			<Destination>
				<Bucket>arn:aws:s3:::dest</Bucket>
				<StorageClass>STANDARD_IA</StorageClass>
			</Destination>
		</Rule>
	</ReplicationConfiguration>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(xmlResp))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	role, rules, err := client.GetBucketReplication(context.Background(), "ak", "sk", "sc-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if role != "arn:aws:iam::789:role/sc-role" {
		t.Errorf("expected role arn:aws:iam::789:role/sc-role, got %s", role)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].DestinationStorageClass != "STANDARD_IA" {
		t.Errorf("expected DestinationStorageClass STANDARD_IA, got %s", rules[0].DestinationStorageClass)
	}
	if rules[0].DestinationBucket != "arn:aws:s3:::dest" {
		t.Errorf("expected DestinationBucket arn:aws:s3:::dest, got %s", rules[0].DestinationBucket)
	}
}
