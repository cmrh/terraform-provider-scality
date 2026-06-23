package iamrolepolicyattachment

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &IAMRolePolicyAttachmentResource{}
var _ resource.ResourceWithImportState = &IAMRolePolicyAttachmentResource{}

type IAMRolePolicyAttachmentResource struct {
	client *client.IAMClient
}

func NewIAMRolePolicyAttachmentResource() resource.Resource {
	return &IAMRolePolicyAttachmentResource{}
}

func (r *IAMRolePolicyAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role_policy_attachment"
}

func (r *IAMRolePolicyAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches an IAM managed policy to an IAM role within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this role and policy",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this role and policy",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "Name of the IAM role to attach the policy to",
				Required:            true,
				Validators:          validators.IAMName(64),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the IAM managed policy to attach",
				Required:            true,
				Validators:          validators.PolicyARN(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *IAMRolePolicyAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials must be configured to use scality_iam_role_policy_attachment resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *IAMRolePolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IAMRolePolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Attaching policy to role", map[string]any{
		"role_name":  data.RoleName.ValueString(),
		"policy_arn": data.PolicyArn.ValueString(),
	})

	err := r.client.AttachRolePolicy(ctx, ak, sk,
		data.RoleName.ValueString(),
		data.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to attach policy to role: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IAMRolePolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IAMRolePolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	policies, err := r.client.ListAttachedRolePolicies(ctx, ak, sk, data.RoleName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list attached role policies: %s", err))
		return
	}

	targetArn := data.PolicyArn.ValueString()
	found := false
	for _, p := range policies {
		if p.PolicyArn == targetArn {
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

func (r *IAMRolePolicyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All attributes on scality_iam_role_policy_attachment require replacement. This should not be called.",
	)
}

func (r *IAMRolePolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IAMRolePolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Detaching policy from role", map[string]any{
		"role_name":  data.RoleName.ValueString(),
		"policy_arn": data.PolicyArn.ValueString(),
	})

	err := r.client.DetachRolePolicy(ctx, ak, sk,
		data.RoleName.ValueString(),
		data.PolicyArn.ValueString(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to detach policy from role: %s", err))
		return
	}
}

func (r *IAMRolePolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		// POLICY_ARN itself contains colons — split only the leading role name off.
		idParts := strings.SplitN(req.ID, ":", 2)
		if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be in format: ROLE_NAME:POLICY_ARN (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), idParts[1])...)
		return
	}

	// ARN contains colons, so use SplitN with 4 — everything after the third : is the policy ARN
	parts := strings.SplitN(req.ID, ":", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:ROLE_NAME:POLICY_ARN",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), parts[3])...)
}
