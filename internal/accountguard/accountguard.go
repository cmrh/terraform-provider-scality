// Package accountguard stops account-level resources from following credentials
// into a different account. The owning account ID is kept in private state.
package accountguard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

const privateKey = "account_id"

// Getter is satisfied by a request's Private field.
type Getter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

// Setter is satisfied by a response's Private field.
type Setter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// Verify errors when the credentials act in a different account than the one
// recorded in recorded, and records the caller's account in record. Either may
// be nil. When the caller's account can't be determined, the check is skipped.
func Verify(ctx context.Context, c *client.IAMClient, accessKey, secretKey string, recorded Getter, record Setter) diag.Diagnostics {
	var diags diag.Diagnostics

	current, err := c.CallerAccount(ctx, accessKey, secretKey)
	if err != nil {
		tflog.Warn(ctx, "Skipping account check: caller account unknown", map[string]any{"error": err.Error()})
		return diags
	}

	if recorded != nil {
		raw, d := recorded.GetKey(ctx, privateKey)
		diags.Append(d...)
		var want string
		if len(raw) > 0 && json.Unmarshal(raw, &want) == nil && want != "" && want != current {
			diags.AddError(
				"Credentials Belong to a Different Account",
				fmt.Sprintf("This resource is in account %s, but the credentials in use act in account %s. "+
					"Use credentials of account %s. To move the resource to another account, remove it from state and create it there.",
					want, current, want),
			)
			return diags
		}
	}

	if record != nil {
		raw, _ := json.Marshal(current)
		diags.Append(record.SetKey(ctx, privateKey, raw)...)
	}
	return diags
}
