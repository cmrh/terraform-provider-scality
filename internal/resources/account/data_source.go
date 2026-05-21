package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &AccountDataSource{}
var _ datasource.DataSourceWithConfigure = &AccountDataSource{}

type AccountDataSource struct {
	client *client.IAMClient
}

func NewAccountDataSource() datasource.DataSource {
	return &AccountDataSource{}
}

type AccountDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	EmailAddress     types.String `tfsdk:"email_address"`
	QuotaMax         types.Int64  `tfsdk:"quota_max"`
	CustomAttributes types.Map    `tfsdk:"custom_attributes"`
	ARN              types.String `tfsdk:"arn"`
	CanonicalID      types.String `tfsdk:"canonical_id"`
	CreateDate       types.String `tfsdk:"create_date"`
}

func (d *AccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *AccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing Scality account by name. Useful for referencing accounts created outside the current Terraform configuration — pass the looked-up ID, ARN, or canonical ID into downstream resources or modules without hardcoding. Does not expose `access_key`/`secret_key`: the IAM API returns those only at account creation. To get keys for an existing account, mint a new pair via `scality_account_access_key` against an existing root key.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the account to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Vault account ID.",
				Computed:            true,
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Email address registered against the account.",
				Computed:            true,
			},
			"quota_max": schema.Int64Attribute{
				MarkdownDescription: "Maximum bytes storable by the account (0 = unlimited).",
				Computed:            true,
			},
			"custom_attributes": schema.MapAttribute{
				MarkdownDescription: "Custom attributes (key-value string pairs) attached to the account.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "Amazon Resource Name (ARN) of the account.",
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
		},
	}
}

func (d *AccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"IAM API admin credentials (endpoint, access_key, secret_key) must all be configured to use the scality_account data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *AccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	tflog.Debug(ctx, "Reading Scality account via data source", map[string]any{
		"name": name,
	})

	account, err := d.client.GetAccount(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account %q: %s", name, err))
		return
	}

	if account == nil {
		resp.Diagnostics.AddError(
			"Account Not Found",
			fmt.Sprintf("No Scality account named %q exists.", name),
		)
		return
	}

	data.ID = types.StringValue(account.ID)
	data.EmailAddress = types.StringValue(account.EmailAddress)
	data.QuotaMax = types.Int64Value(account.QuotaMax)
	data.ARN = types.StringValue(account.ARN)
	data.CanonicalID = types.StringValue(account.CanonicalID)
	data.CreateDate = types.StringValue(account.CreateDate)

	if len(account.CustomAttributes) > 0 {
		elements := make(map[string]attr.Value, len(account.CustomAttributes))
		for k, v := range account.CustomAttributes {
			elements[k] = types.StringValue(v)
		}
		data.CustomAttributes = types.MapValueMust(types.StringType, elements)
	} else {
		data.CustomAttributes = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
