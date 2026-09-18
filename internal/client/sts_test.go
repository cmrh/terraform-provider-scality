package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssumeRole_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "AssumeRole" {
			t.Errorf("Action = %q, want AssumeRole", got)
		}
		if got := r.Form.Get("RoleArn"); got != "arn:aws:iam::123:role/demo" {
			t.Errorf("RoleArn = %q", got)
		}
		// nginx routes STS on the credential scope: the request must sign with service "sts".
		if auth := r.Header.Get("Authorization"); !strings.Contains(auth, "/sts/aws4_request") {
			t.Errorf("Authorization must carry the sts service scope, got: %s", auth)
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `<AssumeRoleResponse><AssumeRoleResult><Credentials>`+
			`<AccessKeyId>ASIATEMP</AccessKeyId>`+
			`<SecretAccessKey>tempsecret</SecretAccessKey>`+
			`<SessionToken>tok123</SessionToken>`+
			`<Expiration>2026-09-18T12:00:00Z</Expiration>`+
			`</Credentials></AssumeRoleResult></AssumeRoleResponse>`)
	}))
	defer server.Close()

	c := NewSTSClient(server.URL, false)
	creds, err := c.AssumeRole(context.Background(), "AKIABASE", "basesecret", "arn:aws:iam::123:role/demo", "terraform")
	if err != nil {
		t.Fatalf("AssumeRole returned error: %v", err)
	}
	if creds.AccessKeyID != "ASIATEMP" || creds.SecretAccessKey != "tempsecret" || creds.SessionToken != "tok123" {
		t.Errorf("unexpected creds: %+v", creds)
	}
	if creds.Expiration != "2026-09-18T12:00:00Z" {
		t.Errorf("Expiration = %q", creds.Expiration)
	}
}

func TestAssumeRole_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `<ErrorResponse><Error><Code>AccessDenied</Code>`+
			`<Message>not authorized to assume role</Message></Error></ErrorResponse>`)
	}))
	defer server.Close()

	c := NewSTSClient(server.URL, false)
	_, err := c.AssumeRole(context.Background(), "AKIABASE", "basesecret", "arn:aws:iam::123:role/demo", "terraform")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "AccessDenied" {
		t.Errorf("Code = %q, want AccessDenied", apiErr.Code)
	}
}

func TestAssumeRole_MissingCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `<AssumeRoleResponse><AssumeRoleResult></AssumeRoleResult></AssumeRoleResponse>`)
	}))
	defer server.Close()

	c := NewSTSClient(server.URL, false)
	_, err := c.AssumeRole(context.Background(), "AKIABASE", "basesecret", "arn:aws:iam::123:role/demo", "terraform")
	if err == nil {
		t.Fatal("expected error when response has no credentials, got nil")
	}
}
