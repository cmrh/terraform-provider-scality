package iamgrouppolicyattachment

import "github.com/hashicorp/terraform-plugin-framework/types"

type IAMGroupPolicyAttachmentResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	GroupName        types.String `tfsdk:"group_name"`
	PolicyArn        types.String `tfsdk:"policy_arn"`
}
