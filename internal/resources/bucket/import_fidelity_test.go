package bucket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/resources/bucket"
)

func bucketSchema(t *testing.T) schema.Schema {
	t.Helper()
	var sr resource.SchemaResponse
	bucket.NewBucketResource().Schema(context.Background(), resource.SchemaRequest{}, &sr)
	return sr.Schema
}

func bucketValue(s schema.Schema, lock, versioning tftypes.Value) tftypes.Value {
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"account_access_key":  tftypes.NewValue(tftypes.String, "ak"),
		"account_secret_key":  tftypes.NewValue(tftypes.String, "sk"),
		"bucket":              tftypes.NewValue(tftypes.String, "test-bucket"),
		"object_lock_enabled": lock,
		"versioning":          versioning,
		"tags":                tftypes.NewValue(objType.AttributeTypes["tags"], nil),
	})
}

// TestObjectLockRequiresReplace guards #108: null and false are equivalent, so an
// imported (null) bucket never plans a replacement unless Object Lock really changes.
func TestObjectLockRequiresReplace(t *testing.T) {
	ctx := context.Background()
	s := bucketSchema(t)
	mods := s.Attributes["object_lock_enabled"].(schema.BoolAttribute).PlanModifiers

	null := tftypes.NewValue(tftypes.Bool, nil)
	unknown := tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
	tru := tftypes.NewValue(tftypes.Bool, true)
	fls := tftypes.NewValue(tftypes.Bool, false)
	toBool := func(v tftypes.Value) types.Bool {
		switch {
		case v.IsNull():
			return types.BoolNull()
		case !v.IsKnown():
			return types.BoolUnknown()
		}
		var b bool
		_ = v.As(&b)
		return types.BoolValue(b)
	}

	cases := []struct {
		name         string
		state, plan  tftypes.Value
		wantReplaced bool
	}{
		{"null to false", null, fls, false},
		{"false to null", fls, null, false},
		{"true to true", tru, tru, false},
		{"null to true", null, tru, true},
		{"true to null", tru, null, true},
		{"false to true", fls, tru, true},
		{"true to false", tru, fls, true},
		{"null to unknown", null, unknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.BoolRequest{
				Path:       path.Root("object_lock_enabled"),
				State:      tfsdk.State{Schema: s, Raw: bucketValue(s, tc.state, null)},
				Plan:       tfsdk.Plan{Schema: s, Raw: bucketValue(s, tc.plan, null)},
				StateValue: toBool(tc.state),
				PlanValue:  toBool(tc.plan),
			}
			resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
			for _, m := range mods {
				m.PlanModifyBool(ctx, req, resp)
			}
			if resp.RequiresReplace != tc.wantReplaced {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplaced)
			}
		})
	}
}

// TestBucketUpdate_versioningAlreadyEnabled guards #108: Object Lock buckets are
// already Enabled and reject PutBucketVersioning, so Update must not send it.
func TestBucketUpdate_versioningAlreadyEnabled(t *testing.T) {
	ctx := context.Background()

	var putSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "versioning") {
			w.WriteHeader(200)
			return
		}
		if r.Method == http.MethodPut {
			putSeen = true
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`<Error><Code>InvalidBucketState</Code><Message>An Object Lock configuration is present on this bucket, so the versioning state cannot be changed.</Message></Error>`))
			return
		}
		_, _ = w.Write([]byte(`<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`))
	}))
	defer server.Close()

	r := bucket.NewBucketResource()
	s := bucketSchema(t)
	clients := &client.ProviderClients{S3: client.NewS3Client(server.URL, false)}
	var cr resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: clients}, &cr)
	if cr.Diagnostics.HasError() {
		t.Fatalf("configure: %v", cr.Diagnostics.Errors())
	}

	null := tftypes.NewValue(tftypes.Bool, nil)
	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: s, Raw: bucketValue(s, null, null)},
		Plan:  tfsdk.Plan{Schema: s, Raw: bucketValue(s, null, tftypes.NewValue(tftypes.Bool, true))},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected Update error: %v", resp.Diagnostics.Errors())
	}
	if putSeen {
		t.Fatal("PutBucketVersioning sent to an already-Enabled bucket")
	}
}
