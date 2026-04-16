package consoleaccount

import "github.com/hashicorp/terraform-plugin-framework/types"

type ConsoleAccountResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	AccountName            types.String `tfsdk:"account_name"`
	Email                  types.String `tfsdk:"email"`
	Quota                  types.Int64  `tfsdk:"quota"`
	GenerateRandomPassword types.Bool   `tfsdk:"generate_random_password"`
	PasswordLength         types.Int64  `tfsdk:"password_length"`
	Password               types.String `tfsdk:"password"`
	CreatedAt              types.String `tfsdk:"created_at"`
	AccessKey              types.String `tfsdk:"access_key"`
	SecretKey              types.String `tfsdk:"secret_key"`
}
