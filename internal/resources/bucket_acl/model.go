package bucketacl

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketACLResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Bucket           types.String `tfsdk:"bucket"`
	ACL              types.String `tfsdk:"acl"`
}
