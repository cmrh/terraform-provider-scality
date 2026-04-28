package bucketpolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &BucketPolicyResource{}
var _ resource.ResourceWithImportState = &BucketPolicyResource{}

type BucketPolicyResource struct {
	client *client.S3Client
}

func NewBucketPolicyResource() resource.Resource {
	return &BucketPolicyResource{}
}

func (r *BucketPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_policy"
}

func (r *BucketPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an S3 bucket policy within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this bucket",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this bucket",
				Required:            true,
				Sensitive:           true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "Name of the S3 bucket",
				Required:            true,
				Validators:          validators.BucketName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy": schema.StringAttribute{
				MarkdownDescription: "JSON policy document",
				Required:            true,
				Validators:          validators.JSONDocument(),
			},
		},
	}
}

func (r *BucketPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"S3 API credentials (endpoint, access_key, secret_key) must be configured to use scality_bucket_policy resource.",
		)
		return
	}

	r.client = clients.S3
}

func (r *BucketPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating bucket policy", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.PutBucketPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
		data.Policy.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket policy: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetBucketPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket policy: %s", err))
		return
	}

	if policy == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Policy = types.StringValue(policy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.PutBucketPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
		data.Policy.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket policy: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket policy", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.DeleteBucketPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket policy: %s", err))
		return
	}
}

func (r *BucketPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:BUCKET_NAME",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[2])...)
}
