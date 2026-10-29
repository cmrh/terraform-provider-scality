package accountguard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/cmrh/terraform-provider-scality/internal/accountguard"
	"github.com/cmrh/terraform-provider-scality/internal/client"
)

type private map[string][]byte

func (p private) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return p[key], nil
}

func (p private) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	p[key] = value
	return nil
}

func callerServer(t *testing.T, status int, body string) *client.IAMClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return client.NewIAMClient(server.URL, "", "", false)
}

const identityA = `<GetCallerIdentityResponse><GetCallerIdentityResult><Account>111111111111</Account></GetCallerIdentityResult></GetCallerIdentityResponse>`

func TestVerify(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		recorded  private
		wantError bool
		wantStore string
	}{
		{"records when none recorded", http.StatusOK, identityA, private{}, false, `"111111111111"`},
		{"same account", http.StatusOK, identityA, private{"account_id": []byte(`"111111111111"`)}, false, `"111111111111"`},
		{"different account", http.StatusOK, identityA, private{"account_id": []byte(`"222222222222"`)}, true, ""},
		{"identity unavailable", http.StatusBadRequest, `<ErrorResponse><Error><Code>InvalidAction</Code></Error></ErrorResponse>`, private{"account_id": []byte(`"222222222222"`)}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := callerServer(t, tc.status, tc.body)
			store := private{}
			diags := accountguard.Verify(context.Background(), c, "ak", "sk", tc.recorded, store)
			if diags.HasError() != tc.wantError {
				t.Fatalf("HasError = %v, want %v: %v", diags.HasError(), tc.wantError, diags)
			}
			if got := string(store["account_id"]); got != tc.wantStore {
				t.Fatalf("stored %q, want %q", got, tc.wantStore)
			}
		})
	}
}

func TestVerify_nilGetterRecords(t *testing.T) {
	c := callerServer(t, http.StatusOK, identityA)
	store := private{}
	if diags := accountguard.Verify(context.Background(), c, "ak", "sk", nil, store); diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if got := string(store["account_id"]); got != `"111111111111"` {
		t.Fatalf("stored %q", got)
	}
}
