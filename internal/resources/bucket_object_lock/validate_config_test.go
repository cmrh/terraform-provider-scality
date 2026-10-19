package bucketobjectlock

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestValidateConfig_unknownRetention guards #103: when both retention inputs come
// from variables (unknown at validate time), ValidateConfig must not raise a false
// "Conflicting retention period" — unknown is not "set".
func TestValidateConfig_unknownRetention(t *testing.T) {
	ctx := context.Background()
	var sr resource.SchemaResponse
	(&BucketObjectLockResource{}).Schema(ctx, resource.SchemaRequest{}, &sr)
	obj := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)

	vals := map[string]tftypes.Value{}
	for name, typ := range obj.AttributeTypes {
		switch name {
		case "retention_days", "retention_years":
			vals[name] = tftypes.NewValue(typ, tftypes.UnknownValue)
		default:
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(obj, vals)}

	var resp resource.ValidateConfigResponse
	(&BucketObjectLockResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown retention values must not error at validate, got: %v", resp.Diagnostics.Errors())
	}
}

// TestValidateConfig_bothSetConflicts confirms the real conflict is still caught
// when both values are known.
func TestValidateConfig_bothSetConflicts(t *testing.T) {
	ctx := context.Background()
	var sr resource.SchemaResponse
	(&BucketObjectLockResource{}).Schema(ctx, resource.SchemaRequest{}, &sr)
	obj := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)

	vals := map[string]tftypes.Value{}
	for name, typ := range obj.AttributeTypes {
		switch name {
		case "retention_days":
			vals[name] = tftypes.NewValue(typ, int64(30))
		case "retention_years":
			vals[name] = tftypes.NewValue(typ, int64(1))
		default:
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	cfg := tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(obj, vals)}

	var resp resource.ValidateConfigResponse
	(&BucketObjectLockResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("both retention values set must still conflict")
	}
}
