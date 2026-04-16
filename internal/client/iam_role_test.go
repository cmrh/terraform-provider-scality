package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "CreateRole" {
			t.Errorf("Action = %q, want CreateRole", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}
		if got := r.FormValue("AssumeRolePolicyDocument"); got != `{"Version":"2012-10-17"}` {
			t.Errorf("AssumeRolePolicyDocument = %q, want {\"Version\":\"2012-10-17\"}", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<CreateRoleResponse><CreateRoleResult><Role><RoleName>test-role</RoleName><RoleId>AROATEST123</RoleId><Arn>arn:aws:iam::123:role/test-role</Arn><Path>/</Path><AssumeRolePolicyDocument>%7B%22Version%22%3A%222012-10-17%22%7D</AssumeRolePolicyDocument></Role></CreateRoleResult></CreateRoleResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	role, err := client.CreateRole(context.Background(), "ak", "sk", "test-role", `{"Version":"2012-10-17"}`)
	if err != nil {
		t.Fatalf("CreateRole returned error: %v", err)
	}
	if role == nil {
		t.Fatal("CreateRole returned nil role")
	}
	if role.RoleName != "test-role" {
		t.Errorf("RoleName = %q, want test-role", role.RoleName)
	}
	if role.RoleId != "AROATEST123" {
		t.Errorf("RoleId = %q, want AROATEST123", role.RoleId)
	}
	if role.Arn != "arn:aws:iam::123:role/test-role" {
		t.Errorf("Arn = %q, want arn:aws:iam::123:role/test-role", role.Arn)
	}
	if role.Path != "/" {
		t.Errorf("Path = %q, want /", role.Path)
	}
	if role.AssumeRolePolicyDocument != `%7B%22Version%22%3A%222012-10-17%22%7D` {
		t.Errorf("AssumeRolePolicyDocument = %q, want URL-encoded JSON", role.AssumeRolePolicyDocument)
	}
}

func TestGetRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "GetRole" {
			t.Errorf("Action = %q, want GetRole", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<GetRoleResponse><GetRoleResult><Role><RoleName>test-role</RoleName><RoleId>AROATEST123</RoleId><Arn>arn:aws:iam::123:role/test-role</Arn><Path>/</Path><AssumeRolePolicyDocument>%7B%22Version%22%3A%222012-10-17%22%7D</AssumeRolePolicyDocument></Role></GetRoleResult></GetRoleResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	role, err := client.GetRole(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("GetRole returned error: %v", err)
	}
	if role == nil {
		t.Fatal("GetRole returned nil role")
	}
	if role.RoleName != "test-role" {
		t.Errorf("RoleName = %q, want test-role", role.RoleName)
	}
	if role.RoleId != "AROATEST123" {
		t.Errorf("RoleId = %q, want AROATEST123", role.RoleId)
	}
	if role.Arn != "arn:aws:iam::123:role/test-role" {
		t.Errorf("Arn = %q, want arn:aws:iam::123:role/test-role", role.Arn)
	}
}

func TestGetRole_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>The role with name test-role cannot be found.</Message></Error></ErrorResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	role, err := client.GetRole(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("GetRole NoSuchEntity should not return error, got: %v", err)
	}
	if role != nil {
		t.Errorf("GetRole NoSuchEntity should return nil role, got: %+v", role)
	}
}

func TestDeleteRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "DeleteRole" {
			t.Errorf("Action = %q, want DeleteRole", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<DeleteRoleResponse></DeleteRoleResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteRole(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("DeleteRole returned error: %v", err)
	}
}

func TestDeleteRole_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>The role with name test-role cannot be found.</Message></Error></ErrorResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteRole(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("DeleteRole NoSuchEntity should return nil, got: %v", err)
	}
}

func TestAttachRolePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "AttachRolePolicy" {
			t.Errorf("Action = %q, want AttachRolePolicy", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<AttachRolePolicyResponse></AttachRolePolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.AttachRolePolicy(context.Background(), "ak", "sk", "test-role", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("AttachRolePolicy returned error: %v", err)
	}
}

func TestDetachRolePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "DetachRolePolicy" {
			t.Errorf("Action = %q, want DetachRolePolicy", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}
		if got := r.FormValue("PolicyArn"); got != "arn:aws:iam::123:policy/test-policy" {
			t.Errorf("PolicyArn = %q, want arn:aws:iam::123:policy/test-policy", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<DetachRolePolicyResponse></DetachRolePolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DetachRolePolicy(context.Background(), "ak", "sk", "test-role", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("DetachRolePolicy returned error: %v", err)
	}
}

func TestDetachRolePolicy_NoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>Policy arn:aws:iam::123:policy/test-policy was not found.</Message></Error></ErrorResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DetachRolePolicy(context.Background(), "ak", "sk", "test-role", "arn:aws:iam::123:policy/test-policy")
	if err != nil {
		t.Fatalf("DetachRolePolicy NoSuchEntity should return nil, got: %v", err)
	}
}

func TestListAttachedRolePolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		if got := r.FormValue("Action"); got != "ListAttachedRolePolicies" {
			t.Errorf("Action = %q, want ListAttachedRolePolicies", got)
		}
		if got := r.FormValue("RoleName"); got != "test-role" {
			t.Errorf("RoleName = %q, want test-role", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<ListAttachedRolePoliciesResponse><ListAttachedRolePoliciesResult><AttachedPolicies><member><PolicyName>policy1</PolicyName><PolicyArn>arn:aws:iam::123:policy/policy1</PolicyArn></member><member><PolicyName>policy2</PolicyName><PolicyArn>arn:aws:iam::123:policy/policy2</PolicyArn></member></AttachedPolicies></ListAttachedRolePoliciesResult></ListAttachedRolePoliciesResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policies, err := client.ListAttachedRolePolicies(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("ListAttachedRolePolicies returned error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].PolicyName != "policy1" {
		t.Errorf("policies[0].PolicyName = %q, want policy1", policies[0].PolicyName)
	}
	if policies[0].PolicyArn != "arn:aws:iam::123:policy/policy1" {
		t.Errorf("policies[0].PolicyArn = %q, want arn:aws:iam::123:policy/policy1", policies[0].PolicyArn)
	}
	if policies[1].PolicyName != "policy2" {
		t.Errorf("policies[1].PolicyName = %q, want policy2", policies[1].PolicyName)
	}
	if policies[1].PolicyArn != "arn:aws:iam::123:policy/policy2" {
		t.Errorf("policies[1].PolicyArn = %q, want arn:aws:iam::123:policy/policy2", policies[1].PolicyArn)
	}
}

func TestListAttachedRolePolicies_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<ListAttachedRolePoliciesResponse><ListAttachedRolePoliciesResult><AttachedPolicies></AttachedPolicies></ListAttachedRolePoliciesResult></ListAttachedRolePoliciesResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	policies, err := client.ListAttachedRolePolicies(context.Background(), "ak", "sk", "test-role")
	if err != nil {
		t.Fatalf("ListAttachedRolePolicies returned error: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(policies))
	}
}
