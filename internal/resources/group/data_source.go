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

var _ datasource.DataSource = &GroupDataSource{}
var _ datasource.DataSourceWithConfigure = &GroupDataSource{}

type GroupDataSource struct {
	client *client.IAMClient
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

type GroupDataSourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	GroupName        types.String `tfsdk:"group_name"`
	ID               types.String `tfsdk:"id"`
	GroupID          types.String `tfsdk:"group_id"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing IAM group by name within an account. Useful for attaching Terraform-managed sub-resources (`scality_group_membership`) to a group created outside the current Terraform configuration.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns the group.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns the group.",
				Required:            true,
				Sensitive:           true,
			},
			"group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the group to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `group_id`.",
				Computed:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "Stable unique identifier for the group.",
				Computed:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the group.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "IAM path of the group (usually `/`).",
				Computed:            true,
			},
		},
	}
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_group data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	groupName := data.GroupName.ValueString()

	tflog.Debug(ctx, "Reading IAM group via data source", map[string]any{
		"group_name": groupName,
	})

	group, _, err := d.client.GetGroup(ctx, ak, sk, groupName)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group %q: %s", groupName, err))
		return
	}

	if group == nil {
		resp.Diagnostics.AddError(
			"Group Not Found",
			fmt.Sprintf("No IAM group named %q exists in this account.", groupName),
		)
		return
	}

	data.ID = types.StringValue(group.GroupId)
	data.GroupID = types.StringValue(group.GroupId)
	data.ARN = types.StringValue(group.Arn)
	data.Path = types.StringValue(group.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
