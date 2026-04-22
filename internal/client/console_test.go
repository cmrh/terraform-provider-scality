package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// newTestConsoleServer creates a mock Console API server that handles
// authentication and returns the provided handler for all other requests.
// The auth endpoint always succeeds with "test-jwt-token".
func newTestConsoleServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consoleAuthPath && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"success":true,"token":"test-jwt-token"}`))

			return
		}
		handler(w, r)
	}))
}

// --- Authenticate tests ---

func TestAuthenticate_Success(t *testing.T) {
	var capturedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != consoleAuthPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)

			return
		}

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(405)

			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true,"token":"test-jwt-token"}`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)
	// Clear any cached token from a previous test run sharing the same endpoint+user hash.
	client.token = ""

	err := client.Authenticate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.token != "test-jwt-token" {
		t.Errorf("expected token 'test-jwt-token', got %q", client.token)
	}

	if capturedBody["username"] != "admin" {
		t.Errorf("expected username 'admin', got %q", capturedBody["username"])
	}

	if capturedBody["password"] != "password123" {
		t.Errorf("expected password 'password123', got %q", capturedBody["password"])
	}
}

func TestAuthenticate_FailureResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":false,"message":"Invalid credentials"}`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "wrongpassword", false)
	err := client.Authenticate(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid credentials") {
		t.Errorf("expected error to contain 'Invalid credentials', got: %v", err)
	}
}

func TestAuthenticate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`Unauthorized`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)
	err := client.Authenticate(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention status 401, got: %v", err)
	}
}

func TestAuthenticate_TokenUsedInSubsequentCalls(t *testing.T) {
	var tokenReceived string

	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		tokenReceived = r.Header.Get("x-access-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true,"data":{"accountName":"test","email":"t@t.com","quota":0,"createdAt":"2024-01-01T00:00:00Z"}}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)
	// Token is empty, so the first API call triggers Authenticate, then uses the token.
	_, _ = client.GetConsoleAccount(context.Background(), "test")

	if tokenReceived != "test-jwt-token" {
		t.Errorf("expected x-access-token 'test-jwt-token', got %q", tokenReceived)
	}
}

// --- CreateConsoleAccount tests ---

func TestCreateConsoleAccount_Success(t *testing.T) {
	var (
		capturedToken string
		capturedBody  map[string]interface{}
	)

	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != consoleAccountPath || r.Method != "POST" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)

			return
		}

		capturedToken = r.Header.Get("x-access-token")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{
			"account": {
				"arn": "arn:aws:iam::123456:root",
				"canonicalId": "abc123",
				"id": "123456",
				"emailAddress": "test@example.com",
				"name": "test-account",
				"createDate": "2024-01-01T00:00:00Z",
				"quotaMax": 0
			}
		}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	req := ConsoleAccountCreateRequest{
		AccountName: "test-account",
		Email:       "test@example.com",
	}

	resp, err := client.CreateConsoleAccount(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the token was sent.
	if capturedToken != "test-jwt-token" {
		t.Errorf("expected x-access-token 'test-jwt-token', got %q", capturedToken)
	}

	// Verify the request body fields.
	if capturedBody["accountName"] != "test-account" {
		t.Errorf("expected accountName 'test-account', got %v", capturedBody["accountName"])
	}

	if capturedBody["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", capturedBody["email"])
	}

	// Verify all response fields.
	if resp.Account.ARN != "arn:aws:iam::123456:root" {
		t.Errorf("unexpected ARN: %q", resp.Account.ARN)
	}

	if resp.Account.CanonicalID != "abc123" {
		t.Errorf("unexpected CanonicalID: %q", resp.Account.CanonicalID)
	}

	if resp.Account.ID != "123456" {
		t.Errorf("unexpected ID: %q", resp.Account.ID)
	}

	if resp.Account.EmailAddress != "test@example.com" {
		t.Errorf("unexpected EmailAddress: %q", resp.Account.EmailAddress)
	}

	if resp.Account.Name != "test-account" {
		t.Errorf("unexpected Name: %q", resp.Account.Name)
	}

	if resp.Account.CreateDate != "2024-01-01T00:00:00Z" {
		t.Errorf("unexpected CreateDate: %q", resp.Account.CreateDate)
	}

	if resp.Account.QuotaMax != 0 {
		t.Errorf("unexpected QuotaMax: %d", resp.Account.QuotaMax)
	}
}

func TestCreateConsoleAccount_Conflict(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"message":"conflict"}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	req := ConsoleAccountCreateRequest{
		AccountName: "existing-account",
		Email:       "existing@example.com",
	}

	resp, err := client.CreateConsoleAccount(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "account already exists") {
		t.Errorf("expected 'account already exists' error, got: %v", err)
	}

	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

// --- GetConsoleAccount tests ---

func TestGetConsoleAccount_Success(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := consoleAccountPath + "/test-account"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
			w.WriteHeader(404)

			return
		}

		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
			w.WriteHeader(405)

			return
		}

		if token := r.Header.Get("x-access-token"); token != "test-jwt-token" {
			t.Errorf("expected x-access-token 'test-jwt-token', got %q", token)
			w.WriteHeader(401)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"success": true,
			"data": {
				"accountName": "test-account",
				"email": "test@example.com",
				"quota": 0,
				"createdAt": "2024-01-01T00:00:00Z"
			}
		}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	resp, err := client.GetConsoleAccount(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	if resp.Data.AccountName != "test-account" {
		t.Errorf("expected accountName 'test-account', got %q", resp.Data.AccountName)
	}

	if resp.Data.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", resp.Data.Email)
	}

	if resp.Data.Quota != 0 {
		t.Errorf("expected quota 0, got %d", resp.Data.Quota)
	}

	if resp.Data.CreatedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("expected createdAt '2024-01-01T00:00:00Z', got %q", resp.Data.CreatedAt)
	}
}

