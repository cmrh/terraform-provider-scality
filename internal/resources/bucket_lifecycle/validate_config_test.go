package bucketlifecycle

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// schemaObjectType returns the resource's config object type for building raw
// tftypes config values.
func schemaObjectType(ctx context.Context, t *testing.T) (tfsdk.Config, tftypes.Object) {
	t.Helper()
	var sr resource.SchemaResponse
	(&BucketLifecycleResource{}).Schema(ctx, resource.SchemaRequest{}, &sr)
	obj := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)
	return tfsdk.Config{Schema: sr.Schema}, obj
}

// TestValidateConfig_unknownRuleList reproduces the 1.1.2/1.1.3 regression: a
// dynamic rule block over an unknown for_each makes the whole list unknown,
// which must not error at validate time (it did — Config.Get can't decode an
// unknown list into a Go slice).
func TestValidateConfig_unknownRuleList(t *testing.T) {
	ctx := context.Background()
	cfg, obj := schemaObjectType(ctx, t)

	vals := map[string]tftypes.Value{}
	for name, typ := range obj.AttributeTypes {
		if name == "rule" {
			vals[name] = tftypes.NewValue(typ, tftypes.UnknownValue)
		} else {
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	cfg.Raw = tftypes.NewValue(obj, vals)

	var resp resource.ValidateConfigResponse
	(&BucketLifecycleResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown rule list must not error at validate, got: %v", resp.Diagnostics.Errors())
	}
}

// TestValidateConfig_unknownActionValue guards the per-rule unknown check: a rule
// whose only action (expiration_days) is unknown must not be flagged "no action".
func TestValidateConfig_unknownActionValue(t *testing.T) {
	ctx := context.Background()
	cfg, obj := schemaObjectType(ctx, t)

	elemType := obj.AttributeTypes["rule"].(tftypes.List).ElementType.(tftypes.Object)
	elemVals := map[string]tftypes.Value{}
	for name, typ := range elemType.AttributeTypes {
		switch name {
		case "status":
			elemVals[name] = tftypes.NewValue(typ, "Enabled")
		case "expiration_days":
			elemVals[name] = tftypes.NewValue(typ, tftypes.UnknownValue)
		default:
			elemVals[name] = tftypes.NewValue(typ, nil)
		}
	}
	ruleList := tftypes.NewValue(obj.AttributeTypes["rule"], []tftypes.Value{
		tftypes.NewValue(elemType, elemVals),
	})

	vals := map[string]tftypes.Value{}
	for name, typ := range obj.AttributeTypes {
		if name == "rule" {
			vals[name] = ruleList
		} else {
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	cfg.Raw = tftypes.NewValue(obj, vals)

	var resp resource.ValidateConfigResponse
	(&BucketLifecycleResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("rule with an unknown action value must not error, got: %v", resp.Diagnostics.Errors())
	}
}
