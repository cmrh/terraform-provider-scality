package iamrole

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IAMRoleResourceModel struct {
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	RoleName         types.String         `tfsdk:"role_name"`
	AssumeRolePolicy jsontypes.Normalized `tfsdk:"assume_role_policy"`
	Arn              types.String         `tfsdk:"arn"`
}
