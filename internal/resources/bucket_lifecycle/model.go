package bucketlifecycle

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketLifecycleResourceModel struct {
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	Bucket           types.String         `tfsdk:"bucket"`
	Rules            []LifecycleRuleModel `tfsdk:"rule"`
}

type LifecycleRuleModel struct {
	ID                                 types.String `tfsdk:"id"`
	Status                             types.String `tfsdk:"status"`
	Prefix                             types.String `tfsdk:"prefix"`
	ExpirationDays                     types.Int64  `tfsdk:"expiration_days"`
	ExpirationDate                     types.String `tfsdk:"expiration_date"`
	NoncurrentVersionExpirationDays    types.Int64  `tfsdk:"noncurrent_version_expiration_days"`
	AbortIncompleteMultipartUploadDays types.Int64  `tfsdk:"abort_incomplete_multipart_upload_days"`
}
