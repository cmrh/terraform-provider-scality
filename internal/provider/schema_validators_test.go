package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func getSchema(t *testing.T, newResource func() resource.Resource) schema.Schema {
	t.Helper()
	r := newResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", schemaResp.Diagnostics)
	}
	return schemaResp.Schema
}

func countStringValidators(t *testing.T, s schema.Schema, attr string) int {
	t.Helper()
	a, ok := s.Attributes[attr]
	if !ok {
		t.Fatalf("attribute %q not found in schema", attr)
	}
	sa, ok := a.(schema.StringAttribute)
	if !ok {
		t.Fatalf("attribute %q is not a StringAttribute", attr)
	}
	return len(sa.Validators)
}

func countInt64Validators(t *testing.T, s schema.Schema, attr string) int {
	t.Helper()
	a, ok := s.Attributes[attr]
	if !ok {
		t.Fatalf("attribute %q not found in schema", attr)
	}
	ia, ok := a.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("attribute %q is not an Int64Attribute", attr)
	}
	return len(ia.Validators)
}

func countNestedStringValidators(t *testing.T, s schema.Schema, blockName, attr string) int {
	t.Helper()
	b, ok := s.Blocks[blockName]
	if !ok {
		t.Fatalf("block %q not found in schema", blockName)
	}
	lb, ok := b.(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("block %q is not a ListNestedBlock", blockName)
	}
	a, ok := lb.NestedObject.Attributes[attr]
	if !ok {
		t.Fatalf("attribute %q not found in block %q", attr, blockName)
	}
	sa, ok := a.(schema.StringAttribute)
	if !ok {
		t.Fatalf("attribute %q in block %q is not a StringAttribute", attr, blockName)
	}
	return len(sa.Validators)
}

// TestSchemaValidatorsWired verifies that each resource attribute that should
// have input validators actually has them wired up. If you add a new resource
// or validated field, add a row to this table.
func TestSchemaValidatorsWired(t *testing.T) {
	p := New("test")()

	resFuncs := p.(*ScalityProvider).resourceFactories()

	// Build a lookup from type name → constructor.
	registry := make(map[string]func() resource.Resource, len(resFuncs))
	for _, fn := range resFuncs {
		r := fn()
		metaReq := resource.MetadataRequest{ProviderTypeName: "scality"}
		metaResp := &resource.MetadataResponse{}
		r.Metadata(context.Background(), metaReq, metaResp)
		registry[metaResp.TypeName] = fn
	}

	// ---------------------------------------------------------------
	// TABLE: one row per validated attribute.
	//
	// To maintain: when you add validators to a resource field, add a
	// row here. The test will fail if the validator count drops below
	// minCount, catching accidental removals.
	// ---------------------------------------------------------------
	tests := []struct {
		resource  string // type name, e.g. "scality_bucket"
		attribute string // "field" or "block.field" for nested blocks
		kind      string // "string", "int64", or "nested_string"
		minCount  int    // minimum validators expected
	}{
		// --- account ---
		{"scality_account", "name", "string", 2},
		{"scality_account", "email_address", "string", 1},

		// --- console_account ---
		{"scality_console_account", "account_name", "string", 2},
		{"scality_console_account", "email", "string", 1},
		{"scality_console_account", "password_length", "int64", 1},

		// --- user ---
		{"scality_user", "username", "string", 2},

		// --- user_policy ---
		{"scality_user_policy", "username", "string", 2},
		{"scality_user_policy", "policy_name", "string", 2},
		{"scality_user_policy", "policy_document", "string", 1},

		// --- group ---
		{"scality_group", "group_name", "string", 2},

		// --- iam_policy ---
		{"scality_iam_policy", "policy_name", "string", 2},
		{"scality_iam_policy", "policy_document", "string", 1},

		// --- iam_role ---
		{"scality_iam_role", "role_name", "string", 2},
		{"scality_iam_role", "assume_role_policy", "string", 1},

		// --- iam_role_policy_attachment ---
		{"scality_iam_role_policy_attachment", "role_name", "string", 2},
		{"scality_iam_role_policy_attachment", "policy_arn", "string", 1},

		// --- bucket ---
		{"scality_bucket", "bucket", "string", 2},

		// --- bucket_encryption ---
		{"scality_bucket_encryption", "bucket", "string", 2},
		{"scality_bucket_encryption", "sse_algorithm", "string", 1},

		// --- bucket_object_lock ---
		{"scality_bucket_object_lock", "bucket", "string", 2},
		{"scality_bucket_object_lock", "retention_mode", "string", 1},

		// --- bucket_lifecycle (nested block) ---
		{"scality_bucket_lifecycle", "bucket", "string", 2},
		{"scality_bucket_lifecycle", "rule.status", "nested_string", 1},

		// --- bucket_policy ---
		{"scality_bucket_policy", "bucket", "string", 2},
		{"scality_bucket_policy", "policy", "string", 1},

		// --- bucket_replication (nested block) ---
		{"scality_bucket_replication", "bucket", "string", 2},
		{"scality_bucket_replication", "rule.status", "nested_string", 1},
	}

	for _, tt := range tests {
		t.Run(tt.resource+"/"+tt.attribute, func(t *testing.T) {
			fn, ok := registry[tt.resource]
			if !ok {
				t.Fatalf("resource %q not registered in provider", tt.resource)
			}
			s := getSchema(t, fn)

			var got int
			switch tt.kind {
			case "string":
				got = countStringValidators(t, s, tt.attribute)
			case "int64":
				got = countInt64Validators(t, s, tt.attribute)
			case "nested_string":
				parts := splitBlockAttr(tt.attribute)
				got = countNestedStringValidators(t, s, parts[0], parts[1])
			default:
				t.Fatalf("unknown kind %q", tt.kind)
			}

			if got < tt.minCount {
				t.Errorf("%s.%s: expected at least %d validator(s), got %d", tt.resource, tt.attribute, tt.minCount, got)
			}
		})
	}
}

func splitBlockAttr(s string) [2]string {
	for i, c := range s {
		if c == '.' {
			return [2]string{s[:i], s[i+1:]}
		}
	}
	return [2]string{s, ""}
}
