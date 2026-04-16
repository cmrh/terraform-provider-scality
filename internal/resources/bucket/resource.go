package bucket

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

var _ resource.Resource = &BucketResource{}
var _ resource.ResourceWithImportState = &BucketResource{}

type BucketResource struct {
	client *client.S3Client
}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an S3 bucket within a Scality account.",

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
			"versioning": schema.BoolAttribute{
				MarkdownDescription: "Enable versioning on the bucket. Set to true to enable, false to suspend. Leave unset to not manage versioning.",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for the bucket. Overwrites all existing tags when set.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket resource.",
		)
		return
	}

	r.client = clients.S3
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Creating S3 bucket", map[string]interface{}{
		"bucket": bucket,
	})

	if err := r.client.CreateBucket(ctx, ak, sk, bucket); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket: %s", err))
		return
	}

	if !data.Versioning.IsNull() {
		status := "Suspended"
		if data.Versioning.ValueBool() {
			status = "Enabled"
		}
		if err := r.client.PutBucketVersioning(ctx, ak, sk, bucket, status); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket versioning: %s", err))
			return
		}
	}

	if !data.Tags.IsNull() {
		tags := make(map[string]string)
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(tags) > 0 {
			if err := r.client.PutBucketTagging(ctx, ak, sk, bucket, tags); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket tags: %s", err))
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	exists, err := r.client.HeadBucket(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket: %s", err))
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	if !data.Versioning.IsNull() {
		status, err := r.client.GetBucketVersioning(ctx, ak, sk, bucket)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket versioning: %s", err))
			return
		}
		switch status {
		case "Enabled":
			data.Versioning = types.BoolValue(true)
		case "Suspended":
			data.Versioning = types.BoolValue(false)
		default:
			data.Versioning = types.BoolNull()
		}
	}

	if !data.Tags.IsNull() {
		tags, err := r.client.GetBucketTagging(ctx, ak, sk, bucket)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket tags: %s", err))
			return
		}
		if tags == nil {
			data.Tags = types.MapNull(types.StringType)
		} else {
			tagValues := make(map[string]types.String, len(tags))
			for k, v := range tags {
				tagValues[k] = types.StringValue(v)
			}
			mapVal, diags := types.MapValueFrom(ctx, types.StringType, tagValues)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Tags = mapVal
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := plan.AccountAccessKey.ValueString()
	sk := plan.AccountSecretKey.ValueString()
	bucket := plan.Bucket.ValueString()

	if !plan.Versioning.Equal(state.Versioning) {
		if plan.Versioning.IsNull() {
			if err := r.client.PutBucketVersioning(ctx, ak, sk, bucket, "Suspended"); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to suspend bucket versioning: %s", err))
				return
			}
		} else {
			status := "Suspended"
			if plan.Versioning.ValueBool() {
				status = "Enabled"
			}
			if err := r.client.PutBucketVersioning(ctx, ak, sk, bucket, status); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket versioning: %s", err))
				return
			}
		}
	}

	if !plan.Tags.Equal(state.Tags) {
		if plan.Tags.IsNull() {
			if err := r.client.DeleteBucketTagging(ctx, ak, sk, bucket); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket tags: %s", err))
				return
			}
		} else {
			tags := make(map[string]string)
			resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			if len(tags) == 0 {
				if err := r.client.DeleteBucketTagging(ctx, ak, sk, bucket); err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket tags: %s", err))
					return
				}
			} else {
				if err := r.client.PutBucketTagging(ctx, ak, sk, bucket, tags); err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket tags: %s", err))
					return
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting S3 bucket", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.DeleteBucket(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket: %s", err))
		return
	}
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
