package accountaccesskey

import "github.com/hashicorp/terraform-plugin-framework/types"

type AccountAccessKeyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	AccessKey        types.String `tfsdk:"access_key"`
	SecretKey        types.String `tfsdk:"secret_key"`
	Status           types.String `tfsdk:"status"`
	CreateDate       types.String `tfsdk:"create_date"`
}
