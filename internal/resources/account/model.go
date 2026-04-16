package account

import "github.com/hashicorp/terraform-plugin-framework/types"

type AccountResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	EmailAddress      types.String `tfsdk:"email_address"`
	QuotaMax          types.Int64  `tfsdk:"quota_max"`
	ExternalAccountID types.String `tfsdk:"external_account_id"`
	CustomAttributes  types.Map    `tfsdk:"custom_attributes"`
	ARN               types.String `tfsdk:"arn"`
	CanonicalID       types.String `tfsdk:"canonical_id"`
	CreateDate        types.String `tfsdk:"create_date"`
	AccessKey         types.String `tfsdk:"access_key"`
	SecretKey         types.String `tfsdk:"secret_key"`
}
