package iamrole

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &IAMRoleDataSource{}
var _ datasource.DataSourceWithConfigure = &IAMRoleDataSource{}

type IAMRoleDataSource struct {
	client *client.IAMClient
}

func NewIAMRoleDataSource() datasource.DataSource {
	return &IAMRoleDataSource{}
}

type IAMRoleDataSourceModel struct {
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	RoleName         types.String         `tfsdk:"role_name"`
	ID               types.String         `tfsdk:"id"`
	ARN              types.String         `tfsdk:"arn"`
	Path             types.String         `tfsdk:"path"`
	AssumeRolePolicy jsontypes.Normalized `tfsdk:"assume_role_policy"`
}

func (d *IAMRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (d *IAMRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing IAM role by name within an account. Useful for attaching Terraform-managed sub-resources (`scality_iam_role_policy_attachment`) to a role created outside the current Terraform configuration.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns the role.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns the role.",
				Required:            true,
				Sensitive:           true,
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "Name of the role to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `role_name`.",
				Computed:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the role.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "IAM path of the role (usually `/`).",
				Computed:            true,
			},
			"assume_role_policy": schema.StringAttribute{
				MarkdownDescription: "Trust policy (AssumeRolePolicyDocument) attached to the role, as a JSON string.",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
		},
	}
}

func (d *IAMRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_iam_role data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *IAMRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IAMRoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Reading IAM role via data source", map[string]any{
		"role_name": roleName,
	})

	role, err := d.client.GetRole(ctx, ak, sk, roleName)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role %q: %s", roleName, err))
		return
	}

	if role == nil {
		resp.Diagnostics.AddError(
			"Role Not Found",
			fmt.Sprintf("No IAM role named %q exists in this account.", roleName),
		)
		return
	}

	data.ID = types.StringValue(role.RoleName)
	data.ARN = types.StringValue(role.Arn)
	data.Path = types.StringValue(role.Path)

	if role.AssumeRolePolicyDocument != "" {
		if decoded, err := url.QueryUnescape(role.AssumeRolePolicyDocument); err == nil {
			data.AssumeRolePolicy = jsontypes.NewNormalizedValue(decoded)
		} else {
			data.AssumeRolePolicy = jsontypes.NewNormalizedValue(role.AssumeRolePolicyDocument)
		}
	} else {
		data.AssumeRolePolicy = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
