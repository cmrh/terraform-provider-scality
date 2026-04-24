package bucketacl

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &BucketACLResource{}
var _ resource.ResourceWithImportState = &BucketACLResource{}

type BucketACLResource struct {
	client *client.S3Client
}

func NewBucketACLResource() resource.Resource {
	return &BucketACLResource{}
}

func (r *BucketACLResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_acl"
}

func (r *BucketACLResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"account_secret_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"bucket": schema.StringAttribute{
				Required:   true,
				Validators: validators.BucketName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"acl": schema.StringAttribute{
				Required:   true,
				Validators: validators.OneOf("private", "public-read", "public-read-write", "authenticated-read"),
			},
		},
	}
}

func (r *BucketACLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T.", req.ProviderData),
		)
		return
	}

	if clients.S3 == nil {
		resp.Diagnostics.AddError(
			"Missing S3 Client Configuration",
			"An IAM endpoint must be configured.",
		)
		return
	}

	r.client = clients.S3
}

func (r *BucketACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketACLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()
	acl := data.ACL.ValueString()

	tflog.Debug(ctx, "Setting bucket ACL", map[string]interface{}{
		"bucket": bucket,
		"acl":    acl,
	})

	if err := r.client.PutBucketACL(ctx, ak, sk, bucket, acl); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket ACL: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketACLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()
	acl := data.ACL.ValueString()

	if err := r.client.PutBucketACL(ctx, ak, sk, bucket, acl); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket ACL: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketACLResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Resetting bucket ACL to private", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.PutBucketACL(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
		"private",
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset bucket ACL: %s", err))
		return
	}
}

func (r *BucketACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
