package bucketlogging

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketLoggingResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Bucket           types.String `tfsdk:"bucket"`
	TargetBucket     types.String `tfsdk:"target_bucket"`
	TargetPrefix     types.String `tfsdk:"target_prefix"`
}
