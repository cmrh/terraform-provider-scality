package userpolicy

import "github.com/hashicorp/terraform-plugin-framework/types"

type UserPolicyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Username         types.String `tfsdk:"username"`
	PolicyName       types.String `tfsdk:"policy_name"`
	PolicyDocument   types.String `tfsdk:"policy_document"`
}
