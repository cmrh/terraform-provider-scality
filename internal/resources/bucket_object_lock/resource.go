package bucketobjectlock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ resource.Resource = &BucketObjectLockResource{}
var _ resource.ResourceWithImportState = &BucketObjectLockResource{}

type BucketObjectLockResource struct {
	client *client.S3Client
}

func NewBucketObjectLockResource() resource.Resource {
	return &BucketObjectLockResource{}
}

func (r *BucketObjectLockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_object_lock"
}

func (r *BucketObjectLockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Object Lock configuration for an S3 bucket.",

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"retention_mode": schema.StringAttribute{
				MarkdownDescription: "Default retention mode for objects placed in the bucket (GOVERNANCE or COMPLIANCE)",
				Required:            true,
			},
			"retention_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days for the default retention period (mutually exclusive with retention_years)",
				Optional:            true,
			},
			"retention_years": schema.Int64Attribute{
				MarkdownDescription: "Number of years for the default retention period (mutually exclusive with retention_days)",
				Optional:            true,
			},
		},
	}
}

func (r *BucketObjectLockResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket_object_lock resource.",
		)
		return
	}

	r.client = clients.S3
}

func (r *BucketObjectLockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketObjectLockResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Setting object lock configuration", map[string]interface{}{
		"bucket": bucket,
	})

	config := client.ObjectLockConfig{
		Enabled:       true,
		RetentionMode: data.RetentionMode.ValueString(),
	}

	if !data.RetentionDays.IsNull() {
		config.RetentionDays = int(data.RetentionDays.ValueInt64())
	}

	if !data.RetentionYears.IsNull() {
		config.RetentionYears = int(data.RetentionYears.ValueInt64())
	}

	if err := r.client.PutObjectLockConfiguration(ctx, ak, sk, bucket, config); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set object lock configuration: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketObjectLockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketObjectLockResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	result, err := r.client.GetObjectLockConfiguration(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read object lock configuration: %s", err))
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.RetentionMode = types.StringValue(result.RetentionMode)

	if result.RetentionDays > 0 {
		data.RetentionDays = types.Int64Value(int64(result.RetentionDays))
	} else {
		data.RetentionDays = types.Int64Null()
	}

	if result.RetentionYears > 0 {
		data.RetentionYears = types.Int64Value(int64(result.RetentionYears))
	} else {
		data.RetentionYears = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketObjectLockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketObjectLockResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	config := client.ObjectLockConfig{
		Enabled:       true,
		RetentionMode: data.RetentionMode.ValueString(),
	}

	if !data.RetentionDays.IsNull() {
		config.RetentionDays = int(data.RetentionDays.ValueInt64())
	}

	if !data.RetentionYears.IsNull() {
		config.RetentionYears = int(data.RetentionYears.ValueInt64())
	}

	if err := r.client.PutObjectLockConfiguration(ctx, ak, sk, bucket, config); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update object lock configuration: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketObjectLockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketObjectLockResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Removing object lock default retention", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.PutObjectLockConfiguration(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
		client.ObjectLockConfig{Enabled: true},
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove object lock default retention: %s", err))
		return
	}
}

func (r *BucketObjectLockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
