package user

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &UserDataSource{}
var _ datasource.DataSourceWithConfigure = &UserDataSource{}

type UserDataSource struct {
	client *client.IAMClient
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Username         types.String `tfsdk:"username"`
	ID               types.String `tfsdk:"id"`
	UserID           types.String `tfsdk:"user_id"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing IAM user by name within an account. Useful for attaching Terraform-managed sub-resources (`scality_user_policy`, `scality_user_access_key`, `scality_group_membership`) to a user created outside the current Terraform configuration.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns the user.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns the user.",
				Required:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Name of the user to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `user_id`.",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "Stable unique identifier for the user.",
				Computed:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the user.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "IAM path of the user (usually `/`).",
				Computed:            true,
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T.", req.ProviderData),
		)
		return
	}

	if clients.IAM == nil {
		resp.Diagnostics.AddError(
			"Missing IAM Client Configuration",
			"An IAM endpoint must be configured to use the scality_user data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	username := data.Username.ValueString()

	tflog.Debug(ctx, "Reading IAM user via data source", map[string]any{
		"username": username,
	})

	user, err := d.client.GetUser(ctx, ak, sk, username)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user %q: %s", username, err))
		return
	}

	if user == nil {
		resp.Diagnostics.AddError(
			"User Not Found",
			fmt.Sprintf("No IAM user named %q exists in this account.", username),
		)
		return
	}

	data.ID = types.StringValue(user.UserId)
	data.UserID = types.StringValue(user.UserId)
	data.ARN = types.StringValue(user.Arn)
	data.Path = types.StringValue(user.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
