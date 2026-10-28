package consoleaccount_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	consoleaccount "github.com/cmrh/terraform-provider-scality/internal/resources/console_account"
)

// TestConsoleAccountUpdate_passwordInputsAfterImport guards #109: setting
// password inputs on an imported account records them, keeps the computed
// values from state, and warns that no password was generated.
func TestConsoleAccountUpdate_passwordInputsAfterImport(t *testing.T) {
	ctx := context.Background()
	r := consoleaccount.NewConsoleAccountResource()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	s := sr.Schema
	objType := s.Type().TerraformType(ctx).(tftypes.Object)

	str := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	unknown := func(typ tftypes.Type) tftypes.Value { return tftypes.NewValue(typ, tftypes.UnknownValue) }
	base := map[string]tftypes.Value{
		"id":                       str("acct"),
		"account_name":             str("acct"),
		"email":                    str("acct@test.local"),
		"quota":                    tftypes.NewValue(tftypes.Number, 0),
		"generate_random_password": tftypes.NewValue(tftypes.Bool, nil),
		"password_length":          tftypes.NewValue(tftypes.Number, nil),
		"password":                 tftypes.NewValue(tftypes.String, nil),
		"created_at":               str("2026-10-28T00:00:00Z"),
		"access_key":               tftypes.NewValue(tftypes.String, nil),
		"secret_key":               tftypes.NewValue(tftypes.String, nil),
	}
	planVals := map[string]tftypes.Value{}
	for k, v := range base {
		planVals[k] = v
	}
	planVals["generate_random_password"] = tftypes.NewValue(tftypes.Bool, true)
	planVals["password_length"] = tftypes.NewValue(tftypes.Number, 20)
	planVals["password"] = unknown(tftypes.String)
	planVals["access_key"] = unknown(tftypes.String)
	planVals["secret_key"] = unknown(tftypes.String)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, base)},
		Plan:  tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objType, planVals)},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics.Errors())
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("want 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	var got consoleaccount.ConsoleAccountResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if !got.GenerateRandomPassword.ValueBool() || got.PasswordLength.ValueInt64() != 20 {
		t.Fatalf("password inputs not recorded: %v / %v", got.GenerateRandomPassword, got.PasswordLength)
	}
	if !got.Password.IsNull() || !got.AccessKey.IsNull() || got.CreatedAt.ValueString() != "2026-10-28T00:00:00Z" {
		t.Fatalf("computed values not kept from state: password=%v access_key=%v created_at=%v", got.Password, got.AccessKey, got.CreatedAt)
	}
}
