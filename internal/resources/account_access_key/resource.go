package accountaccesskey

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

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ resource.Resource = &AccountAccessKeyResource{}
var _ resource.ResourceWithImportState = &AccountAccessKeyResource{}

type AccountAccessKeyResource struct {
	client *client.IAMClient
}

func NewAccountAccessKeyResource() resource.Resource {
	return &AccountAccessKeyResource{}
}

func (r *AccountAccessKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_access_key"
}

func (r *AccountAccessKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an additional S3 API access key for a Scality account. " +
			"Uses per-account credentials to create and manage root access keys.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access key ID (same as access_key)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account to create the key for",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account to create the key for",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "S3 API access key (only available after creation)",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "S3 API secret key (only available after creation, cannot be retrieved later)",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the access key",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date": schema.StringAttribute{
				MarkdownDescription: "Key creation timestamp",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AccountAccessKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if clients.IAM == nil {
		resp.Diagnostics.AddError(
			"Missing IAM Configuration",
			"IAM endpoint must be configured to use scality_account_access_key resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *AccountAccessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Generating root access key for account")

	accessKey, err := r.client.CreateRootAccessKey(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to generate access key: %s", err))
		return
	}

	data.ID = types.StringValue(accessKey.AccessKeyId)
	data.AccessKey = types.StringValue(accessKey.AccessKeyId)
	data.SecretKey = types.StringValue(accessKey.SecretAccessKey)
	data.Status = types.StringValue(accessKey.Status)
	data.CreateDate = types.StringValue(accessKey.CreateDate)

	tflog.Trace(ctx, "Created access key resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountAccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountAccessKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := r.client.ListRootAccessKeys(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list access keys: %s", err))
		return
	}

	found := false
	for _, key := range keys {
		if key.AccessKeyId == data.ID.ValueString() {
			data.Status = types.StringValue(key.Status)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountAccessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Access keys are immutable. All changes require resource replacement.",
	)
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

	err := r.client.DeleteRootAccessKey(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access key: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted access key resource")
}

func (r *AccountAccessKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: access_key_id/secret_key_id (the account credentials used to manage this key)",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
}
