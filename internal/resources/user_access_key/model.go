package useraccesskey

import "github.com/hashicorp/terraform-plugin-framework/types"

type UserAccessKeyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Username         types.String `tfsdk:"username"`
	AccessKeyID      types.String `tfsdk:"access_key_id"`
	SecretAccessKey  types.String `tfsdk:"secret_access_key"`
	Status           types.String `tfsdk:"status"`
}
