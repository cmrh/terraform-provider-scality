package bucketlifecycle

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

var _ resource.Resource = &BucketLifecycleResource{}
var _ resource.ResourceWithImportState = &BucketLifecycleResource{}

type BucketLifecycleResource struct {
	client *client.S3Client
}

func NewBucketLifecycleResource() resource.Resource {
	return &BucketLifecycleResource{}
}

func (r *BucketLifecycleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_lifecycle"
}

func (r *BucketLifecycleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
						},
						"status": schema.StringAttribute{
							Required: true,
						},
						"prefix": schema.StringAttribute{
							Optional: true,
						},
						"expiration_days": schema.Int64Attribute{
							Optional: true,
						},
						"expiration_date": schema.StringAttribute{
							Optional: true,
						},
						"noncurrent_version_expiration_days": schema.Int64Attribute{
							Optional: true,
						},
						"abort_incomplete_multipart_upload_days": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *BucketLifecycleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket_lifecycle resource.",
		)
		return
	}

	r.client = clients.S3
}

func modelRulesToClient(rules []LifecycleRuleModel) []client.LifecycleRule {
	clientRules := make([]client.LifecycleRule, 0, len(rules))
	for _, rule := range rules {
		cr := client.LifecycleRule{
			ID:     rule.ID.ValueString(),
			Status: rule.Status.ValueString(),
			Prefix: rule.Prefix.ValueString(),
		}
		if !rule.ExpirationDays.IsNull() {
			cr.ExpirationDays = int(rule.ExpirationDays.ValueInt64())
		}
		if !rule.ExpirationDate.IsNull() {
			cr.ExpirationDate = rule.ExpirationDate.ValueString()
		}
		if !rule.NoncurrentVersionExpirationDays.IsNull() {
			cr.NoncurrentVersionExpirationDays = int(rule.NoncurrentVersionExpirationDays.ValueInt64())
		}
		if !rule.AbortIncompleteMultipartUploadDays.IsNull() {
			cr.AbortIncompleteMultipartUploadDays = int(rule.AbortIncompleteMultipartUploadDays.ValueInt64())
		}
		clientRules = append(clientRules, cr)
	}
	return clientRules
}

func clientRulesToModel(rules []client.LifecycleRule) []LifecycleRuleModel {
	modelRules := make([]LifecycleRuleModel, 0, len(rules))
	for _, rule := range rules {
		mr := LifecycleRuleModel{
			ID:     types.StringValue(rule.ID),
			Status: types.StringValue(rule.Status),
			Prefix: types.StringValue(rule.Prefix),
		}
		if rule.ExpirationDays > 0 {
			mr.ExpirationDays = types.Int64Value(int64(rule.ExpirationDays))
		} else {
			mr.ExpirationDays = types.Int64Null()
		}
		if rule.ExpirationDate != "" {
			mr.ExpirationDate = types.StringValue(rule.ExpirationDate)
		} else {
			mr.ExpirationDate = types.StringNull()
		}
		if rule.NoncurrentVersionExpirationDays > 0 {
			mr.NoncurrentVersionExpirationDays = types.Int64Value(int64(rule.NoncurrentVersionExpirationDays))
		} else {
			mr.NoncurrentVersionExpirationDays = types.Int64Null()
		}
		if rule.AbortIncompleteMultipartUploadDays > 0 {
			mr.AbortIncompleteMultipartUploadDays = types.Int64Value(int64(rule.AbortIncompleteMultipartUploadDays))
		} else {
			mr.AbortIncompleteMultipartUploadDays = types.Int64Null()
		}
		modelRules = append(modelRules, mr)
	}
	return modelRules
}

func (r *BucketLifecycleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketLifecycleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Creating bucket lifecycle configuration", map[string]interface{}{
		"bucket": bucket,
	})

	clientRules := modelRulesToClient(data.Rules)

	if err := r.client.PutBucketLifecycle(ctx, ak, sk, bucket, clientRules); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket lifecycle configuration: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketLifecycleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	rules, err := r.client.GetBucketLifecycle(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket lifecycle configuration: %s", err))
		return
	}

	if rules == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Rules = clientRulesToModel(rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketLifecycleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	clientRules := modelRulesToClient(data.Rules)

	if err := r.client.PutBucketLifecycle(ctx, ak, sk, bucket, clientRules); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket lifecycle configuration: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketLifecycleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketLifecycleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket lifecycle configuration", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.DeleteBucketLifecycle(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket lifecycle configuration: %s", err))
		return
	}
}

func (r *BucketLifecycleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
