package group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &GroupsDataSource{}
var _ datasource.DataSourceWithConfigure = &GroupsDataSource{}

type GroupsDataSource struct {
	client *client.IAMClient
}

func NewGroupsDataSource() datasource.DataSource {
	return &GroupsDataSource{}
}

type GroupsDataSourceModel struct {
	ID               types.String          `tfsdk:"id"`
	AccountAccessKey types.String          `tfsdk:"account_access_key"`
	AccountSecretKey types.String          `tfsdk:"account_secret_key"`
	Groups           []GroupsListItemModel `tfsdk:"groups"`
}

type GroupsListItemModel struct {
	GroupID    types.String `tfsdk:"group_id"`
	GroupName  types.String `tfsdk:"group_name"`
	ARN        types.String `tfsdk:"arn"`
	Path       types.String `tfsdk:"path"`
	CreateDate types.String `tfsdk:"create_date"`
}

func (d *GroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *GroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all IAM groups in the calling account. Useful for inventory, audit, and dashboard tooling — `for_each` over the result and drill down with `data.scality_group` per-group. Empty account returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account whose groups to list.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account whose groups to list.",
				Required:            true,
				Sensitive:           true,
			},
			"groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of IAM groups in the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							MarkdownDescription: "Stable unique identifier for the group.",
							Computed:            true,
						},
						"group_name": schema.StringAttribute{
							MarkdownDescription: "Group name.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the group.",
							Computed:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "IAM path of the group.",
							Computed:            true,
						},
						"create_date": schema.StringAttribute{
							MarkdownDescription: "Group creation date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *GroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_groups data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *GroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Listing IAM groups via data source", nil)

	groups, err := d.client.ListGroups(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list groups: %s", err))
		return
	}

	data.ID = types.StringValue("scality_groups")
	data.Groups = make([]GroupsListItemModel, 0, len(groups))
	for _, g := range groups {
		data.Groups = append(data.Groups, GroupsListItemModel{
			GroupID:    types.StringValue(g.GroupId),
			GroupName:  types.StringValue(g.GroupName),
			ARN:        types.StringValue(g.Arn),
			Path:       types.StringValue(g.Path),
			CreateDate: types.StringValue(g.CreateDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
