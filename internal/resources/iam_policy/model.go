package iampolicy

import "github.com/hashicorp/terraform-plugin-framework/types"

type IAMPolicyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	PolicyName       types.String `tfsdk:"policy_name"`
	PolicyDocument   types.String `tfsdk:"policy_document"`
	Arn              types.String `tfsdk:"arn"`
}
