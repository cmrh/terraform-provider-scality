package bucket

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestValidateConfig_allUnknown locks in that ValidateConfig tolerates unknown
// values at validate time (module/variable-driven config): no attribute may be
// treated as "set" when it is unknown. Guards the #87/#103 class.
func TestValidateConfig_allUnknown(t *testing.T) {
	ctx := context.Background()
	var sr resource.SchemaResponse
	(&BucketResource{}).Schema(ctx, resource.SchemaRequest{}, &sr)
	obj := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)

	vals := map[string]tftypes.Value{}
	for name, typ := range obj.AttributeTypes {
		vals[name] = tftypes.NewValue(typ, tftypes.UnknownValue)
	}
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(obj, vals)}

	var resp resource.ValidateConfigResponse
	(&BucketResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("all-unknown config must not error at validate, got: %v", resp.Diagnostics.Errors())
	}
}
