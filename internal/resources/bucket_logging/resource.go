package bucketlogging

import (
	"context"
	"errors"
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

var _ resource.Resource = &BucketLoggingResource{}
var _ resource.ResourceWithImportState = &BucketLoggingResource{}

type BucketLoggingResource struct {
	clients *client.ProviderClients
}

func NewBucketLoggingResource() resource.Resource {
	return &BucketLoggingResource{}
}

func (r *BucketLoggingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_logging"
}

func (r *BucketLoggingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages server access logging for an S3 bucket. Requires server access logging to be enabled at the cluster level.",

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
				MarkdownDescription: "Name of the source S3 bucket to log requests for.",
				Required:            true,
				Validators:          validators.BucketName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_bucket": schema.StringAttribute{
				MarkdownDescription: "Name of the bucket where log objects are delivered. Must be owned by the same account as the source bucket, and must grant the internal logging service user `s3:PutObject` for delivery to succeed (see the resource docs).",
				Required:            true,
				Validators:          validators.BucketName(),
			},
			"target_prefix": schema.StringAttribute{
				MarkdownDescription: "Key prefix for log objects in the target bucket. May be an empty string.",
				Required:            true,
			},
		},
	}
}

func (r *BucketLoggingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket_logging resource.",
		)
		return
	}

	r.clients = clients
}

func (r *BucketLoggingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketLoggingResourceModel

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

	tflog.Debug(ctx, "Enabling bucket logging", map[string]interface{}{
		"bucket": bucket,
	})

	err = c.PutBucketLogging(ctx, ak, sk, bucket, client.LoggingConfig{
		TargetBucket: data.TargetBucket.ValueString(),
		TargetPrefix: data.TargetPrefix.ValueString(),
	})
	if err != nil {
		if errors.Is(err, client.ErrServerAccessLoggingDisabled) {
			resp.Diagnostics.AddError(loggingDisabledSummary, loggingDisabledDetail(bucket))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable bucket logging: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLoggingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketLoggingResourceModel

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

	config, err := c.GetBucketLogging(ctx, ak, sk, bucket)
	if err != nil {
		// Feature disabled cluster-side: surface it, do not drop the resource
		// from state — re-enabling the feature must let it reconcile again.
		if errors.Is(err, client.ErrServerAccessLoggingDisabled) {
			resp.Diagnostics.AddError(loggingDisabledSummary, loggingDisabledDetail(bucket))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket logging: %s", err))
		return
	}

	// Feature enabled but logging no longer configured: it was disabled out of
	// band — treat as deleted so the next apply re-creates it.
	if config == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.TargetBucket = types.StringValue(config.TargetBucket)
	data.TargetPrefix = types.StringValue(config.TargetPrefix)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLoggingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketLoggingResourceModel

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

	err = c.PutBucketLogging(ctx, ak, sk, bucket, client.LoggingConfig{
		TargetBucket: data.TargetBucket.ValueString(),
		TargetPrefix: data.TargetPrefix.ValueString(),
	})
	if err != nil {
		if errors.Is(err, client.ErrServerAccessLoggingDisabled) {
			resp.Diagnostics.AddError(loggingDisabledSummary, loggingDisabledDetail(bucket))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket logging: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLoggingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketLoggingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}

	tflog.Debug(ctx, "Disabling bucket logging", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err = c.DeleteBucketLogging(ctx, ak, sk, data.Bucket.ValueString())
	if err != nil {
		// Feature already disabled cluster-side: nothing to disable, let destroy
		// succeed rather than trapping the resource in state.
		if errors.Is(err, client.ErrServerAccessLoggingDisabled) {
			return
		}
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable bucket logging: %s", err))
		return
	}
}

func (r *BucketLoggingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

const loggingDisabledSummary = "Server access logging is not enabled on this cluster"

func loggingDisabledDetail(bucket string) string {
	return fmt.Sprintf(
		"The S3 API returned 501 NotImplemented for bucket %q. Server access logging must be "+
			"enabled at the cluster level before scality_bucket_logging can be used. Ask your Scality "+
			"administrator to enable it and try again, or remove this resource from your configuration.",
		bucket,
	)
}
