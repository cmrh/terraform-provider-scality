package iamrole

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &IAMRoleResource{}

type IAMRoleResource struct {
	client *client.IAMClient
}

func NewIAMRoleResource() resource.Resource {
	return &IAMRoleResource{}
}

func (r *IAMRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (r *IAMRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an IAM role within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this role",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this role",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "Name of the IAM role",
				Required:            true,
				Validators:          validators.IAMName(64),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"assume_role_policy": schema.StringAttribute{
				MarkdownDescription: "JSON trust policy document that grants entities permission to assume the role",
				Required:            true,
				Validators:          validators.JSONDocument(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the IAM role",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *IAMRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials must be configured to use scality_iam_role resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *IAMRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IAMRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Creating IAM role", map[string]any{
		"role_name": data.RoleName.ValueString(),
	})

	role, err := r.client.CreateRole(ctx, ak, sk,
		data.RoleName.ValueString(),
		data.AssumeRolePolicy.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create IAM role: %s", err))
		return
	}

	data.Arn = types.StringValue(role.Arn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IAMRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IAMRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	role, err := r.client.GetRole(ctx, ak, sk, data.RoleName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IAM role: %s", err))
		return
	}

	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Arn = types.StringValue(role.Arn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IAMRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All attributes on scality_iam_role require replacement. This should not be called.",
	)
}

func (r *IAMRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IAMRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Deleting IAM role", map[string]any{
		"role_name": data.RoleName.ValueString(),
	})

	err := r.client.DeleteRole(ctx, ak, sk, data.RoleName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete IAM role: %s", err))
		return
	}
}
