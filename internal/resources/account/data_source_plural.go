package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &AccountsDataSource{}
var _ datasource.DataSourceWithConfigure = &AccountsDataSource{}

type AccountsDataSource struct {
	client *client.IAMClient
}

func NewAccountsDataSource() datasource.DataSource {
	return &AccountsDataSource{}
}

type AccountsDataSourceModel struct {
	ID       types.String            `tfsdk:"id"`
	Accounts []AccountsListItemModel `tfsdk:"accounts"`
}

type AccountsListItemModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	ARN          types.String `tfsdk:"arn"`
	CanonicalID  types.String `tfsdk:"canonical_id"`
	CreateDate   types.String `tfsdk:"create_date"`
	QuotaMax     types.Int64  `tfsdk:"quota_max"`
}

func (d *AccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accounts"
}

func (d *AccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all Scality accounts in the cluster. Useful for inventory, audit, and dashboard tooling — `for_each` over the result and drill down with `data.scality_account` per-account. The list does not include `custom_attributes` (those require a per-account call; use `data.scality_account` for the drill-down). Empty cluster returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"accounts": schema.ListNestedAttribute{
				MarkdownDescription: "List of accounts in the cluster.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Vault account ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Account name.",
							Computed:            true,
						},
						"email_address": schema.StringAttribute{
							MarkdownDescription: "Email address registered against the account.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "Amazon Resource Name of the account.",
							Computed:            true,
						},
						"canonical_id": schema.StringAttribute{
							MarkdownDescription: "Canonical ID of the account, used in S3 bucket policies.",
							Computed:            true,
						},
						"create_date": schema.StringAttribute{
							MarkdownDescription: "Account creation date.",
							Computed:            true,
						},
						"quota_max": schema.Int64Attribute{
							MarkdownDescription: "Maximum bytes storable by the account (0 = unlimited).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *AccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clients.IAM == nil || clients.IAM.AccessKey == "" {
		resp.Diagnostics.AddError(
			"Missing IAM Admin Credentials",
			"IAM API admin credentials (endpoint, access_key, secret_key) must all be configured to use the scality_accounts data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *AccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing Scality accounts via data source", nil)

	accounts, err := d.client.ListAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list accounts: %s", err))
		return
	}

	data.ID = types.StringValue("scality_accounts")
	data.Accounts = make([]AccountsListItemModel, 0, len(accounts))
	for _, a := range accounts {
		data.Accounts = append(data.Accounts, AccountsListItemModel{
			ID:           types.StringValue(a.ID),
			Name:         types.StringValue(a.Name),
			EmailAddress: types.StringValue(a.EmailAddress),
			ARN:          types.StringValue(a.ARN),
			CanonicalID:  types.StringValue(a.CanonicalID),
			CreateDate:   types.StringValue(a.CreateDate),
			QuotaMax:     types.Int64Value(a.QuotaMax),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
