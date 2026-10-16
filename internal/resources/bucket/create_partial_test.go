package bucket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/resources/bucket"
)

// TestBucketCreate_partialFailurePersistsState guards #92: if a step after the
// bucket is created (here, versioning) fails, Create must persist state so the
// bucket is tracked (tainted) and the next apply replaces it, instead of
// orphaning it and failing with BucketAlreadyOwnedByYou.
func TestBucketCreate_partialFailurePersistsState(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.RawQuery, "versioning") {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>boom</Message></Error>`))
			return
		}
		w.WriteHeader(200) // bucket create (and anything else) succeeds
	}))
	defer server.Close()

	r := bucket.NewBucketResource()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	schema := sr.Schema

	clients := &client.ProviderClients{S3: client.NewS3Client(server.URL, false)}
	var cr resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: clients}, &cr)
	if cr.Diagnostics.HasError() {
		t.Fatalf("configure: %v", cr.Diagnostics.Errors())
	}

	objType := schema.Type().TerraformType(ctx).(tftypes.Object)
	plan := tftypes.NewValue(objType, map[string]tftypes.Value{
		"account_access_key":  tftypes.NewValue(tftypes.String, "ak"),
		"account_secret_key":  tftypes.NewValue(tftypes.String, "sk"),
		"bucket":              tftypes.NewValue(tftypes.String, "test-bucket"),
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, nil),
		"versioning":          tftypes.NewValue(tftypes.Bool, true),
		"tags":                tftypes.NewValue(objType.AttributeTypes["tags"], nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: plan}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a Create error from the failed versioning call")
	}
	var got types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("bucket"), &got)...)
	if got.IsNull() || got.ValueString() != "test-bucket" {
		t.Fatalf("bucket must be persisted in state after partial failure; got %q (null=%v)", got.ValueString(), got.IsNull())
	}
}
