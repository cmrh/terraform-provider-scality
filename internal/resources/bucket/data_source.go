package bucket

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ datasource.DataSource = &BucketDataSource{}
var _ datasource.DataSourceWithConfigure = &BucketDataSource{}

type BucketDataSource struct {
	client *client.S3Client
}

func NewBucketDataSource() datasource.DataSource {
	return &BucketDataSource{}
}

type BucketDataSourceModel struct {
	AccountAccessKey  types.String `tfsdk:"account_access_key"`
	AccountSecretKey  types.String `tfsdk:"account_secret_key"`
	Bucket            types.String `tfsdk:"bucket"`
	ID                types.String `tfsdk:"id"`
	ARN               types.String `tfsdk:"arn"`
	Versioning        types.Bool   `tfsdk:"versioning"`
	ObjectLockEnabled types.Bool   `tfsdk:"object_lock_enabled"`
	Tags              types.Map    `tfsdk:"tags"`
}

func (d *BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *BucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing S3 bucket by name within an account. Useful for attaching Terraform-managed sub-resources (`scality_bucket_policy`, `scality_bucket_lifecycle`, `scality_bucket_replication`, etc.) to a bucket that was created outside the current configuration. Sub-feature data sources (encryption, lifecycle, replication configuration of the existing bucket) are not yet implemented — open an issue if needed.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns the bucket.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns the bucket.",
				Required:            true,
				Sensitive:           true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "Name of the bucket to look up.",
				Required:            true,
				Validators:          validators.BucketName(),
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Bucket identifier (same value as `bucket`).",
				Computed:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the bucket (`arn:aws:s3:::<bucket>`).",
				Computed:            true,
			},
			"versioning": schema.BoolAttribute{
				MarkdownDescription: "Whether versioning is enabled on the bucket (`true` when status is `Enabled`, `false` otherwise).",
				Computed:            true,
			},
			"object_lock_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Object Lock is enabled on the bucket.",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags attached to the bucket.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *BucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T.", req.ProviderData),
		)
		return
	}

	if clients.S3 == nil {
		resp.Diagnostics.AddError(
			"Missing S3 Client Configuration",
			"An IAM endpoint must be configured to use the scality_bucket data source.",
		)
		return
	}

	d.client = clients.S3
}

func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Looking up S3 bucket via data source", map[string]any{
		"bucket": bucket,
	})

	exists, err := d.client.HeadBucket(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to look up bucket %q: %s", bucket, err))
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"Bucket Not Found",
			fmt.Sprintf("No bucket named %q exists in this account.", bucket),
		)
		return
	}

	data.ID = types.StringValue(bucket)
	data.ARN = types.StringValue(fmt.Sprintf("arn:aws:s3:::%s", bucket))

	versioningStatus, err := d.client.GetBucketVersioning(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket versioning: %s", err))
		return
	}
	data.Versioning = types.BoolValue(versioningStatus == "Enabled")

	lockConfig, err := d.client.GetObjectLockConfiguration(ctx, ak, sk, bucket)
	if err != nil {
		// Treat read errors here as "not configured" rather than fatal —
		// some buckets simply don't have Object Lock and the API surface
		// varies. The flag is informational.
		data.ObjectLockEnabled = types.BoolValue(false)
	} else {
		data.ObjectLockEnabled = types.BoolValue(lockConfig != nil && lockConfig.Enabled)
	}

	tags, err := d.client.GetBucketTagging(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket tags: %s", err))
		return
	}
	if len(tags) > 0 {
		elements := make(map[string]attr.Value, len(tags))
		for k, v := range tags {
			elements[k] = types.StringValue(v)
		}
		data.Tags = types.MapValueMust(types.StringType, elements)
	} else {
		data.Tags = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
