package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AccountResource{}
var _ resource.ResourceWithImportState = &AccountResource{}

func NewAccountResource() resource.Resource {
	return &AccountResource{}
}

// AccountResource defines the resource implementation
type AccountResource struct {
	client *ScalityClient
}

// AccountResourceModel describes the resource data model
type AccountResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	EmailAddress      types.String `tfsdk:"email_address"`
	QuotaMax          types.Int64  `tfsdk:"quota_max"`
	ExternalAccountID types.String `tfsdk:"external_account_id"`
	ARN               types.String `tfsdk:"arn"`
	CanonicalID       types.String `tfsdk:"canonical_id"`
	CreateDate        types.String `tfsdk:"create_date"`
	AccessKey         types.String `tfsdk:"access_key"`
	SecretKey         types.String `tfsdk:"secret_key"`
}

func (r *AccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *AccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Scality account with automatically generated S3 API credentials.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Account ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the account",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Email address for the account",
				Required:            true,
			},
			"quota_max": schema.Int64Attribute{
				MarkdownDescription: "Maximum amount of bytes storable by the account (0 = unlimited)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"external_account_id": schema.StringAttribute{
				MarkdownDescription: "External account ID for integration with other systems",
				Optional:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "Amazon Resource Name (ARN) of the account",
				Computed:            true,
			},
			"canonical_id": schema.StringAttribute{
				MarkdownDescription: "Canonical ID of the account",
				Computed:            true,
			},
			"create_date": schema.StringAttribute{
				MarkdownDescription: "Account creation date",
				Computed:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "S3 API access key (generated automatically)",
				Computed:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "S3 API secret key (generated automatically)",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *AccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials (endpoint, access_key, secret_key) must be configured to use scality_account resource.",
		)
		return
	}

	r.client = clients.IAMClient
}

func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Scality account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Create account
	createReq := AccountCreateRequest{
		Name:              data.Name.ValueString(),
		EmailAddress:      data.EmailAddress.ValueString(),
		QuotaMax:          data.QuotaMax.ValueInt64(),
		ExternalAccountID: data.ExternalAccountID.ValueString(),
	}

	account, err := r.client.CreateAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create account: %s", err))
		return
	}

	// Generate access key
	tflog.Debug(ctx, "Generating access key for account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	accessKey, err := r.client.GenerateAccountAccessKey(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to generate access key: %s", err))
		return
	}

	// Update model with response data
	data.ID = types.StringValue(account.Account.Data.ID)
	data.ARN = types.StringValue(account.Account.Data.ARN)
	data.CanonicalID = types.StringValue(account.Account.Data.CanonicalID)
	data.CreateDate = types.StringValue(account.Account.Data.CreateDate)
	data.AccessKey = types.StringValue(accessKey.Data.ID)
	data.SecretKey = types.StringValue(accessKey.Data.Value)

	// Set default quota if not specified
	if data.QuotaMax.IsNull() || data.QuotaMax.IsUnknown() {
		data.QuotaMax = types.Int64Value(account.Account.Data.QuotaMax)
	}

	tflog.Trace(ctx, "Created account resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Scality account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Get account details
	account, err := r.client.GetAccount(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account: %s", err))
		return
	}

	// Account was deleted outside Terraform
	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with refreshed data
	data.ID = types.StringValue(account.Data.ID)
	data.ARN = types.StringValue(account.Data.ARN)
	data.CanonicalID = types.StringValue(account.Data.CanonicalID)
	data.CreateDate = types.StringValue(account.Data.CreateDate)
	data.QuotaMax = types.Int64Value(account.Data.QuotaMax)
	data.EmailAddress = types.StringValue(account.Data.EmailAddress)

	// Keep access key and secret key from state (they can't be retrieved)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Scality account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Note: The Scality API may not support updates.
	// For now, we'll just update the state with the planned values.
	// In a production provider, you would implement UpdateAccount API calls here.

	resp.Diagnostics.AddWarning(
		"Update Not Fully Implemented",
		"Account updates may require replacement. Check the Scality API documentation for update capabilities.",
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Scality account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	err := r.client.DeleteAccount(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete account: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted account resource")
}

func (r *AccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
