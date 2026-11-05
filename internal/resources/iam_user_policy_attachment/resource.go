package iamuserpolicyattachment

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

	"github.com/cmrh/terraform-provider-scality/internal/accountguard"
	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &IAMUserPolicyAttachmentResource{}
var _ resource.ResourceWithImportState = &IAMUserPolicyAttachmentResource{}

type IAMUserPolicyAttachmentResource struct {
	clients *client.ProviderClients
}

func NewIAMUserPolicyAttachmentResource() resource.Resource {
	return &IAMUserPolicyAttachmentResource{}
}

func (r *IAMUserPolicyAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_user_policy_attachment"
}

func (r *IAMUserPolicyAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches an IAM managed policy to an IAM user within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this user and policy. Omit to use the provider's assumed-role credentials (see the provider `assume_role` block).",
				Optional:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this user and policy. Omit to use the provider's assumed-role credentials.",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Name of the IAM user to attach the policy to",
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

func (r *IAMUserPolicyAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials must be configured to use scality_iam_user_policy_attachment resource.",
		)
		return
	}

	r.clients = clients
}

func (r *IAMUserPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IAMUserPolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveIAM(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	resp.Diagnostics.Append(accountguard.Verify(ctx, c, ak, sk, nil, resp.Private)...)

	tflog.Debug(ctx, "Attaching policy to user", map[string]any{
		"username":   data.Username.ValueString(),
		"policy_arn": data.PolicyArn.ValueString(),
	})

	err = c.AttachUserPolicy(ctx, ak, sk, data.Username.ValueString(), data.PolicyArn.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to attach policy to user: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IAMUserPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IAMUserPolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveIAM(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	resp.Diagnostics.Append(accountguard.Verify(ctx, c, ak, sk, req.Private, resp.Private)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := c.ListAttachedUserPolicies(ctx, ak, sk, data.Username.ValueString())
	if err != nil {
		if client.IsNotFound(err) { // parent user deleted out of band
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list attached user policies: %s", err))
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

func (r *IAMUserPolicyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IAMUserPolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only the account credentials update in place; every other attribute forces replacement.
	c, ak, sk, err := r.clients.ResolveIAM(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	resp.Diagnostics.Append(accountguard.Verify(ctx, c, ak, sk, req.Private, resp.Private)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IAMUserPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IAMUserPolicyAttachmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveIAM(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	resp.Diagnostics.Append(accountguard.Verify(ctx, c, ak, sk, req.Private, nil)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Detaching policy from user", map[string]any{
		"username":   data.Username.ValueString(),
		"policy_arn": data.PolicyArn.ValueString(),
	})

	err = c.DetachUserPolicy(ctx, ak, sk, data.Username.ValueString(), data.PolicyArn.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to detach policy from user: %s", err))
		return
	}
}

func (r *IAMUserPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		// POLICY_ARN itself contains colons — split only the leading username off.
		idParts := strings.SplitN(req.ID, ":", 2)
		if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be in format: USERNAME:POLICY_ARN (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), idParts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), idParts[1])...)
		return
	}

	// Provider assumed-role credentials: USERNAME:POLICY_ARN ID (second part
	// starts with "arn:"), credentials resolved from the provider at read time.
	if r.clients != nil && r.clients.Assumed != nil {
		if idParts := strings.SplitN(req.ID, ":", 2); len(idParts) == 2 && idParts[0] != "" && strings.HasPrefix(idParts[1], "arn:") {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), idParts[0])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), idParts[1])...)
			return
		}
	}

	// ARN contains colons, so use SplitN with 4 — everything after the third : is the policy ARN
	parts := strings.SplitN(req.ID, ":", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:USERNAME:POLICY_ARN",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), parts[3])...)
}