func TestGetConsoleAccount_NotFound(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	resp, err := client.GetConsoleAccount(context.Background(), "nonexistent-account")
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}

	if resp != nil {
		t.Errorf("expected nil response for 404, got %+v", resp)
	}
}

// --- DeleteConsoleAccount tests ---

func TestDeleteConsoleAccount_Success(t *testing.T) {
	var mu sync.Mutex

	var callSequence []string

	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
			w.WriteHeader(405)

			return
		}

		if token := r.Header.Get("x-access-token"); token != "test-jwt-token" {
			t.Errorf("expected x-access-token 'test-jwt-token', got %q", token)
			w.WriteHeader(401)

			return
		}

		mu.Lock()
		callSequence = append(callSequence, r.URL.Path)
		mu.Unlock()

		w.WriteHeader(200)
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	err := client.DeleteConsoleAccount(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify both requests were made in the correct order.
	if len(callSequence) != 2 {
		t.Fatalf("expected 2 DELETE calls, got %d", len(callSequence))
	}

	expectedFirst := consoleAccountPath + "/test-account"
	if callSequence[0] != expectedFirst {
		t.Errorf("expected first DELETE to %q, got %q", expectedFirst, callSequence[0])
	}

	expectedSecond := consoleAccountPath + "/test-account/user"
	if callSequence[1] != expectedSecond {
		t.Errorf("expected second DELETE to %q, got %q", expectedSecond, callSequence[1])
	}
}

func TestDeleteConsoleAccount_IdempotentWith404(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Both steps return 404 -- already deleted.
		w.WriteHeader(404)
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	err := client.DeleteConsoleAccount(context.Background(), "already-deleted")
	if err != nil {
		t.Fatalf("expected no error for idempotent 404 delete, got: %v", err)
	}
}

func TestDeleteConsoleAccount_Step1Fails(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consoleAccountPath+"/test-account" {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`internal server error`))

			return
		}

		w.WriteHeader(200)
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	err := client.DeleteConsoleAccount(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error when step 1 fails, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestDeleteConsoleAccount_Step2Fails(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consoleAccountPath+"/test-account/user" {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`internal server error`))

			return
		}

		w.WriteHeader(200)
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	err := client.DeleteConsoleAccount(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error when step 2 fails, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

// --- GenerateConsoleAccessKey tests ---

func TestGenerateConsoleAccessKey_Success(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := consoleAccountPath + "/test-account/keys"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
			w.WriteHeader(404)

			return
		}

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
			w.WriteHeader(405)

			return
		}

		if token := r.Header.Get("x-access-token"); token != "test-jwt-token" {
			t.Errorf("expected x-access-token 'test-jwt-token', got %q", token)
			w.WriteHeader(401)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"key": {
				"id": "AKIATEST123",
				"value": "secretkey123",
				"createDate": "2024-01-01T00:00:00Z",
				"status": "Active"
			}
		}`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	resp, err := client.GenerateConsoleAccessKey(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Key.ID != "AKIATEST123" {
		t.Errorf("expected key ID 'AKIATEST123', got %q", resp.Key.ID)
	}

	if resp.Key.Value != "secretkey123" {
		t.Errorf("expected key value 'secretkey123', got %q", resp.Key.Value)
	}

	if resp.Key.CreateDate != "2024-01-01T00:00:00Z" {
		t.Errorf("expected createDate '2024-01-01T00:00:00Z', got %q", resp.Key.CreateDate)
	}

	if resp.Key.Status != "Active" {
		t.Errorf("expected status 'Active', got %q", resp.Key.Status)
	}
}

func TestGenerateConsoleAccessKey_ServerError(t *testing.T) {
	server := newTestConsoleServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`internal error`))
	})
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	resp, err := client.GenerateConsoleAccessKey(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}

	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

// --- Edge case: pre-set token skips auth ---

func TestPresetToken_SkipsAuthentication(t *testing.T) {
	authCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consoleAuthPath {
			authCalled = true
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"success":true,"token":"should-not-be-used"}`))

			return
		}

		// Verify the pre-set token is used, not the one from auth.
		if token := r.Header.Get("x-access-token"); token != "preset-token" {
			t.Errorf("expected pre-set token 'preset-token', got %q", token)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true,"data":{"accountName":"x","email":"x@x.com","quota":0,"createdAt":"2024-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)
	client.token = "preset-token"

	_, err := client.GetConsoleAccount(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authCalled {
		t.Error("expected auth endpoint NOT to be called when token is pre-set")
	}
}

// --- Auth failure propagation ---

func TestCreateConsoleAccount_AuthFailurePropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`Unauthorized`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "bad", false)

	_, err := client.CreateConsoleAccount(context.Background(), ConsoleAccountCreateRequest{
		AccountName: "test",
		Email:       "test@example.com",
	})

	if err == nil {
		t.Fatal("expected error from auth failure to propagate")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got: %v", err)
	}
}

// --- Concurrency safety ---

func TestConcurrentAuthenticate_NoRace(t *testing.T) {
	authCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == consoleAuthPath {
			mu.Lock()
			authCount++
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"success":true,"token":"test-jwt-token"}`))

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true,"data":{"accountName":"x","email":"x@x.com","quota":0,"createdAt":"2024-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	client := NewConsoleClient(server.URL, "admin", "password123", false)

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetConsoleAccount(context.Background(), "test")
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent request failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if authCount != 1 {
		t.Errorf("expected exactly 1 auth call, got %d", authCount)
	}
}
