package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBucketACL_Success(t *testing.T) {
	aclXML := `<AccessControlPolicy><Owner><ID>owner123</ID><DisplayName>owner</DisplayName></Owner><AccessControlList><Grant><Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser"><ID>owner123</ID><DisplayName>owner</DisplayName></Grantee><Permission>FULL_CONTROL</Permission></Grant></AccessControlList></AccessControlPolicy>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/test-bucket" {
			t.Errorf("expected path /test-bucket, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "acl" {
			t.Errorf("expected query 'acl', got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(aclXML))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	acp, err := client.GetBucketACL(context.Background(), "ak", "sk", "test-bucket")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if acp.Owner.ID != "owner123" {
		t.Errorf("expected Owner.ID 'owner123', got %q", acp.Owner.ID)
	}
	if acp.Owner.DisplayName != "owner" {
		t.Errorf("expected Owner.DisplayName 'owner', got %q", acp.Owner.DisplayName)
	}
	if len(acp.AccessControlList.Grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(acp.AccessControlList.Grants))
	}
	grant := acp.AccessControlList.Grants[0]
	if grant.Permission != "FULL_CONTROL" {
		t.Errorf("expected Permission 'FULL_CONTROL', got %q", grant.Permission)
	}
	if grant.Grantee.ID != "owner123" {
		t.Errorf("expected Grantee.ID 'owner123', got %q", grant.Grantee.ID)
	}
}

func TestGetBucketACL_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	_, err := client.GetBucketACL(context.Background(), "ak", "sk", "test-bucket")
	if err == nil {
		t.Fatal("expected error for 403 status")
	}
}

func TestPutBucketACL_Success(t *testing.T) {
	var gotACLHeader string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		gotACLHeader = r.Header.Get("X-Amz-Acl")
		gotQuery = r.URL.RawQuery
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketACL(context.Background(), "ak", "sk", "test-bucket", "public-read")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotACLHeader != "public-read" {
		t.Errorf("expected x-amz-acl header 'public-read', got %q", gotACLHeader)
	}
	if gotQuery != "acl" {
		t.Errorf("expected query 'acl', got %q", gotQuery)
	}
}

func TestPutBucketACL_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
	}))
	defer server.Close()

	client := NewS3Client(server.URL, false)
	err := client.PutBucketACL(context.Background(), "ak", "sk", "test-bucket", "public-read")
	if err == nil {
		t.Fatal("expected error for 403 status")
	}
}
