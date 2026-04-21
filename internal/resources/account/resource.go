package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &AccountResource{}
var _ resource.ResourceWithImportState = &AccountResource{}

type AccountResource struct {
	client *client.IAMClient
}

func NewAccountResource() resource.Resource {
	return &AccountResource{}
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
				Validators:          validators.AccountName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Email address for the account",
				Required:            true,
				Validators:          validators.Email(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"quota_max": schema.Int64Attribute{
				MarkdownDescription: "Maximum amount of bytes storable by the account (0 = unlimited)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"external_account_id": schema.StringAttribute{
				MarkdownDescription: "External account ID for integration with other systems",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"custom_attributes": schema.MapAttribute{
				MarkdownDescription: "Custom attributes for the account (key-value string pairs, max 10)",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "Amazon Resource Name (ARN) of the account",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"canonical_id": schema.StringAttribute{
				MarkdownDescription: "Canonical ID of the account",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date": schema.StringAttribute{
				MarkdownDescription: "Account creation date",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "S3 API access key (generated automatically)",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "S3 API secret key (generated automatically)",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clients.IAM == nil || clients.IAM.AccessKey == "" {
		resp.Diagnostics.AddError(
			"Missing IAM Admin Credentials",
			"IAM API admin credentials (endpoint, access_key, secret_key) must all be configured to use scality_account resource.",
		)
		return
	}

	r.client = clients.IAM
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

	createReq := client.AccountCreateRequest{
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

	data.ID = types.StringValue(account.Account.Data.ID)
	data.ARN = types.StringValue(account.Account.Data.ARN)
	data.CanonicalID = types.StringValue(account.Account.Data.CanonicalID)
	data.CreateDate = types.StringValue(account.Account.Data.CreateDate)

	if data.QuotaMax.IsNull() || data.QuotaMax.IsUnknown() {
		data.QuotaMax = types.Int64Value(account.Account.Data.QuotaMax)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Debug(ctx, "Generating access key for account", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	accessKey, err := r.client.GenerateAccountAccessKey(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Account created successfully but access key generation failed. "+
				"The account exists and is tracked in state. Run apply again or use "+
				"scality_account_access_key to generate keys separately: %s", err))
		return
	}

	data.AccessKey = types.StringValue(accessKey.Data.ID)
	data.SecretKey = types.StringValue(accessKey.Data.Value)

	if !data.CustomAttributes.IsNull() && !data.CustomAttributes.IsUnknown() && len(data.CustomAttributes.Elements()) > 0 {
		attrs := make(map[string]string)
		for k, v := range data.CustomAttributes.Elements() {
			attrs[k] = v.(types.String).ValueString()
		}

		if err := r.client.UpdateAccountAttributes(ctx, data.Name.ValueString(), attrs); err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Account created but setting custom attributes failed: %s", err))
			return
		}
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

	account, err := r.client.GetAccount(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account: %s", err))
		return
	}

	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(account.ID)
	data.ARN = types.StringValue(account.ARN)
	data.CanonicalID = types.StringValue(account.CanonicalID)
	data.CreateDate = types.StringValue(account.CreateDate)
	data.QuotaMax = types.Int64Value(account.QuotaMax)
	data.EmailAddress = types.StringValue(account.EmailAddress)

	if len(account.CustomAttributes) > 0 {
		elements := make(map[string]attr.Value)
		for k, v := range account.CustomAttributes {
			elements[k] = types.StringValue(v)
		}
		data.CustomAttributes = types.MapValueMust(types.StringType, elements)
	} else if !data.CustomAttributes.IsNull() {
		data.CustomAttributes = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state AccountResourceModel
	var plan AccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := make(map[string]string)
	if !plan.CustomAttributes.IsNull() && !plan.CustomAttributes.IsUnknown() {
		for k, v := range plan.CustomAttributes.Elements() {
			attrs[k] = v.(types.String).ValueString()
		}
	}

	tflog.Debug(ctx, "Updating account custom attributes", map[string]interface{}{
		"name": state.Name.ValueString(),
	})

	if err := r.client.UpdateAccountAttributes(ctx, state.Name.ValueString(), attrs); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update account attributes: %s", err))
		return
	}

	state.CustomAttributes = plan.CustomAttributes

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
