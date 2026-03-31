package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ScalityProvider satisfies various provider interfaces
var _ provider.Provider = &ScalityProvider{}

// ScalityProvider defines the provider implementation
type ScalityProvider struct {
	version string
}

// ScalityProviderModel describes the provider data model
type ScalityProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	AccessKey          types.String `tfsdk:"access_key"`
	SecretKey          types.String `tfsdk:"secret_key"`
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
				Description: "Scality IAM API endpoint (e.g., http://10.164.169.247). Can also be set via SCALITY_ENDPOINT environment variable.",
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
			"console_endpoint": schema.StringAttribute{
				Description: "Scality Console API endpoint (e.g., http://10.164.169.247:8080). Can also be set via SCALITY_CONSOLE_ENDPOINT environment variable.",
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

	// Priority: configuration > environment variables
	// IAM API credentials
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

	// Console API credentials
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

	// TLS configuration - default to false (secure by default)
	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	} else if envVal := os.Getenv("SCALITY_INSECURE_SKIP_VERIFY"); envVal != "" {
		insecureSkipVerify = envVal == "true" || envVal == "1"
	}

	// Validate that at least one set of credentials is configured
	hasIAMConfig := endpoint != "" && accessKey != "" && secretKey != ""
	hasConsoleConfig := consoleEndpoint != "" && consoleUsername != "" && consolePassword != ""

	if !hasIAMConfig && !hasConsoleConfig {
		resp.Diagnostics.AddError(
			"Missing Configuration",
			"The provider requires either IAM API credentials (endpoint, access_key, secret_key) "+
				"or Console API credentials (console_endpoint, console_username, console_password) to be configured. "+
				"Set the credentials in the provider configuration or use the corresponding environment variables.",
		)
		return
	}

	// Create clients based on available credentials
	var iamClient *ScalityClient
	var consoleClient *ConsoleClient

	if hasIAMConfig {
		iamClient = NewScalityClient(endpoint, accessKey, secretKey, insecureSkipVerify)
	}

	if hasConsoleConfig {
		consoleClient = NewConsoleClient(consoleEndpoint, consoleUsername, consolePassword, insecureSkipVerify)
	}

	// Store both clients in a wrapper struct
	clientData := &ProviderClients{
		IAMClient:     iamClient,
		ConsoleClient: consoleClient,
	}

	// Make clients available to resources and data sources
	resp.DataSourceData = clientData
	resp.ResourceData = clientData
}

// ProviderClients holds both IAM and Console clients
type ProviderClients struct {
	IAMClient     *ScalityClient
	ConsoleClient *ConsoleClient
}

func (p *ScalityProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccountResource,
		NewConsoleAccountResource,
	}
}

func (p *ScalityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ScalityProvider{
			version: version,
		}
	}
}
