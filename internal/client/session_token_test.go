package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captures the Authorization + token header a client sends.
func captureHeaders(t *testing.T, body string) (*string, *string, *httptest.Server) {
	t.Helper()
	var gotToken, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Amz-Security-Token")
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		fmt.Fprint(w, body)
	}))
	return &gotToken, &gotAuth, server
}

func TestPerAccountIAM_SessionToken(t *testing.T) {
	const listUsers = `<ListUsersResponse><ListUsersResult><Users></Users><IsTruncated>false</IsTruncated></ListUsersResult></ListUsersResponse>`

	t.Run("with token", func(t *testing.T) {
		gotToken, gotAuth, server := captureHeaders(t, listUsers)
		defer server.Close()
		c := NewIAMClient(server.URL, "", "", false)
		c.SessionToken = "tok-abc"
		if _, err := c.ListUsers(context.Background(), "AKIA", "sk"); err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if *gotToken != "tok-abc" {
			t.Errorf("X-Amz-Security-Token = %q, want tok-abc", *gotToken)
		}
		if !strings.Contains(*gotAuth, "x-amz-security-token") {
			t.Errorf("SignedHeaders must include x-amz-security-token; got: %s", *gotAuth)
		}
	})

	t.Run("without token", func(t *testing.T) {
		gotToken, gotAuth, server := captureHeaders(t, listUsers)
		defer server.Close()
		c := NewIAMClient(server.URL, "", "", false)
		if _, err := c.ListUsers(context.Background(), "AKIA", "sk"); err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if *gotToken != "" {
			t.Errorf("X-Amz-Security-Token should be absent, got %q", *gotToken)
		}
		if strings.Contains(*gotAuth, "x-amz-security-token") {
			t.Errorf("SignedHeaders must not include x-amz-security-token; got: %s", *gotAuth)
		}
	})
}

func TestS3_SessionToken(t *testing.T) {
	gotToken, gotAuth, server := captureHeaders(t, "")
	defer server.Close()
	c := NewS3Client(server.URL, false)
	c.SessionToken = "tok-s3"
	if _, err := c.HeadBucket(context.Background(), "AKIA", "sk", "b"); err != nil {
		t.Fatalf("HeadBucket: %v", err)
	}
	if *gotToken != "tok-s3" {
		t.Errorf("X-Amz-Security-Token = %q, want tok-s3", *gotToken)
	}
	if !strings.Contains(*gotAuth, "x-amz-security-token") {
		t.Errorf("SignedHeaders must include x-amz-security-token; got: %s", *gotAuth)
	}
}
