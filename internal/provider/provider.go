package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scality/terraform-provider-scality/internal/client"
	"github.com/scality/terraform-provider-scality/internal/resources/account"
	accountaccesskey "github.com/scality/terraform-provider-scality/internal/resources/account_access_key"
	"github.com/scality/terraform-provider-scality/internal/resources/bucket"
	bucketencryption "github.com/scality/terraform-provider-scality/internal/resources/bucket_encryption"
	bucketlifecycle "github.com/scality/terraform-provider-scality/internal/resources/bucket_lifecycle"
	bucketobjectlock "github.com/scality/terraform-provider-scality/internal/resources/bucket_object_lock"
	bucketpolicy "github.com/scality/terraform-provider-scality/internal/resources/bucket_policy"
	bucketreplication "github.com/scality/terraform-provider-scality/internal/resources/bucket_replication"
	consoleaccount "github.com/scality/terraform-provider-scality/internal/resources/console_account"
	"github.com/scality/terraform-provider-scality/internal/resources/group"
	groupmembership "github.com/scality/terraform-provider-scality/internal/resources/group_membership"
	iampolicy "github.com/scality/terraform-provider-scality/internal/resources/iam_policy"
	iamrole "github.com/scality/terraform-provider-scality/internal/resources/iam_role"
	iamrolepolicyattachment "github.com/scality/terraform-provider-scality/internal/resources/iam_role_policy_attachment"
	"github.com/scality/terraform-provider-scality/internal/resources/user"
	useraccesskey "github.com/scality/terraform-provider-scality/internal/resources/user_access_key"
	userpolicy "github.com/scality/terraform-provider-scality/internal/resources/user_policy"
)

var _ provider.Provider = &ScalityProvider{}

type ScalityProvider struct {
	version string
}

type ScalityProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	AccessKey          types.String `tfsdk:"access_key"`
	SecretKey          types.String `tfsdk:"secret_key"`
	Region             types.String `tfsdk:"region"`
	ConsoleEndpoint    types.String `tfsdk:"console_endpoint"`
	ConsoleUsername    types.String `tfsdk:"console_username"`
	ConsolePassword    types.String `tfsdk:"console_password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func (p *ScalityProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "scality"
	resp.Version = p.version
}

func (p *ScalityProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Scality accounts and resources. Supports both IAM-style API (AWS Signature V4) and Console API (JWT authentication).",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Scality IAM API endpoint (e.g., https://vault.example.com). Can also be set via SCALITY_ENDPOINT environment variable.",
				Optional:    true,
			},
			"access_key": schema.StringAttribute{
				Description: "Admin access key for IAM API authentication. Can also be set via SCALITY_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "Admin secret key for IAM API authentication. Can also be set via SCALITY_SECRET_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Description: "AWS region used for SigV4 request signing. Defaults to \"us-east-1\". Can also be set via SCALITY_REGION environment variable.",
				Optional:    true,
			},
			"console_endpoint": schema.StringAttribute{
				Description: "Scality Console API endpoint (e.g., https://vault.example.com:8080). Can also be set via SCALITY_CONSOLE_ENDPOINT environment variable.",
				Optional:    true,
			},
			"console_username": schema.StringAttribute{
				Description: "Console API username for JWT authentication. Can also be set via SCALITY_CONSOLE_USERNAME environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"console_password": schema.StringAttribute{
				Description: "Console API password for JWT authentication. Can also be set via SCALITY_CONSOLE_PASSWORD environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification (useful for self-signed certificates). Can also be set via SCALITY_INSECURE_SKIP_VERIFY environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *ScalityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ScalityProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("SCALITY_ENDPOINT")
	}

	accessKey := config.AccessKey.ValueString()
	if accessKey == "" {
		accessKey = os.Getenv("SCALITY_ACCESS_KEY")
	}

	secretKey := config.SecretKey.ValueString()
	if secretKey == "" {
		secretKey = os.Getenv("SCALITY_SECRET_KEY")
	}

	region := config.Region.ValueString()
	if region == "" {
		region = os.Getenv("SCALITY_REGION")
	}

	consoleEndpoint := config.ConsoleEndpoint.ValueString()
	if consoleEndpoint == "" {
		consoleEndpoint = os.Getenv("SCALITY_CONSOLE_ENDPOINT")
	}

	consoleUsername := config.ConsoleUsername.ValueString()
	if consoleUsername == "" {
		consoleUsername = os.Getenv("SCALITY_CONSOLE_USERNAME")
	}

	consolePassword := config.ConsolePassword.ValueString()
	if consolePassword == "" {
		consolePassword = os.Getenv("SCALITY_CONSOLE_PASSWORD")
	}

	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	} else if envVal := os.Getenv("SCALITY_INSECURE_SKIP_VERIFY"); envVal != "" {
		insecureSkipVerify = envVal == "true" || envVal == "1"
	}

	hasIAMAdminConfig := endpoint != "" && accessKey != "" && secretKey != ""
	hasIAMEndpoint := endpoint != ""
	hasConsoleConfig := consoleEndpoint != "" && consoleUsername != "" && consolePassword != ""

	if !hasIAMEndpoint && !hasConsoleConfig {
		resp.Diagnostics.AddError(
			"Missing Configuration",
			"The provider requires at least an IAM endpoint (for per-account resources) "+
				"or Console API credentials (console_endpoint, console_username, console_password) to be configured. "+
				"For admin-level account operations, also provide access_key and secret_key. "+
				"Set the credentials in the provider configuration or use the corresponding environment variables.",
		)
		return
	}

	var iamClient *client.IAMClient
	var consoleClient *client.ConsoleClient
	var s3Client *client.S3Client

	if hasIAMEndpoint {
		if hasIAMAdminConfig {
			iamClient = client.NewIAMClient(endpoint, accessKey, secretKey, insecureSkipVerify)
		} else {
			iamClient = client.NewIAMClient(endpoint, "", "", insecureSkipVerify)
		}
		s3Client = client.NewS3Client(endpoint, insecureSkipVerify)
		if region != "" {
			iamClient.Region = region
			s3Client.Region = region
		}
	}

	if hasConsoleConfig {
		consoleClient = client.NewConsoleClient(consoleEndpoint, consoleUsername, consolePassword, insecureSkipVerify)
	}

	clientData := &client.ProviderClients{
		IAM:     iamClient,
		Console: consoleClient,
		S3:      s3Client,
	}

	resp.DataSourceData = clientData
	resp.ResourceData = clientData
}

func (p *ScalityProvider) resourceFactories() []func() resource.Resource {
	return []func() resource.Resource{
		account.NewAccountResource,
		consoleaccount.NewConsoleAccountResource,
		accountaccesskey.NewAccountAccessKeyResource,
		bucket.NewBucketResource,
		bucketencryption.NewBucketEncryptionResource,
		bucketlifecycle.NewBucketLifecycleResource,
		bucketobjectlock.NewBucketObjectLockResource,
		bucketpolicy.NewBucketPolicyResource,
		bucketreplication.NewBucketReplicationResource,
		iampolicy.NewIAMPolicyResource,
		iamrole.NewIAMRoleResource,
		iamrolepolicyattachment.NewIAMRolePolicyAttachmentResource,
		user.NewUserResource,
		useraccesskey.NewUserAccessKeyResource,
		userpolicy.NewUserPolicyResource,
		group.NewGroupResource,
		groupmembership.NewGroupMembershipResource,
	}
}

func (p *ScalityProvider) Resources(ctx context.Context) []func() resource.Resource {
	return p.resourceFactories()
}

func (p *ScalityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		account.NewAccountDataSource,
		account.NewAccountsDataSource,
		bucket.NewBucketDataSource,
		bucket.NewBucketsDataSource,
		user.NewUserDataSource,
		user.NewUsersDataSource,
		group.NewGroupDataSource,
		group.NewGroupsDataSource,
		iampolicy.NewIAMPolicyDataSource,
		iampolicy.NewIAMPoliciesDataSource,
		iamrole.NewIAMRoleDataSource,
		iamrole.NewIAMRolesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScalityProvider{
			version: version,
		}
	}
}
