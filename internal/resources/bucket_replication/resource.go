package bucketreplication

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

var _ resource.Resource = &BucketReplicationResource{}
var _ resource.ResourceWithImportState = &BucketReplicationResource{}

type BucketReplicationResource struct {
	client *client.S3Client
}

func NewBucketReplicationResource() resource.Resource {
	return &BucketReplicationResource{}
}

func (r *BucketReplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_replication"
}

func (r *BucketReplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"role": schema.StringAttribute{
				Required: true,
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"status": schema.StringAttribute{
							Required: true,
						},
						"prefix": schema.StringAttribute{
							Required: true,
						},
						"destination_bucket": schema.StringAttribute{
							Required: true,
						},
						"destination_storage_class": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *BucketReplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket_replication resource.",
		)
		return
	}

	r.client = clients.S3
}

func modelRulesToClient(rules []ReplicationRuleModel) []client.ReplicationRule {
	clientRules := make([]client.ReplicationRule, 0, len(rules))
	for _, r := range rules {
		cr := client.ReplicationRule{
			ID:                r.ID.ValueString(),
			Status:            r.Status.ValueString(),
			Prefix:            r.Prefix.ValueString(),
			DestinationBucket: r.DestinationBucket.ValueString(),
		}
		if !r.DestinationStorageClass.IsNull() {
			cr.DestinationStorageClass = r.DestinationStorageClass.ValueString()
		}
		clientRules = append(clientRules, cr)
	}
	return clientRules
}

func clientRulesToModel(rules []client.ReplicationRule) []ReplicationRuleModel {
	modelRules := make([]ReplicationRuleModel, 0, len(rules))
	for _, r := range rules {
		mr := ReplicationRuleModel{
			Status:            types.StringValue(r.Status),
			Prefix:            types.StringValue(r.Prefix),
			DestinationBucket: types.StringValue(r.DestinationBucket),
		}
		if r.ID == "" {
			mr.ID = types.StringNull()
		} else {
			mr.ID = types.StringValue(r.ID)
		}
		if r.DestinationStorageClass == "" {
			mr.DestinationStorageClass = types.StringNull()
		} else {
			mr.DestinationStorageClass = types.StringValue(r.DestinationStorageClass)
		}
		modelRules = append(modelRules, mr)
	}
	return modelRules
}

func (r *BucketReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketReplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Creating bucket replication configuration", map[string]interface{}{
		"bucket": bucket,
	})

	clientRules := modelRulesToClient(data.Rules)

	if err := r.client.PutBucketReplication(ctx, ak, sk, bucket, data.Role.ValueString(), clientRules); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket replication: %s", err))
		return
	}

	_, rules, err := r.client.GetBucketReplication(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read back bucket replication after create: %s", err))
		return
	}
	data.Rules = clientRulesToModel(rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketReplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	role, rules, err := r.client.GetBucketReplication(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket replication: %s", err))
		return
	}

	if role == "" && rules == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Role = types.StringValue(role)
	data.Rules = clientRulesToModel(rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketReplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	bucket := data.Bucket.ValueString()

	clientRules := modelRulesToClient(data.Rules)

	if err := r.client.PutBucketReplication(ctx, ak, sk, bucket, data.Role.ValueString(), clientRules); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket replication: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketReplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket replication configuration", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err := r.client.DeleteBucketReplication(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Bucket.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket replication: %s", err))
		return
	}
}

func (r *BucketReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("bucket"), req, resp)
}
