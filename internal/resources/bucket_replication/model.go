package bucketreplication

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketReplicationResourceModel struct {
	AccountAccessKey types.String           `tfsdk:"account_access_key"`
	AccountSecretKey types.String           `tfsdk:"account_secret_key"`
	Bucket           types.String           `tfsdk:"bucket"`
	Role             types.String           `tfsdk:"role"`
	Rules            []ReplicationRuleModel `tfsdk:"rule"`
}

type ReplicationRuleModel struct {
	ID                      types.String `tfsdk:"id"`
	Status                  types.String `tfsdk:"status"`
	Prefix                  types.String `tfsdk:"prefix"`
	DestinationBucket       types.String `tfsdk:"destination_bucket"`
	DestinationStorageClass types.String `tfsdk:"destination_storage_class"`
}
