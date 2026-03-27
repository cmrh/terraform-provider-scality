package provider

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
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AccountAccessKeyResource{}
var _ resource.ResourceWithImportState = &AccountAccessKeyResource{}

func NewAccountAccessKeyResource() resource.Resource {
	return &AccountAccessKeyResource{}
}

// AccountAccessKeyResource defines the resource implementation
type AccountAccessKeyResource struct {
	client *ScalityClient
}

// AccountAccessKeyResourceModel describes the resource data model
type AccountAccessKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	AccountName types.String `tfsdk:"account_name"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
	Status      types.String `tfsdk:"status"`
	CreateDate  types.String `tfsdk:"create_date"`
}

func (r *AccountAccessKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_access_key"
}

func (r *AccountAccessKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an access key for a Scality account. Useful for key rotation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access key ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_name": schema.StringAttribute{
				MarkdownDescription: "Name of the account this key belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "S3 API access key",
				Computed:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "S3 API secret key",
				Computed:            true,
				Sensitive:           true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the key (Active)",
				Computed:            true,
			},
			"create_date": schema.StringAttribute{
				MarkdownDescription: "Key creation date",
				Computed:            true,
			},
		},
	}
}

func (r *AccountAccessKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clients.IAMClient == nil {
		resp.Diagnostics.AddError(
			"Missing IAM Client Configuration",
			"IAM API credentials (endpoint, access_key, secret_key) must be configured to use scality_account_access_key resource.",
		)
		return
	}

	r.client = clients.IAMClient
}

func (r *AccountAccessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Generating access key", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	accessKey, err := r.client.GenerateAccountAccessKey(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to generate access key: %s", err))
		return
	}

	// Update model with response data
	data.ID = types.StringValue(accessKey.Data.ID)
	data.AccessKey = types.StringValue(accessKey.Data.ID)
	data.SecretKey = types.StringValue(accessKey.Data.Value)
	data.Status = types.StringValue(accessKey.Data.Status)
	data.CreateDate = types.StringValue(accessKey.Data.CreateDate)

	tflog.Trace(ctx, "Created access key resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountAccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading access key", map[string]interface{}{
		"access_key_id": data.ID.ValueString(),
	})

	// Note: The API doesn't provide a way to retrieve the secret key after creation
	// We can only check if the key still exists by listing keys
	// For now, we'll just preserve the state

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountAccessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Access keys cannot be updated, only created and deleted
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"Access keys cannot be updated. To rotate a key, delete this resource and create a new one.",
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountAccessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting access key", map[string]interface{}{
		"access_key_id": data.ID.ValueString(),
	})

	err := r.client.DeleteAccessKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access key: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted access key resource")
}

func (r *AccountAccessKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
