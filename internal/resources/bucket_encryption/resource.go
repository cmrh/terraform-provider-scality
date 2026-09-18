package bucketencryption

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &BucketEncryptionResource{}
var _ resource.ResourceWithImportState = &BucketEncryptionResource{}

type BucketEncryptionResource struct {
	clients *client.ProviderClients
}

func NewBucketEncryptionResource() resource.Resource {
	return &BucketEncryptionResource{}
}

func (r *BucketEncryptionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_encryption"
}

func (r *BucketEncryptionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages server-side encryption configuration for an S3 bucket.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns this bucket. Omit to use the provider's assumed-role credentials (see the provider `assume_role` block).",
				Optional:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns this bucket. Omit to use the provider's assumed-role credentials.",
				Optional:            true,
				Sensitive:           true,
			},
			"bucket": schema.StringAttribute{
				MarkdownDescription: "Name of the S3 bucket",
				Required:            true,
				Validators:          validators.BucketName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sse_algorithm": schema.StringAttribute{
				MarkdownDescription: "Server-side encryption algorithm to use (AES256 or aws:kms)",
				Required:            true,
				Validators:          validators.OneOf("AES256", "aws:kms"),
			},
			"kms_master_key_id": schema.StringAttribute{
				MarkdownDescription: "KMS master key ID to use for encryption (only for aws:kms)",
				Optional:            true,
			},
		},
	}
}

func (r *BucketEncryptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T.", req.ProviderData),
		)
		return
	}

	if clients.S3 == nil {
		resp.Diagnostics.AddError(
			"Missing S3 Client Configuration",
			"An IAM endpoint must be configured to use scality_bucket_encryption resource.",
		)
		return
	}

	r.clients = clients
}

func (r *BucketEncryptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketEncryptionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	bucket := data.Bucket.ValueString()

	tflog.Debug(ctx, "Setting bucket encryption", map[string]interface{}{
		"bucket": bucket,
	})

	err = c.PutBucketEncryption(ctx, ak, sk, bucket, client.EncryptionConfig{
		SSEAlgorithm:   data.SSEAlgorithm.ValueString(),
		KMSMasterKeyID: data.KMSMasterKeyID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket encryption: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketEncryptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketEncryptionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	bucket := data.Bucket.ValueString()

	config, err := c.GetBucketEncryption(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket encryption: %s", err))
		return
	}

	if config == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.SSEAlgorithm = types.StringValue(config.SSEAlgorithm)
	if config.KMSMasterKeyID != "" {
		data.KMSMasterKeyID = types.StringValue(config.KMSMasterKeyID)
	} else {
		data.KMSMasterKeyID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketEncryptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketEncryptionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	bucket := data.Bucket.ValueString()

	err = c.PutBucketEncryption(ctx, ak, sk, bucket, client.EncryptionConfig{
		SSEAlgorithm:   data.SSEAlgorithm.ValueString(),
		KMSMasterKeyID: data.KMSMasterKeyID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket encryption: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketEncryptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketEncryptionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleting bucket encryption", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err = c.DeleteBucketEncryption(ctx, ak, sk, data.Bucket.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket encryption: %s", err))
		return
	}
}

func (r *BucketEncryptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if ak, sk, ok := client.ImportAccountCreds(); ok {
		if req.ID == "" {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be: BUCKET_NAME (account credentials are taken from SCALITY_ACCOUNT_ACCESS_KEY / SCALITY_ACCOUNT_SECRET_KEY)",
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), ak)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), sk)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), req.ID)...)
		return
	}

	// Provider assumed-role credentials: bare BUCKET_NAME ID, credentials resolved
	// from the provider at read time (left null in state).
	if r.clients != nil && r.clients.Assumed != nil && !strings.Contains(req.ID, ":") {
		if req.ID == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be: BUCKET_NAME")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), req.ID)...)
		return
	}

	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: ACCESS_KEY:SECRET_KEY:BUCKET_NAME",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_access_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_secret_key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[2])...)
}
