package groupmembership

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

var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

type GroupMembershipResource struct {
	client *client.IAMClient
}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

func (r *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the membership of IAM users in a group within a Scality account.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this group",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this group",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the IAM group",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"users": schema.SetAttribute{
				MarkdownDescription: "Set of IAM usernames that belong to this group",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"IAM API credentials (endpoint, access_key, secret_key) must be configured to use scality_group_membership resource.",
		)
		return
	}

	r.client = clients.IAM
}

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var users []string
	resp.Diagnostics.Append(data.Users.ElementsAs(ctx, &users, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	groupName := data.GroupName.ValueString()

	tflog.Debug(ctx, "Adding users to group", map[string]interface{}{
		"group_name": groupName,
		"user_count": len(users),
	})

	for _, userName := range users {
		if err := r.client.AddUserToGroup(ctx, ak, sk, groupName, userName); err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to add user %q to group %q: %s", userName, groupName, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	grp, members, err := r.client.GetGroup(ctx,
		data.AccountAccessKey.ValueString(),
		data.AccountSecretKey.ValueString(),
		data.GroupName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group: %s", err))
		return
	}

	if grp == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	userNames := make([]string, len(members))
	for i, m := range members {
		userNames[i] = m.UserName
	}

	usersSet, diags := types.SetValueFrom(ctx, types.StringType, userNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Users = usersSet
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state GroupMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planUsers, stateUsers []string
	resp.Diagnostics.Append(plan.Users.ElementsAs(ctx, &planUsers, false)...)
	resp.Diagnostics.Append(state.Users.ElementsAs(ctx, &stateUsers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := plan.AccountAccessKey.ValueString()
	sk := plan.AccountSecretKey.ValueString()
	groupName := plan.GroupName.ValueString()

	oldSet := make(map[string]bool, len(stateUsers))
	for _, u := range stateUsers {
		oldSet[u] = true
	}

	newSet := make(map[string]bool, len(planUsers))
	for _, u := range planUsers {
		newSet[u] = true
	}

	for _, u := range planUsers {
		if !oldSet[u] {
			if err := r.client.AddUserToGroup(ctx, ak, sk, groupName, u); err != nil {
				resp.Diagnostics.AddError("Client Error",
					fmt.Sprintf("Unable to add user %q to group %q: %s", u, groupName, err))
				return
			}
		}
	}

	for _, u := range stateUsers {
		if !newSet[u] {
			if err := r.client.RemoveUserFromGroup(ctx, ak, sk, groupName, u); err != nil {
				resp.Diagnostics.AddError("Client Error",
					fmt.Sprintf("Unable to remove user %q from group %q: %s", u, groupName, err))
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var users []string
	resp.Diagnostics.Append(data.Users.ElementsAs(ctx, &users, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	groupName := data.GroupName.ValueString()

	tflog.Debug(ctx, "Removing all users from group", map[string]interface{}{
		"group_name": groupName,
		"user_count": len(users),
	})

	for _, userName := range users {
		if err := r.client.RemoveUserFromGroup(ctx, ak, sk, groupName, userName); err != nil {
			// Account gone: nothing left to remove.
			if strings.Contains(err.Error(), "InvalidAccessKeyId") {
				return
			}
			// This user (or the group) is already gone: skip it, keep removing the rest.
			if strings.Contains(err.Error(), "NoSuchEntity") {
				continue
			}
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to remove user %q from group %q: %s", userName, groupName, err))
			return
		}
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		if req.ID == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be: GROUP_NAME (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), req.ID)...)
		return
	}

	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:GROUP_NAME",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), parts[2])...)
}
