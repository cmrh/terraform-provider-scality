package bucketobjectlock

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketObjectLockResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Bucket           types.String `tfsdk:"bucket"`
	RetentionMode    types.String `tfsdk:"retention_mode"`
	RetentionDays    types.Int64  `tfsdk:"retention_days"`
	RetentionYears   types.Int64  `tfsdk:"retention_years"`
}
