package useraccesskey

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
)

var _ resource.Resource = &UserAccessKeyResource{}
var _ resource.ResourceWithImportState = &UserAccessKeyResource{}

type UserAccessKeyResource struct {
	client *client.IAMClient
}

func NewUserAccessKeyResource() resource.Resource {
	return &UserAccessKeyResource{}
}

func (r *UserAccessKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access_key"
}

func (r *UserAccessKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an access key for an IAM user within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this user",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this user",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Name of the IAM user this key belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key_id": schema.StringAttribute{
				MarkdownDescription: "The access key ID",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				MarkdownDescription: "The secret access key (only available after creation)",
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
		},
	}
}

func (r *UserAccessKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if clients.IAM == nil {
		resp.Diagnostics.AddError(
			"Missing IAM Client Configuration",
			"IAM API credentials (endpoint, access_key, secret_key) must be configured to use scality_user_access_key resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *UserAccessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserAccessKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating access key for IAM user", map[string]interface{}{
		"username": data.Username.ValueString(),
	})

	accessKey, err := r.client.CreateUserAccessKey(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access key: %s", err))
		return
	}

	data.AccessKeyID = types.StringValue(accessKey.AccessKeyId)
	data.SecretAccessKey = types.StringValue(accessKey.SecretAccessKey)
	data.Status = types.StringValue(accessKey.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserAccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserAccessKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := r.client.ListUserAccessKeys(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list access keys: %s", err))
		return
	}

	found := false
	for _, key := range keys {
		if key.AccessKeyId == data.AccessKeyID.ValueString() {
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

func (r *UserAccessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All attributes require resource replacement. This is a provider bug if you see this error.",
	)
}

func (r *UserAccessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserAccessKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting access key for IAM user", map[string]interface{}{
		"username":      data.Username.ValueString(),
		"access_key_id": data.AccessKeyID.ValueString(),
	})

	err := r.client.DeleteUserAccessKey(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
		data.AccessKeyID.ValueString(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			tflog.Warn(ctx, "Access key already removed, skipping delete", map[string]interface{}{
				"username":      data.Username.ValueString(),
				"access_key_id": data.AccessKeyID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access key: %s", err))
		return
	}
}

func (r *UserAccessKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		idParts := strings.SplitN(req.ID, ":", 2)
		if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be in format: USERNAME:ACCESS_KEY_ID (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_key_id"), idParts[1])...)
		appendUserAccessKeyImportSecretWarning(resp)
		return
	}

	parts := strings.SplitN(req.ID, ":", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:USERNAME:ACCESS_KEY_ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_key_id"), parts[3])...)
	appendUserAccessKeyImportSecretWarning(resp)
}

// appendUserAccessKeyImportSecretWarning surfaces a non-blocking warning so an
// operator sees that the imported state has no `secret_access_key` and that
// rotating the key is the supported way to bring an existing one under
// Terraform management. The API only returns the secret at creation time, so
// import-then-Read can never repopulate it.
func appendUserAccessKeyImportSecretWarning(resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddWarning(
		"Imported access key has no secret in state",
		"`secret_access_key` is available only at key creation and the IAM API does not return it on subsequent reads. "+
			"After this import, `secret_access_key` will be empty in state — any downstream resource that consumes it will fail.\n\n"+
			"To adopt an existing manually-created access key under Terraform management, create a new managed `scality_user_access_key` "+
			"and retire the manually-created one. A user supports up to 4 access keys; delete an unused one first if already at the cap.",
	)
}
