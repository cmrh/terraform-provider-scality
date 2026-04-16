package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateManagedPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "CreatePolicy" {
			t.Errorf("Action = %q, want CreatePolicy", got)
		}
		if got := r.FormValue("PolicyName"); got != "test-policy" {
			t.Errorf("PolicyName = %q, want test-policy", got)
		}
		if got := r.FormValue("PolicyDocument"); got != `{"Version":"2012-10-17"}` {
			t.Errorf("PolicyDocument = %q, want {\"Version\":\"2012-10-17\"}", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<CreatePolicyResponse><CreatePolicyResult><Policy><PolicyName>test-policy</PolicyName><PolicyId>ANPATEST123</PolicyId><Arn>arn:aws:iam::123:policy/test-policy</Arn><Path>/</Path><DefaultVersionId>v1</DefaultVersionId><AttachmentCount>0</AttachmentCount></Policy></CreatePolicyResult></CreatePolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policy, err := client.CreateManagedPolicy(context.Background(), "ak", "sk", "test-policy", `{"Version":"2012-10-17"}`)
	if err != nil {
		t.Fatalf("CreateManagedPolicy returned error: %v", err)
	}
	if policy == nil {
		t.Fatal("CreateManagedPolicy returned nil policy")
	}
	if policy.PolicyName != "test-policy" {
		t.Errorf("PolicyName = %q, want test-policy", policy.PolicyName)
	}
	if policy.PolicyId != "ANPATEST123" {
		t.Errorf("PolicyId = %q, want ANPATEST123", policy.PolicyId)
	}
	if policy.Arn != "arn:aws:iam::123:policy/test-policy" {
		t.Errorf("Arn = %q, want arn:aws:iam::123:policy/test-policy", policy.Arn)
	}
	if policy.Path != "/" {
		t.Errorf("Path = %q, want /", policy.Path)
	}
	if policy.DefaultVersionId != "v1" {
		t.Errorf("DefaultVersionId = %q, want v1", policy.DefaultVersionId)
	}
	if policy.AttachmentCount != 0 {
		t.Errorf("AttachmentCount = %d, want 0", policy.AttachmentCount)
	}
}

func TestGetManagedPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "GetPolicy" {
			t.Errorf("Action = %q, want GetPolicy", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<GetPolicyResponse><GetPolicyResult><Policy><PolicyName>test-policy</PolicyName><PolicyId>ANPATEST123</PolicyId><Arn>arn:aws:iam::123:policy/test-policy</Arn><Path>/</Path><DefaultVersionId>v1</DefaultVersionId><AttachmentCount>2</AttachmentCount></Policy></GetPolicyResult></GetPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policy, err := client.GetManagedPolicy(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("GetManagedPolicy returned error: %v", err)
	}
	if policy == nil {
		t.Fatal("GetManagedPolicy returned nil policy")
	}
	if policy.PolicyName != "test-policy" {
		t.Errorf("PolicyName = %q, want test-policy", policy.PolicyName)
	}
	if policy.PolicyId != "ANPATEST123" {
		t.Errorf("PolicyId = %q, want ANPATEST123", policy.PolicyId)
	}
	if policy.Arn != "arn:aws:iam::123:policy/test-policy" {
		t.Errorf("Arn = %q, want arn:aws:iam::123:policy/test-policy", policy.Arn)
	}
	if policy.AttachmentCount != 2 {
		t.Errorf("AttachmentCount = %d, want 2", policy.AttachmentCount)
	}
}

func TestGetManagedPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>Policy arn:aws:iam::123:policy/test-policy was not found.</Message></Error></ErrorResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policy, err := client.GetManagedPolicy(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("GetManagedPolicy NoSuchEntity should not return error, got: %v", err)
	}
	if policy != nil {
		t.Errorf("GetManagedPolicy NoSuchEntity should return nil policy, got: %+v", policy)
	}
}

func TestGetManagedPolicyVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "GetPolicyVersion" {
			t.Errorf("Action = %q, want GetPolicyVersion", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}
		if got := r.FormValue("VersionId"); got != "v1" {
			t.Errorf("VersionId = %q, want v1", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<GetPolicyVersionResponse><GetPolicyVersionResult><PolicyVersion><Document>%7B%22Version%22%3A%222012-10-17%22%7D</Document><VersionId>v1</VersionId><IsDefaultVersion>true</IsDefaultVersion></PolicyVersion></GetPolicyVersionResult></GetPolicyVersionResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	doc, err := client.GetManagedPolicyVersion(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy", "v1")
	if err != nil {
		t.Fatalf("GetManagedPolicyVersion returned error: %v", err)
	}
	expected := `{"Version":"2012-10-17"}`
	if doc != expected {
		t.Errorf("Document = %q, want %q (URL-decoded)", doc, expected)
	}
}

func TestCreateManagedPolicyVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "CreatePolicyVersion" {
			t.Errorf("Action = %q, want CreatePolicyVersion", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}
		if got := r.FormValue("PolicyDocument"); got != `{"Version":"2012-10-17","Statement":[]}` {
			t.Errorf("PolicyDocument = %q, unexpected value", got)
		}
		if got := r.FormValue("SetAsDefault"); got != "true" {
			t.Errorf("SetAsDefault = %q, want true", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<CreatePolicyVersionResponse><CreatePolicyVersionResult><PolicyVersion><Document>%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%5D%7D</Document><VersionId>v2</VersionId><IsDefaultVersion>true</IsDefaultVersion></PolicyVersion></CreatePolicyVersionResult></CreatePolicyVersionResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.CreateManagedPolicyVersion(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy", `{"Version":"2012-10-17","Statement":[]}`)
	if err != nil {
		t.Fatalf("CreateManagedPolicyVersion returned error: %v", err)
	}
}

func TestDeleteManagedPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "DeletePolicy" {
			t.Errorf("Action = %q, want DeletePolicy", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<DeletePolicyResponse></DeletePolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteManagedPolicy(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("DeleteManagedPolicy returned error: %v", err)
	}
}

func TestDeleteManagedPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>Policy arn:aws:iam::123:policy/test-policy was not found.</Message></Error></ErrorResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteManagedPolicy(context.Background(), "ak", "sk", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("DeleteManagedPolicy NoSuchEntity should return nil, got: %v", err)
	}
}
