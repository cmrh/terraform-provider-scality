package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCallerAccount(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if got := r.FormValue("Action"); got != "GetCallerIdentity" {
			t.Errorf("Action = %q, want GetCallerIdentity", got)
		}
		if got := r.FormValue("Version"); got != stsAPIVersion {
			t.Errorf("Version = %q, want %s", got, stsAPIVersion)
		}
		if auth := r.Header.Get("Authorization"); !strings.Contains(auth, "/sts/aws4_request") {
			t.Errorf("not signed for sts: %s", auth)
		}
		if got := r.Header.Get("X-Amz-Security-Token"); got != "tok" {
			t.Errorf("X-Amz-Security-Token = %q, want tok", got)
		}
		_, _ = w.Write([]byte(`<GetCallerIdentityResponse><GetCallerIdentityResult><Arn>arn:aws:sts::123456789012:assumed-role/mgr/s</Arn><Account>123456789012</Account></GetCallerIdentityResult></GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	c := NewIAMClient(server.URL, "", "", false)
	c.SessionToken = "tok"
	for range 2 {
		id, err := c.CallerAccount(context.Background(), "ak", "sk")
		if err != nil {
			t.Fatalf("CallerAccount returned error: %v", err)
		}
		if id != "123456789012" {
			t.Fatalf("account = %q, want 123456789012", id)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("want 1 request (cached), got %d", n)
	}
}

func TestCallerAccount_error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(xmlErrorResponse("InvalidAction", "not supported")))
	}))
	defer server.Close()

	c := NewIAMClient(server.URL, "", "", false)
	if _, err := c.CallerAccount(context.Background(), "ak", "sk"); err == nil {
		t.Fatal("want error")
	}
}
