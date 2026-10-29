package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testGroupPolicyArn = "arn:aws:iam::123:policy/test-policy"

func TestAttachGroupPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "AttachGroupPolicy" {
			t.Errorf("Action = %q, want AttachGroupPolicy", got)
		}
		if got := r.FormValue("GroupName"); got != "test-group" {
			t.Errorf("GroupName = %q, want test-group", got)
		}
		if got := r.FormValue("PolicyArn"); got != testGroupPolicyArn {
			t.Errorf("PolicyArn = %q, want %s", got, testGroupPolicyArn)
		}
		_, _ = w.Write([]byte(`<AttachGroupPolicyResponse></AttachGroupPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.AttachGroupPolicy(context.Background(), "ak", "sk", "test-group", testGroupPolicyArn); err != nil {
		t.Fatalf("AttachGroupPolicy returned error: %v", err)
	}
}

func TestAttachGroupPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.AttachGroupPolicy(context.Background(), "ak", "sk", "test-group", testGroupPolicyArn)
	if err == nil || !IsNotFound(err) {
		t.Fatalf("want NoSuchEntity error, got: %v", err)
	}
}

func TestDetachGroupPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "DetachGroupPolicy" {
			t.Errorf("Action = %q, want DetachGroupPolicy", got)
		}
		if got := r.FormValue("GroupName"); got != "test-group" {
			t.Errorf("GroupName = %q, want test-group", got)
		}
		if got := r.FormValue("PolicyArn"); got != testGroupPolicyArn {
			t.Errorf("PolicyArn = %q, want %s", got, testGroupPolicyArn)
		}
		_, _ = w.Write([]byte(`<DetachGroupPolicyResponse></DetachGroupPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.DetachGroupPolicy(context.Background(), "ak", "sk", "test-group", testGroupPolicyArn); err != nil {
		t.Fatalf("DetachGroupPolicy returned error: %v", err)
	}
}

func TestDetachGroupPolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	if err := client.DetachGroupPolicy(context.Background(), "ak", "sk", "test-group", testGroupPolicyArn); err != nil {
		t.Fatalf("DetachGroupPolicy NoSuchEntity should return nil, got: %v", err)
	}
}

func TestListAttachedGroupPolicies_paginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "ListAttachedGroupPolicies" {
			t.Errorf("Action = %q, want ListAttachedGroupPolicies", got)
		}
		if got := r.FormValue("GroupName"); got != "test-group" {
			t.Errorf("GroupName = %q, want test-group", got)
		}
		switch r.FormValue("Marker") {
		case "":
			_, _ = w.Write([]byte(`<ListAttachedGroupPoliciesResponse><ListAttachedGroupPoliciesResult><AttachedPolicies><member><PolicyName>p1</PolicyName><PolicyArn>arn:aws:iam::123:policy/p1</PolicyArn></member></AttachedPolicies><IsTruncated>true</IsTruncated><Marker>page2</Marker></ListAttachedGroupPoliciesResult></ListAttachedGroupPoliciesResponse>`))
		case "page2":
			_, _ = w.Write([]byte(`<ListAttachedGroupPoliciesResponse><ListAttachedGroupPoliciesResult><AttachedPolicies><member><PolicyName>p2</PolicyName><PolicyArn>arn:aws:iam::123:policy/p2</PolicyArn></member></AttachedPolicies><IsTruncated>false</IsTruncated></ListAttachedGroupPoliciesResult></ListAttachedGroupPoliciesResponse>`))
		default:
			t.Errorf("unexpected Marker=%s", r.FormValue("Marker"))
		}
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policies, err := client.ListAttachedGroupPolicies(context.Background(), "ak", "sk", "test-group")
	if err != nil {
		t.Fatalf("ListAttachedGroupPolicies returned error: %v", err)
	}
	if len(policies) != 2 || policies[0].PolicyArn != "arn:aws:iam::123:policy/p1" || policies[1].PolicyArn != "arn:aws:iam::123:policy/p2" {
		t.Fatalf("unexpected policies: %+v", policies)
	}
}

func TestListAttachedGroupPolicies_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The request was rejected because it referenced an entity that does not exist.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.ListAttachedGroupPolicies(context.Background(), "ak", "sk", "missing")
	if err == nil || !IsNotFound(err) {
		t.Fatalf("want NoSuchEntity error, got: %v", err)
	}
}
