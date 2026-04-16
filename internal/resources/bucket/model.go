package bucket

import "github.com/hashicorp/terraform-plugin-framework/types"

type BucketResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Bucket           types.String `tfsdk:"bucket"`
	Versioning       types.Bool   `tfsdk:"versioning"`
	Tags             types.Map    `tfsdk:"tags"`
}
