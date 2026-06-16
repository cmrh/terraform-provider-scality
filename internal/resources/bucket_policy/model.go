package bucketpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketPolicyResourceModel struct {
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	Bucket           types.String         `tfsdk:"bucket"`
	Policy           jsontypes.Normalized `tfsdk:"policy"`
}
