package bucket

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &BucketResource{}
var _ resource.ResourceWithImportState = &BucketResource{}
var _ resource.ResourceWithValidateConfig = &BucketResource{}

type BucketResource struct {
	clients *client.ProviderClients
}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an S3 bucket within a Scality account.",

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
			"object_lock_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable Object Lock on the bucket. Can only be set at creation time.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIf(
						objectLockRequiresReplace,
						"Changing Object Lock requires replacing the bucket.",
						"Changing Object Lock requires replacing the bucket.",
					),
				},
			},
			"versioning": schema.BoolAttribute{
				MarkdownDescription: "Enable object versioning. `true` enables it; `false` leaves it off (a new bucket stays Unversioned; a bucket that was previously enabled is Suspended, since S3 has no way back to Unversioned). Leave unset to not manage versioning.",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for the bucket. Overwrites all existing tags when set.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// importedPrivateKey marks state written by ImportState until the first Read populates it.
const importedPrivateKey = "imported"

// objectLockRequiresReplace treats null as false, so null <-> false never replaces the bucket.
func objectLockRequiresReplace(ctx context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsUnknown() {
		resp.RequiresReplace = true
		return
	}
	resp.RequiresReplace = req.PlanValue.ValueBool() != req.StateValue.ValueBool()
}

func (r *BucketResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data BucketResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Object Lock forces versioning on, so object_lock_enabled=true with versioning=false is contradictory.
	objectLock := !data.ObjectLockEnabled.IsNull() && !data.ObjectLockEnabled.IsUnknown() && data.ObjectLockEnabled.ValueBool()
	versioningOff := !data.Versioning.IsNull() && !data.Versioning.IsUnknown() && !data.Versioning.ValueBool()
	if objectLock && versioningOff {
		resp.Diagnostics.AddAttributeError(
			path.Root("versioning"),
			"versioning conflicts with Object Lock",
			"object_lock_enabled = true requires versioning to be enabled. Set versioning = true, or remove it — Object Lock enables versioning automatically.",
		)
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use scality_bucket resource.",
		)
		return
	}

	r.clients = clients
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketResourceModel

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

	tflog.Debug(ctx, "Creating S3 bucket", map[string]interface{}{
		"bucket": bucket,
	})

	objectLockEnabled := !data.ObjectLockEnabled.IsNull() && data.ObjectLockEnabled.ValueBool()

	if err := c.CreateBucket(ctx, ak, sk, bucket, objectLockEnabled); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket: %s", err))
		return
	}

	// The bucket now exists. If a later step fails, persist state so the bucket
	// is tracked (Terraform taints it on the Create error) and the next apply
	// replaces it, instead of orphaning it and failing with
	// BucketAlreadyOwnedByYou.

	// Only ever enable explicitly; a new bucket is already Unversioned, and Object Lock enables versioning itself.
	if !objectLockEnabled && !data.Versioning.IsNull() && data.Versioning.ValueBool() {
		if err := c.PutBucketVersioning(ctx, ak, sk, bucket, "Enabled"); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket versioning: %s", err))
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	if !data.Tags.IsNull() {
		tags := make(map[string]string)
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(tags) > 0 {
			if err := c.PutBucketTagging(ctx, ak, sk, bucket, tags); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set bucket tags: %s", err))
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketResourceModel

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

	exists, err := c.HeadBucket(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket: %s", err))
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	imported, diags := req.Private.GetKey(ctx, importedPrivateKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if imported != nil {
		if err := readImported(ctx, c, ak, sk, bucket, &data); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read imported bucket: %s", err))
			return
		}
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, importedPrivateKey, nil)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if !data.Versioning.IsNull() {
		status, err := c.GetBucketVersioning(ctx, ak, sk, bucket)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket versioning: %s", err))
			return
		}
		// Enabled is the only "on" state; Suspended and Unversioned both map to false.
		data.Versioning = types.BoolValue(status == "Enabled")
	}

	if !data.Tags.IsNull() {
		tags, err := c.GetBucketTagging(ctx, ak, sk, bucket)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket tags: %s", err))
			return
		}
		if tags == nil {
			// The server reports "no tags" identically for a configured empty
			// map and for genuinely unset tags; map it to an empty (non-null)
			// map so a configured {} round-trips (apply {} then plan is empty).
			mapVal, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Tags = mapVal
		} else {
			tagValues := make(map[string]types.String, len(tags))
			for k, v := range tags {
				tagValues[k] = types.StringValue(v)
			}
			mapVal, diags := types.MapValueFrom(ctx, types.StringType, tagValues)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Tags = mapVal
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// readImported records only settings that are on server-side; anything off stays
// null, so a config that omits it plans clean.
func readImported(ctx context.Context, c *client.S3Client, ak, sk, bucket string, data *BucketResourceModel) error {
	lock, err := c.GetObjectLockConfiguration(ctx, ak, sk, bucket)
	if err != nil {
		return err
	}
	if lock != nil && lock.Enabled {
		data.ObjectLockEnabled = types.BoolValue(true)
	}

	status, err := c.GetBucketVersioning(ctx, ak, sk, bucket)
	if err != nil {
		return err
	}
	if status == "Enabled" {
		data.Versioning = types.BoolValue(true)
	}

	tags, err := c.GetBucketTagging(ctx, ak, sk, bucket)
	if err != nil {
		return err
	}
	if len(tags) > 0 {
		mapVal, diags := types.MapValueFrom(ctx, types.StringType, tags)
		if diags.HasError() {
			return fmt.Errorf("converting tags: %v", diags.Errors())
		}
		data.Tags = mapVal
	}
	return nil
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(plan.AccountAccessKey.ValueString(), plan.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}
	bucket := plan.Bucket.ValueString()

	// Unset = stop managing versioning; leave the bucket's state untouched.
	if !plan.Versioning.Equal(state.Versioning) && !plan.Versioning.IsNull() {
		current, err := c.GetBucketVersioning(ctx, ak, sk, bucket)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket versioning: %s", err))
			return
		}
		switch {
		case plan.Versioning.ValueBool() && current != "Enabled":
			// Skip when already Enabled: Object Lock buckets reject any versioning change.
			if err := c.PutBucketVersioning(ctx, ak, sk, bucket, "Enabled"); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to enable bucket versioning: %s", err))
				return
			}
		case !plan.Versioning.ValueBool() && current == "Enabled":
			// Explicit false: suspend only a currently-Enabled bucket; never touch an Unversioned one.
			if err := c.PutBucketVersioning(ctx, ak, sk, bucket, "Suspended"); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to suspend bucket versioning: %s", err))
				return
			}
		}
	}

	if !plan.Tags.Equal(state.Tags) {
		if plan.Tags.IsNull() {
			if err := c.DeleteBucketTagging(ctx, ak, sk, bucket); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket tags: %s", err))
				return
			}
		} else {
			tags := make(map[string]string)
			resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			if len(tags) == 0 {
				if err := c.DeleteBucketTagging(ctx, ak, sk, bucket); err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket tags: %s", err))
					return
				}
			} else {
				if err := c.PutBucketTagging(ctx, ak, sk, bucket, tags); err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket tags: %s", err))
					return
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, ak, sk, err := r.clients.ResolveS3(data.AccountAccessKey.ValueString(), data.AccountSecretKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Missing Credentials", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleting S3 bucket", map[string]interface{}{
		"bucket": data.Bucket.ValueString(),
	})

	err = c.DeleteBucket(ctx, ak, sk, data.Bucket.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "InvalidAccessKeyId") || strings.Contains(err.Error(), "NoSuchEntity") {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket: %s", err))
		return
	}
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, importedPrivateKey, []byte("true"))...)

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
