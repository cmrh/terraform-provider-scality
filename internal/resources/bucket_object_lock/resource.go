package bucketobjectlock

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

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &BucketObjectLockResource{}
var _ resource.ResourceWithImportState = &BucketObjectLockResource{}

type BucketObjectLockResource struct {
	clients *client.ProviderClients
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
				MarkdownDescription: "Access key of the account that owns this bucket. Omit to use the provider's assumed-role credentials (see the provider `assume_role` block).",
				Optional:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this bucket. Omit to use the provider's assumed-role credentials.",
				Optional:            true,
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
			"retention_mode": schema.StringAttribute{
				MarkdownDescription: "Default retention mode for objects placed in the bucket (GOVERNANCE or COMPLIANCE)",
				Required:            true,
				Validators:          validators.OneOf("GOVERNANCE", "COMPLIANCE"),
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

	r.clients = clients
}

func (r *BucketObjectLockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketObjectLockResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
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

	if err := c.PutObjectLockConfiguration(ctx, ak, sk, bucket, config); err != nil {
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

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	bucket := data.Bucket.ValueString()

	result, err := c.GetObjectLockConfiguration(ctx, ak, sk, bucket)
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

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
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

	if err := c.PutObjectLockConfiguration(ctx, ak, sk, bucket, config); err != nil {
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

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}

	tflog.Debug(ctx, "Removing object lock default retention", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err = c.PutObjectLockConfiguration(ctx, ak, sk, data.Bucket.ValueString(), client.ObjectLockConfig{Enabled: true})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove object lock default retention: %s", err))
		return
	}
}

func (r *BucketObjectLockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		if req.ID == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be: BUCKET_NAME (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), req.ID)...)
		return
	}

	// Provider assumed-role credentials: bare BUCKET_NAME ID, credentials resolved
	// from the provider at read time (left null in state).
	if r.clients != nil && r.clients.Assumed != nil && !strings.Contains(req.ID, ":") {
		if req.ID == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be: BUCKET_NAME")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), req.ID)...)
		return
	}

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
