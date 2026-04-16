package userpolicy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ resource.Resource = &UserPolicyResource{}

type UserPolicyResource struct {
	client *client.IAMClient
}

func NewUserPolicyResource() resource.Resource {
	return &UserPolicyResource{}
}

func (r *UserPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_policy"
}

func (r *UserPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an inline IAM policy for a user within a Scality account.",

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
				MarkdownDescription: "Name of the IAM user this policy is attached to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_name": schema.StringAttribute{
				MarkdownDescription: "Name of the inline policy",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_document": schema.StringAttribute{
				MarkdownDescription: "JSON policy document",
				Required:            true,
			},
		},
	}
}

func (r *UserPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials (endpoint, access_key, secret_key) must be configured to use scality_user_policy resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *UserPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating user policy", map[string]interface{}{
		"username":    data.Username.ValueString(),
		"policy_name": data.PolicyName.ValueString(),
	})

	err := r.client.PutUserPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
		data.PolicyName.ValueString(),
		data.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create user policy: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyDoc, err := r.client.GetUserPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
		data.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user policy: %s", err))
		return
	}

	if policyDoc == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating user policy", map[string]interface{}{
		"username":    data.Username.ValueString(),
		"policy_name": data.PolicyName.ValueString(),
	})

	err := r.client.PutUserPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
		data.PolicyName.ValueString(),
		data.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user policy: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting user policy", map[string]interface{}{
		"username":    data.Username.ValueString(),
		"policy_name": data.PolicyName.ValueString(),
	})

	err := r.client.DeleteUserPolicy(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.Username.ValueString(),
		data.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete user policy: %s", err))
		return
	}
}
