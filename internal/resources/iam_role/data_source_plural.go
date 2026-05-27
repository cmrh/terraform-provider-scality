package iamrole

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &IAMRolesDataSource{}
var _ datasource.DataSourceWithConfigure = &IAMRolesDataSource{}

type IAMRolesDataSource struct {
	client *client.IAMClient
}

func NewIAMRolesDataSource() datasource.DataSource {
	return &IAMRolesDataSource{}
}

type IAMRolesDataSourceModel struct {
	ID               types.String         `tfsdk:"id"`
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	Roles            []RolesListItemModel `tfsdk:"roles"`
}

type RolesListItemModel struct {
	RoleID     types.String `tfsdk:"role_id"`
	RoleName   types.String `tfsdk:"role_name"`
	ARN        types.String `tfsdk:"arn"`
	Path       types.String `tfsdk:"path"`
	CreateDate types.String `tfsdk:"create_date"`
}

func (d *IAMRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_roles"
}

func (d *IAMRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all IAM roles in the calling account. Trust policies (`assume_role_policy`) are not included in the list — use `data.scality_iam_role` for per-role drill-down. Empty account returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account whose roles to list.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account whose roles to list.",
				Required:            true,
				Sensitive:           true,
			},
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "List of IAM roles in the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							MarkdownDescription: "Stable unique identifier for the role.",
							Computed:            true,
						},
						"role_name": schema.StringAttribute{
							MarkdownDescription: "Role name.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the role.",
							Computed:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "IAM path of the role.",
							Computed:            true,
						},
						"create_date": schema.StringAttribute{
							MarkdownDescription: "Role creation date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *IAMRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_iam_roles data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *IAMRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IAMRolesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Listing IAM roles via data source", nil)

	roles, err := d.client.ListRoles(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}

	data.ID = types.StringValue("scality_iam_roles")
	data.Roles = make([]RolesListItemModel, 0, len(roles))
	for _, r := range roles {
		data.Roles = append(data.Roles, RolesListItemModel{
			RoleID:     types.StringValue(r.RoleId),
			RoleName:   types.StringValue(r.RoleName),
			ARN:        types.StringValue(r.Arn),
			Path:       types.StringValue(r.Path),
			CreateDate: types.StringValue(r.CreateDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
