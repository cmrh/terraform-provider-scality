package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testUserPolicyArn = "arn:aws:iam::123:policy/test-policy"

func TestAttachUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "AttachUserPolicy" {
			t.Errorf("Action = %q, want AttachUserPolicy", got)
		}
		if got := r.FormValue("UserName"); got != "test-user" {
			t.Errorf("UserName = %q, want test-user", got)
		}
		if got := r.FormValue("PolicyArn"); got != testUserPolicyArn {
			t.Errorf("PolicyArn = %q, want %s", got, testUserPolicyArn)
		}
		_, _ = w.Write([]byte(`<AttachUserPolicyResponse></AttachUserPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.AttachUserPolicy(context.Background(), "ak", "sk", "test-user", testUserPolicyArn); err != nil {
		t.Fatalf("AttachUserPolicy returned error: %v", err)
	}
}

func TestAttachUserPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.AttachUserPolicy(context.Background(), "ak", "sk", "test-user", testUserPolicyArn)
	if err == nil || !IsNotFound(err) {
		t.Fatalf("want NoSuchEntity error, got: %v", err)
	}
}

func TestDetachUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "DetachUserPolicy" {
			t.Errorf("Action = %q, want DetachUserPolicy", got)
		}
		if got := r.FormValue("UserName"); got != "test-user" {
			t.Errorf("UserName = %q, want test-user", got)
		}
		if got := r.FormValue("PolicyArn"); got != testUserPolicyArn {
			t.Errorf("PolicyArn = %q, want %s", got, testUserPolicyArn)
		}
		_, _ = w.Write([]byte(`<DetachUserPolicyResponse></DetachUserPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.DetachUserPolicy(context.Background(), "ak", "sk", "test-user", testUserPolicyArn); err != nil {
		t.Fatalf("DetachUserPolicy returned error: %v", err)
	}
}

func TestDetachUserPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.DetachUserPolicy(context.Background(), "ak", "sk", "test-user", testUserPolicyArn); err != nil {
		t.Fatalf("DetachUserPolicy NoSuchEntity should return nil, got: %v", err)
	}
}

func TestListAttachedUserPolicies_paginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "ListAttachedUserPolicies" {
			t.Errorf("Action = %q, want ListAttachedUserPolicies", got)
		}
		if got := r.FormValue("UserName"); got != "test-user" {
			t.Errorf("UserName = %q, want test-user", got)
		}
		switch r.FormValue("Marker") {
		case "":
			_, _ = w.Write([]byte(`<ListAttachedUserPoliciesResponse><ListAttachedUserPoliciesResult><AttachedPolicies><member><PolicyName>p1</PolicyName><PolicyArn>arn:aws:iam::123:policy/p1</PolicyArn></member></AttachedPolicies><IsTruncated>true</IsTruncated><Marker>page2</Marker></ListAttachedUserPoliciesResult></ListAttachedUserPoliciesResponse>`))
		case "page2":
			_, _ = w.Write([]byte(`<ListAttachedUserPoliciesResponse><ListAttachedUserPoliciesResult><AttachedPolicies><member><PolicyName>p2</PolicyName><PolicyArn>arn:aws:iam::123:policy/p2</PolicyArn></member></AttachedPolicies><IsTruncated>false</IsTruncated></ListAttachedUserPoliciesResult></ListAttachedUserPoliciesResponse>`))
		default:
			t.Errorf("unexpected Marker=%s", r.FormValue("Marker"))
		}
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policies, err := client.ListAttachedUserPolicies(context.Background(), "ak", "sk", "test-user")
	if err != nil {
		t.Fatalf("ListAttachedUserPolicies returned error: %v", err)
	}
	if len(policies) != 2 || policies[0].PolicyArn != "arn:aws:iam::123:policy/p1" || policies[1].PolicyArn != "arn:aws:iam::123:policy/p2" {
		t.Fatalf("unexpected policies: %+v", policies)
	}
}

func TestListAttachedUserPolicies_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.ListAttachedUserPolicies(context.Background(), "ak", "sk", "missing")
	if err == nil || !IsNotFound(err) {
		t.Fatalf("want NoSuchEntity error, got: %v", err)
	}
}
