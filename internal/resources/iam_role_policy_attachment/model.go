package iamrolepolicyattachment

import "github.com/hashicorp/terraform-plugin-framework/types"

type IAMRolePolicyAttachmentResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	RoleName         types.String `tfsdk:"role_name"`
	PolicyArn        types.String `tfsdk:"policy_arn"`
}
