package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// CreateAccount
// ---------------------------------------------------------------------------

func TestCreateAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateAccount" {
			t.Errorf("Action = %q, want CreateAccount", got)
		}
		if got := r.Form.Get("name"); got != "test-account" {
			t.Errorf("name = %q, want test-account", got)
		}
		if got := r.Form.Get("emailAddress"); got != "test@example.com" {
			t.Errorf("emailAddress = %q, want test@example.com", got)
		}

		w.WriteHeader(201)
		fmt.Fprint(w, `{
			"account": {
				"data": {
					"arn": "arn:aws:iam::123456:root",
					"canonicalId": "abc123",
					"id": "123456",
					"emailAddress": "test@example.com",
					"name": "test-account",
					"createDate": "2024-01-01T00:00:00Z",
					"quotaMax": 0
				}
			}
		}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.CreateAccount(context.Background(), AccountCreateRequest{
		Name:         "test-account",
		EmailAddress: "test@example.com",
	})
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if resp.Account.Data.ARN != "arn:aws:iam::123456:root" {
		t.Errorf("ARN = %q, want arn:aws:iam::123456:root", resp.Account.Data.ARN)
	}
	if resp.Account.Data.CanonicalID != "abc123" {
		t.Errorf("CanonicalID = %q, want abc123", resp.Account.Data.CanonicalID)
	}
	if resp.Account.Data.ID != "123456" {
		t.Errorf("ID = %q, want 123456", resp.Account.Data.ID)
	}
	if resp.Account.Data.EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want test@example.com", resp.Account.Data.EmailAddress)
	}
	if resp.Account.Data.Name != "test-account" {
		t.Errorf("Name = %q, want test-account", resp.Account.Data.Name)
	}
	if resp.Account.Data.CreateDate != "2024-01-01T00:00:00Z" {
		t.Errorf("CreateDate = %q, want 2024-01-01T00:00:00Z", resp.Account.Data.CreateDate)
	}
	if resp.Account.Data.QuotaMax != 0 {
		t.Errorf("QuotaMax = %d, want 0", resp.Account.Data.QuotaMax)
	}
}

func TestCreateAccount_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		fmt.Fprint(w, `{"error":"EntityAlreadyExists"}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateAccount(context.Background(), AccountCreateRequest{
		Name:         "existing-account",
		EmailAddress: "dup@example.com",
	})
	if err == nil {
		t.Fatal("expected error for 409 status, got nil")
	}
	if !strings.Contains(err.Error(), "account already exists") {
		t.Errorf("error = %q, want it to contain 'account already exists'", err.Error())
	}
}

