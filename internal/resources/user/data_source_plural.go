package user

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &UsersDataSource{}
var _ datasource.DataSourceWithConfigure = &UsersDataSource{}

type UsersDataSource struct {
	client *client.IAMClient
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSourceModel struct {
	ID               types.String         `tfsdk:"id"`
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	Users            []UsersListItemModel `tfsdk:"users"`
}

type UsersListItemModel struct {
	UserID     types.String `tfsdk:"user_id"`
	Username   types.String `tfsdk:"username"`
	ARN        types.String `tfsdk:"arn"`
	Path       types.String `tfsdk:"path"`
	CreateDate types.String `tfsdk:"create_date"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all IAM users in the calling account. Useful for inventory, audit, and dashboard tooling — `for_each` over the result and drill down with `data.scality_user` per-user. Empty account returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account whose users to list.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account whose users to list.",
				Required:            true,
				Sensitive:           true,
			},
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "List of IAM users in the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.StringAttribute{
							MarkdownDescription: "Stable unique identifier for the user.",
							Computed:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "User name.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the user.",
							Computed:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "IAM path of the user.",
							Computed:            true,
						},
						"create_date": schema.StringAttribute{
							MarkdownDescription: "User creation date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_users data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Listing IAM users via data source", nil)

	users, err := d.client.ListUsers(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list users: %s", err))
		return
	}

	data.ID = types.StringValue("scality_users")
	data.Users = make([]UsersListItemModel, 0, len(users))
	for _, u := range users {
		data.Users = append(data.Users, UsersListItemModel{
			UserID:     types.StringValue(u.UserId),
			Username:   types.StringValue(u.UserName),
			ARN:        types.StringValue(u.Arn),
			Path:       types.StringValue(u.Path),
			CreateDate: types.StringValue(u.CreateDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
