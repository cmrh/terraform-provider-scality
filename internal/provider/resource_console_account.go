package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &ConsoleAccountResource{}
var _ resource.ResourceWithImportState = &ConsoleAccountResource{}

func NewConsoleAccountResource() resource.Resource {
	return &ConsoleAccountResource{}
}

// ConsoleAccountResource defines the resource implementation
type ConsoleAccountResource struct {
	client *ConsoleClient
}

// ConsoleAccountResourceModel describes the resource data model
type ConsoleAccountResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	AccountName            types.String `tfsdk:"account_name"`
	Email                  types.String `tfsdk:"email"`
	Quota                  types.Int64  `tfsdk:"quota"`
	GenerateRandomPassword types.Bool   `tfsdk:"generate_random_password"`
	PasswordLength         types.Int64  `tfsdk:"password_length"`
	Password               types.String `tfsdk:"password"`
	CreatedAt              types.String `tfsdk:"created_at"`
	AccessKey              types.String `tfsdk:"access_key"`
	SecretKey              types.String `tfsdk:"secret_key"`
}

const (
	// Password generation constants
	minPasswordLength     = 16
	defaultPasswordLength = 16
)

// generateRandomPassword generates a cryptographically secure random password.
//
// The password uses crypto/rand for secure randomness and includes:
//   - Uppercase letters (excluding O)
//   - Lowercase letters (excluding l)
//   - Digits (excluding 0, 1)
//   - Special characters (!@#$%^&*-_=+?)
//
// Ambiguous characters (0, O, 1, l, I) are excluded for clarity.
// Minimum length is 16 characters regardless of the requested length.
//
// Returns the generated password or an error if randomness generation fails.
func generateRandomPassword(length int) (string, error) {
	if length < minPasswordLength {
		length = minPasswordLength
	}

	// Character sets - excluding ambiguous characters (0, O, 1, l, I)
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address for the account",
				Required:            true,
			},
			"quota": schema.Int64Attribute{
				MarkdownDescription: "Maximum amount of bytes storable by the account (0 = unlimited)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"generate_random_password": schema.BoolAttribute{
				MarkdownDescription: "Generate a random password for Console access (optional, default false)",
				Optional:            true,
			},
			"password_length": schema.Int64Attribute{
				MarkdownDescription: "Length of generated password (default 16, only used if generate_random_password is true)",
				Optional:            true,
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

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clients.ConsoleClient == nil {
		resp.Diagnostics.AddError(
			"Missing Console Client Configuration",
			"Console API credentials (console_endpoint, console_username, console_password) must be configured to use scality_console_account resource.",
		)
		return
	}

	r.client = clients.ConsoleClient
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

	// Build account creation request
	createReq := ConsoleAccountCreateRequest{
		AccountName: data.AccountName.ValueString(),
		Email:       data.Email.ValueString(),
		Quota:       data.Quota.ValueInt64(),
	}

	// Generate random password if requested
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
	}

	account, err := r.client.CreateConsoleAccount(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Console account: %s", err))
		return
	}

	// Generate persistent access keys
	tflog.Debug(ctx, "Generating persistent access keys for Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	accessKey, err := r.client.GenerateConsoleAccessKey(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to generate access key: %s", err))
		return
	}

	// Update model with response data
	data.ID = types.StringValue(account.Data.AccountName)
	data.CreatedAt = types.StringValue(account.Data.CreatedAt)
	data.AccessKey = types.StringValue(accessKey.Data.AccessKey)
	data.SecretKey = types.StringValue(accessKey.Data.SecretKey)

	// Set default quota if not specified
	if data.Quota.IsNull() || data.Quota.IsUnknown() {
		data.Quota = types.Int64Value(account.Data.Quota)
	}

	tflog.Trace(ctx, "Created Console account resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsoleAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConsoleAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	// Get account details
	account, err := r.client.GetConsoleAccount(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Console account: %s", err))
		return
	}

	// Account was deleted outside Terraform
	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with refreshed data
	// Note: Console API may return different field names, adjust as needed
	if accountData, ok := account["data"].(map[string]interface{}); ok {
		if email, ok := accountData["email"].(string); ok {
			data.Email = types.StringValue(email)
		}
		if quota, ok := accountData["quota"].(float64); ok {
			data.Quota = types.Int64Value(int64(quota))
		}
		if createdAt, ok := accountData["createdAt"].(string); ok {
			data.CreatedAt = types.StringValue(createdAt)
		}
	}

	// Keep access key and secret key from state (they can't be retrieved)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConsoleAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConsoleAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Console account", map[string]interface{}{
		"account_name": data.AccountName.ValueString(),
	})

	// Note: The Console API may not support updates.
	// For now, we'll just update the state with the planned values.
	// In a production provider, you would implement UpdateAccount API calls here.

	resp.Diagnostics.AddWarning(
		"Update Not Fully Implemented",
		"Console account updates may require replacement. Check the Scality Console API documentation for update capabilities.",
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