func TestCreateAccount_WithQuotaMax(t *testing.T) {
	var capturedQuotaMax string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		capturedQuotaMax = r.Form.Get("quotaMax")

		w.WriteHeader(201)
		fmt.Fprint(w, `{
			"account": {
				"data": {
					"arn": "arn:aws:iam::123456:root",
					"canonicalId": "abc123",
					"id": "123456",
					"emailAddress": "quota@example.com",
					"name": "quota-account",
					"createDate": "2024-01-01T00:00:00Z",
					"quotaMax": 1073741824
				}
			}
		}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.CreateAccount(context.Background(), AccountCreateRequest{
		Name:         "quota-account",
		EmailAddress: "quota@example.com",
		QuotaMax:     1073741824,
	})
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if capturedQuotaMax != "1073741824" {
		t.Errorf("quotaMax param = %q, want 1073741824", capturedQuotaMax)
	}
	if resp.Account.Data.QuotaMax != 1073741824 {
		t.Errorf("QuotaMax = %d, want 1073741824", resp.Account.Data.QuotaMax)
	}
}

func TestCreateAccount_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `internal server error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateAccount(context.Background(), AccountCreateRequest{
		Name:         "fail-account",
		EmailAddress: "fail@example.com",
	})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// GetAccount
// ---------------------------------------------------------------------------

func TestGetAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GetAccount" {
			t.Errorf("Action = %q, want GetAccount", got)
		}
		if got := r.Form.Get("accountName"); got != "test-account" {
			t.Errorf("accountName = %q, want test-account", got)
		}

		w.WriteHeader(200)
		fmt.Fprint(w, `{
			"arn": "arn:aws:iam::123456:root",
			"canonicalId": "abc123",
			"id": "123456",
			"emailAddress": "test@example.com",
			"name": "test-account",
			"createDate": "2024-01-01T00:00:00Z",
			"quotaMax": 0
		}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.GetAccount(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("GetAccount returned nil response")
	}
	if resp.ARN != "arn:aws:iam::123456:root" {
		t.Errorf("ARN = %q, want arn:aws:iam::123456:root", resp.ARN)
	}
	if resp.CanonicalID != "abc123" {
		t.Errorf("CanonicalID = %q, want abc123", resp.CanonicalID)
	}
	if resp.ID != "123456" {
		t.Errorf("ID = %q, want 123456", resp.ID)
	}
	if resp.EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want test@example.com", resp.EmailAddress)
	}
	if resp.Name != "test-account" {
		t.Errorf("Name = %q, want test-account", resp.Name)
	}
	if resp.CreateDate != "2024-01-01T00:00:00Z" {
		t.Errorf("CreateDate = %q, want 2024-01-01T00:00:00Z", resp.CreateDate)
	}
	if resp.QuotaMax != 0 {
		t.Errorf("QuotaMax = %d, want 0", resp.QuotaMax)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":"NoSuchEntity"}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.GetAccount(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for 404, got %+v", resp)
	}
}

func TestGetAccount_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `server error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.GetAccount(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// DeleteAccount
// ---------------------------------------------------------------------------

func TestDeleteAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteAccount" {
			t.Errorf("Action = %q, want DeleteAccount", got)
		}
		if got := r.Form.Get("AccountName"); got != "test-account" {
			t.Errorf("AccountName = %q, want test-account", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccount(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
}

func TestDeleteAccount_NotFound_Idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":"NoSuchEntity"}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccount(context.Background(), "already-gone")
	if err != nil {
		t.Fatalf("DeleteAccount should succeed on 404 (idempotent), got error: %v", err)
	}
}

func TestDeleteAccount_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		fmt.Fprint(w, `{"error":"DeleteConflict"}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccount(context.Background(), "busy-account")
	if err == nil {
		t.Fatal("expected error for 409 status, got nil")
	}
	if !strings.Contains(err.Error(), "cannot delete account") {
		t.Errorf("error = %q, want it to contain 'cannot delete account'", err.Error())
	}
	if !strings.Contains(err.Error(), "busy-account") {
		t.Errorf("error = %q, want it to contain the account name 'busy-account'", err.Error())
	}
	if !strings.Contains(err.Error(), "S3 buckets") {
		t.Errorf("error = %q, want it to contain helpful guidance about 'S3 buckets'", err.Error())
	}
}

func TestDeleteAccount_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `internal error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccount(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// GenerateAccountAccessKey
// ---------------------------------------------------------------------------

func TestGenerateAccountAccessKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GenerateAccountAccessKey" {
			t.Errorf("Action = %q, want GenerateAccountAccessKey", got)
		}
		if got := r.Form.Get("AccountName"); got != "test-account" {
			t.Errorf("AccountName = %q, want test-account", got)
		}

		w.WriteHeader(201)
		fmt.Fprint(w, `{
			"data": {
				"id": "AKIATEST123",
				"value": "secretkey123",
				"createDate": "2024-01-01T00:00:00Z",
				"status": "Active",
				"userId": "123456"
			}
		}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.GenerateAccountAccessKey(context.Background(), "test-account")
	if err != nil {
		t.Fatalf("GenerateAccountAccessKey returned error: %v", err)
	}
	if resp.Data.ID != "AKIATEST123" {
		t.Errorf("ID = %q, want AKIATEST123", resp.Data.ID)
	}
	if resp.Data.Value != "secretkey123" {
		t.Errorf("Value = %q, want secretkey123", resp.Data.Value)
	}
	if resp.Data.CreateDate != "2024-01-01T00:00:00Z" {
		t.Errorf("CreateDate = %q, want 2024-01-01T00:00:00Z", resp.Data.CreateDate)
	}
	if resp.Data.Status != "Active" {
		t.Errorf("Status = %q, want Active", resp.Data.Status)
	}
	if resp.Data.UserID != "123456" {
		t.Errorf("UserID = %q, want 123456", resp.Data.UserID)
	}
}

func TestGenerateAccountAccessKey_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.GenerateAccountAccessKey(context.Background(), "test-account")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// DeleteAccountAccessKey
// ---------------------------------------------------------------------------

func TestDeleteAccountAccessKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteAccessKey" {
			t.Errorf("Action = %q, want DeleteAccessKey", got)
		}
		if got := r.Form.Get("AccessKeyId"); got != "AKIATEST123" {
			t.Errorf("AccessKeyId = %q, want AKIATEST123", got)
		}
		if got := r.Form.Get("AccountName"); got != "test-account" {
			t.Errorf("AccountName = %q, want test-account", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccountAccessKey(context.Background(), "AKIATEST123", "test-account")
	if err != nil {
		t.Fatalf("DeleteAccountAccessKey returned error: %v", err)
	}
}

func TestDeleteAccountAccessKey_NotFound_Idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":"NoSuchEntity"}`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccountAccessKey(context.Background(), "AKIANOTEXIST", "test-account")
	if err != nil {
		t.Fatalf("DeleteAccountAccessKey should succeed on 404 (idempotent), got error: %v", err)
	}
}

func TestDeleteAccountAccessKey_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteAccountAccessKey(context.Background(), "AKIATEST", "test-account")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// UpdateAccountAttributes
// ---------------------------------------------------------------------------

func TestUpdateAccountAttributes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "UpdateAccountAttributes" {
			t.Errorf("Action = %q, want UpdateAccountAttributes", got)
		}
		if got := r.Form.Get("name"); got != "test-account" {
			t.Errorf("name = %q, want test-account", got)
		}
		customAttrs := r.Form.Get("customAttributes")
		if customAttrs == "" {
			t.Error("customAttributes param is empty")
		}
		// Verify JSON is valid and contains expected keys.
		if !strings.Contains(customAttrs, "env") {
			t.Errorf("customAttributes = %q, want it to contain 'env'", customAttrs)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.UpdateAccountAttributes(context.Background(), "test-account", map[string]string{
		"env":  "production",
		"team": "platform",
	})
	if err != nil {
		t.Fatalf("UpdateAccountAttributes returned error: %v", err)
	}
}

func TestUpdateAccountAttributes_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `error`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.UpdateAccountAttributes(context.Background(), "test-account", map[string]string{
		"key": "value",
	})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want it to contain 'unexpected status 500'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// CreateRootAccessKey (per-account auth, XML response)
// ---------------------------------------------------------------------------

func TestCreateRootAccessKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateAccessKey" {
			t.Errorf("Action = %q, want CreateAccessKey", got)
		}
		// Root access key: no UserName param.
		if got := r.Form.Get("UserName"); got != "" {
			t.Errorf("UserName = %q, want empty (root key)", got)
		}

		w.WriteHeader(200)
		fmt.Fprint(w, `<CreateAccessKeyResponse>
			<CreateAccessKeyResult>
				<AccessKey>
					<AccessKeyId>AKIATEST</AccessKeyId>
					<SecretAccessKey>secret123</SecretAccessKey>
					<Status>Active</Status>
				</AccessKey>
			</CreateAccessKeyResult>
		</CreateAccessKeyResponse>`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	resp, err := client.CreateRootAccessKey(context.Background(), "account-ak", "account-sk")
	if err != nil {
		t.Fatalf("CreateRootAccessKey returned error: %v", err)
	}
	if resp.AccessKeyId != "AKIATEST" {
		t.Errorf("AccessKeyId = %q, want AKIATEST", resp.AccessKeyId)
	}
	if resp.SecretAccessKey != "secret123" {
		t.Errorf("SecretAccessKey = %q, want secret123", resp.SecretAccessKey)
	}
	if resp.Status != "Active" {
		t.Errorf("Status = %q, want Active", resp.Status)
	}
}

func TestCreateRootAccessKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `<ErrorResponse><Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error></ErrorResponse>`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	_, err := client.CreateRootAccessKey(context.Background(), "bad-ak", "bad-sk")
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("error = %q, want it to contain 'AccessDenied'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// ListRootAccessKeys (per-account auth, XML response)
// ---------------------------------------------------------------------------

func TestListRootAccessKeys_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "ListAccessKeys" {
			t.Errorf("Action = %q, want ListAccessKeys", got)
		}
		if got := r.Form.Get("UserName"); got != "" {
			t.Errorf("UserName = %q, want empty (root keys)", got)
		}

		w.WriteHeader(200)
		fmt.Fprint(w, `<ListAccessKeysResponse>
			<ListAccessKeysResult>
				<AccessKeyMetadata>
					<member>
						<AccessKeyId>AKIATEST1</AccessKeyId>
						<Status>Active</Status>
					</member>
					<member>
						<AccessKeyId>AKIATEST2</AccessKeyId>
						<Status>Inactive</Status>
					</member>
				</AccessKeyMetadata>
			</ListAccessKeysResult>
		</ListAccessKeysResponse>`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	keys, err := client.ListRootAccessKeys(context.Background(), "account-ak", "account-sk")
	if err != nil {
		t.Fatalf("ListRootAccessKeys returned error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(keys))
	}
	if keys[0].AccessKeyId != "AKIATEST1" {
		t.Errorf("keys[0].AccessKeyId = %q, want AKIATEST1", keys[0].AccessKeyId)
	}
	if keys[0].Status != "Active" {
		t.Errorf("keys[0].Status = %q, want Active", keys[0].Status)
	}
	if keys[1].AccessKeyId != "AKIATEST2" {
		t.Errorf("keys[1].AccessKeyId = %q, want AKIATEST2", keys[1].AccessKeyId)
	}
	if keys[1].Status != "Inactive" {
		t.Errorf("keys[1].Status = %q, want Inactive", keys[1].Status)
	}
}

func TestListRootAccessKeys_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `<ListAccessKeysResponse>
			<ListAccessKeysResult>
				<AccessKeyMetadata/>
			</ListAccessKeysResult>
		</ListAccessKeysResponse>`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	keys, err := client.ListRootAccessKeys(context.Background(), "account-ak", "account-sk")
	if err != nil {
		t.Fatalf("ListRootAccessKeys returned error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("len(keys) = %d, want 0", len(keys))
	}
}

// ---------------------------------------------------------------------------
// DeleteRootAccessKey (per-account auth, XML response)
// ---------------------------------------------------------------------------

func TestDeleteRootAccessKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteAccessKey" {
			t.Errorf("Action = %q, want DeleteAccessKey", got)
		}
		if got := r.Form.Get("AccessKeyId"); got != "AKIATEST" {
			t.Errorf("AccessKeyId = %q, want AKIATEST", got)
		}
		if got := r.Form.Get("UserName"); got != "" {
			t.Errorf("UserName = %q, want empty (root key deletion)", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteRootAccessKey(context.Background(), "account-ak", "account-sk", "AKIATEST")
	if err != nil {
		t.Fatalf("DeleteRootAccessKey returned error: %v", err)
	}
}

func TestDeleteRootAccessKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `<ErrorResponse><Error><Code>NoSuchEntity</Code><Message>The Access Key does not exist</Message></Error></ErrorResponse>`)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	err := client.DeleteRootAccessKey(context.Background(), "account-ak", "account-sk", "AKIANOTEXIST")
	if err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
	if !strings.Contains(err.Error(), "NoSuchEntity") {
		t.Errorf("error = %q, want it to contain 'NoSuchEntity'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Multi-action mock server (integration-style test)
// ---------------------------------------------------------------------------

func TestAccountLifecycle(t *testing.T) {
	var callSequence []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		action := r.Form.Get("Action")
		callSequence = append(callSequence, action)

		switch action {
		case "CreateAccount":
			w.WriteHeader(201)
			fmt.Fprint(w, `{
				"account": {
					"data": {
						"arn": "arn:aws:iam::999:root",
						"canonicalId": "cid999",
						"id": "999",
						"emailAddress": "lifecycle@example.com",
						"name": "lifecycle-account",
						"createDate": "2024-06-01T00:00:00Z",
						"quotaMax": 0
					}
				}
			}`)
		case "GetAccount":
			w.WriteHeader(200)
			fmt.Fprint(w, `{
				"arn": "arn:aws:iam::999:root",
				"canonicalId": "cid999",
				"id": "999",
				"emailAddress": "lifecycle@example.com",
				"name": "lifecycle-account",
				"createDate": "2024-06-01T00:00:00Z",
				"quotaMax": 0
			}`)
		case "GenerateAccountAccessKey":
			w.WriteHeader(201)
			fmt.Fprint(w, `{
				"data": {
					"id": "AKIALIFECYCLE",
					"value": "lifecycle-secret",
					"createDate": "2024-06-01T00:00:00Z",
					"status": "Active",
					"userId": "999"
				}
			}`)
		case "DeleteAccessKey":
			w.WriteHeader(200)
		case "DeleteAccount":
			w.WriteHeader(200)
		default:
			w.WriteHeader(400)
			fmt.Fprintf(w, `{"error":"unknown action: %s"}`, action)
		}
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "admin-ak", "admin-sk", false)
	ctx := context.Background()

	// Step 1: Create account.
	createResp, err := client.CreateAccount(ctx, AccountCreateRequest{
		Name:         "lifecycle-account",
		EmailAddress: "lifecycle@example.com",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if createResp.Account.Data.ID != "999" {
		t.Errorf("created account ID = %q, want 999", createResp.Account.Data.ID)
	}

	// Step 2: Get account.
	getResp, err := client.GetAccount(ctx, "lifecycle-account")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if getResp.Name != "lifecycle-account" {
		t.Errorf("account name = %q, want lifecycle-account", getResp.Name)
	}

	// Step 3: Generate access key.
	keyResp, err := client.GenerateAccountAccessKey(ctx, "lifecycle-account")
	if err != nil {
		t.Fatalf("GenerateAccountAccessKey: %v", err)
	}
	if keyResp.Data.ID != "AKIALIFECYCLE" {
		t.Errorf("access key ID = %q, want AKIALIFECYCLE", keyResp.Data.ID)
	}

	// Step 4: Delete access key.
	err = client.DeleteAccountAccessKey(ctx, "AKIALIFECYCLE", "lifecycle-account")
	if err != nil {
		t.Fatalf("DeleteAccountAccessKey: %v", err)
	}

	// Step 5: Delete account.
	err = client.DeleteAccount(ctx, "lifecycle-account")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Verify call sequence.
	expected := []string{
		"CreateAccount",
		"GetAccount",
		"GenerateAccountAccessKey",
		"DeleteAccessKey",
		"DeleteAccount",
	}
	if len(callSequence) != len(expected) {
		t.Fatalf("call count = %d, want %d; calls = %v", len(callSequence), len(expected), callSequence)
	}
	for i, want := range expected {
		if callSequence[i] != want {
			t.Errorf("call[%d] = %q, want %q", i, callSequence[i], want)
		}
	}
}
