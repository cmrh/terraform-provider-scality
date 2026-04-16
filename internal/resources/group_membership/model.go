package groupmembership

import "github.com/hashicorp/terraform-plugin-framework/types"

type GroupMembershipResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	GroupName        types.String `tfsdk:"group_name"`
	Users            types.Set    `tfsdk:"users"`
}
