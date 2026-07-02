package consoleaccount

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
	"github.com/cmrh/terraform-provider-scality/internal/validators"
)

var _ resource.Resource = &ConsoleAccountResource{}
var _ resource.ResourceWithImportState = &ConsoleAccountResource{}

const (
	minPasswordLength     = 16
	defaultPasswordLength = 16
)

type ConsoleAccountResource struct {
	client    *client.ConsoleClient
	iamClient *client.IAMClient
}

func NewConsoleAccountResource() resource.Resource {
	return &ConsoleAccountResource{}
}

func generateRandomPassword(length int) (string, error) {
	if length < minPasswordLength {
		length = minPasswordLength
	}

	const (
		upperChars   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lowerChars   = "abcdefghijkmnopqrstuvwxyz"
		digitChars   = "23456789"
		specialChars = "!@#$%^&*-_=+?"
	)

	allChars := upperChars + lowerChars + digitChars + specialChars

	password := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(allChars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		password[i] = allChars[num.Int64()]
	}

	return string(password), nil
}

func (r *ConsoleAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_console_account"
}

func (r *ConsoleAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Scality account via Console API with JWT authentication. " +
			"Optionally generates a random password for Console access. Persistent S3 credentials are generated automatically.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Account identifier (same as account_name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_name": schema.StringAttribute{
				MarkdownDescription: "Name of the account",
				Required:            true,
				Validators:          validators.AccountName(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address for the account",
				Required:            true,
				Validators:          validators.Email(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"quota": schema.Int64Attribute{
				MarkdownDescription: "Maximum amount of bytes storable by the account (0 = unlimited)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"generate_random_password": schema.BoolAttribute{
				MarkdownDescription: "Generate a random password for Console access (optional, default false)",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"password_length": schema.Int64Attribute{
				MarkdownDescription: "Length of generated password (default 16, only used if generate_random_password is true)",
				Optional:            true,
				Validators:          validators.Int64AtLeast(16),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Generated Console password (only available if generate_random_password is true)",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Account creation timestamp",
				Computed:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "S3 API access key (persistent credentials, generated automatically)",
				Computed:            true,
				Sensitive:           true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "S3 API secret key (persistent credentials, generated automatically)",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *ConsoleAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clients.Console == nil {
		resp.Diagnostics.AddError(
			"Missing Console Client Configuration",
			"Console API credentials (console_endpoint, console_username, console_password) must be configured to use scality_console_account resource.",
		)
		return
	}

	r.client = clients.Console
	r.iamClient = clients.IAM
}

func (r *ConsoleAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConsoleAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	createReq := client.ConsoleAccountCreateRequest{
		AccountName: data.AccountName.ValueString(),
		Email:       data.Email.ValueString(),
		Quota:       data.Quota.ValueInt64(),
	}

	if !data.GenerateRandomPassword.IsNull() && data.GenerateRandomPassword.ValueBool() {
		passwordLength := defaultPasswordLength
		if !data.PasswordLength.IsNull() && data.PasswordLength.ValueInt64() > 0 {
			passwordLength = int(data.PasswordLength.ValueInt64())
		}

		password, err := generateRandomPassword(passwordLength)
		if err != nil {
			resp.Diagnostics.AddError("Password Generation Error", fmt.Sprintf("Unable to generate random password: %s", err))
			return
		}

		createReq.Password = password
		data.Password = types.StringValue(password)

		tflog.Debug(ctx, "Generated random password for Console account", map[string]interface{}{
			"account_name":    data.AccountName.ValueString(),
			"password_length": passwordLength,
		})
	} else {
		// password is Computed; resolve it so state isn't left unknown after apply.
		data.Password = types.StringNull()
	}

	account, err := r.client.CreateConsoleAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Console account: %s", err))
		return
	}

	data.ID = types.StringValue(account.Account.Name)
	data.CreatedAt = types.StringValue(account.Account.CreateDate)

	if data.Quota.IsNull() || data.Quota.IsUnknown() {
		data.Quota = types.Int64Value(account.Account.QuotaMax)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Debug(ctx, "Generating persistent access keys for Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	accessKey, err := r.client.GenerateConsoleAccessKey(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Account created successfully but access key generation failed. "+
				"The account exists and is tracked in state. Run apply again to retry: %s", err))
		return
	}

	data.AccessKey = types.StringValue(accessKey.Key.ID)
	data.SecretKey = types.StringValue(accessKey.Key.Value)

	tflog.Trace(ctx, "Created Console account resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsoleAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsoleAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Console API has no GetAccount. When IAM admin credentials are configured,
	// probe Vault to surface out-of-band deletion; otherwise preserve state.
	if r.iamClient != nil && r.iamClient.AccessKey != "" {
		name := data.AccountName.ValueString()
		acct, err := r.iamClient.GetAccount(ctx, name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to verify console account %q via Vault: %s", name, err))
			return
		}
		if acct == nil {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsoleAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All scality_console_account attributes require resource replacement. This is a provider bug if you see this error.",
	)
}

func (r *ConsoleAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConsoleAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	err := r.client.DeleteConsoleAccount(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Console account: %s", err))
		return
	}

	tflog.Trace(ctx, "Deleted Console account resource")
}

func (r *ConsoleAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("account_name"), req, resp)
}
