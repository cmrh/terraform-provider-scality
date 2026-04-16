package bucketencryption

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketEncryptionResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Bucket           types.String `tfsdk:"bucket"`
	SSEAlgorithm     types.String `tfsdk:"sse_algorithm"`
	KMSMasterKeyID   types.String `tfsdk:"kms_master_key_id"`
}
