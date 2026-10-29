package iampolicy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	iampolicy "github.com/cmrh/terraform-provider-scality/internal/resources/iam_policy"
)

// TestIAMPolicyUpdate_credentialsOnly guards #118: a credential change must not
// create a policy version for an unchanged document.
func TestIAMPolicyUpdate_credentialsOnly(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		switch action := r.FormValue("Action"); action {
		case "GetCallerIdentity": // account check skipped
			w.WriteHeader(http.StatusBadRequest)
		default:
			t.Errorf("unexpected %s", action)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	r := iampolicy.NewIAMPolicyResource()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := sr.Schema
	clients := &client.ProviderClients{IAM: client.NewIAMClient(server.URL, "", "", false)}
	var cr resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: clients}, &cr)
	if cr.Diagnostics.HasError() {
		t.Fatalf("configure: %v", cr.Diagnostics.Errors())
	}

	objType := s.Type().TerraformType(ctx).(tftypes.Object)
	value := func(ak, doc string) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"account_access_key": tftypes.NewValue(tftypes.String, ak),
			"account_secret_key": tftypes.NewValue(tftypes.String, "sk"),
			"policy_name":        tftypes.NewValue(tftypes.String, "p"),
			"policy_document":    tftypes.NewValue(tftypes.String, doc),
			"arn":                tftypes.NewValue(tftypes.String, "arn:aws:iam::111111111111:policy/p"),
		})
	}
	const doc = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: s, Raw: value("old-ak", doc)},
		Plan:  tfsdk.Plan{Schema: s, Raw: value("new-ak", doc)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected Update error: %v", resp.Diagnostics.Errors())
	}
	var got iampolicy.IAMPolicyResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.AccountAccessKey.ValueString() != "new-ak" {
		t.Fatalf("credentials not recorded: %v", got.AccountAccessKey)
	}
}
