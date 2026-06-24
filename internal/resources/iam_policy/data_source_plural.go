package iampolicy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cmrh/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &IAMPoliciesDataSource{}
var _ datasource.DataSourceWithConfigure = &IAMPoliciesDataSource{}

type IAMPoliciesDataSource struct {
	client *client.IAMClient
}

func NewIAMPoliciesDataSource() datasource.DataSource {
	return &IAMPoliciesDataSource{}
}

type IAMPoliciesDataSourceModel struct {
	ID               types.String            `tfsdk:"id"`
	AccountAccessKey types.String            `tfsdk:"account_access_key"`
	AccountSecretKey types.String            `tfsdk:"account_secret_key"`
	Policies         []PoliciesListItemModel `tfsdk:"policies"`
}

type PoliciesListItemModel struct {
	PolicyID         types.String `tfsdk:"policy_id"`
	PolicyName       types.String `tfsdk:"policy_name"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
	DefaultVersionID types.String `tfsdk:"default_version_id"`
	AttachmentCount  types.Int64  `tfsdk:"attachment_count"`
	CreateDate       types.String `tfsdk:"create_date"`
	UpdateDate       types.String `tfsdk:"update_date"`
}

func (d *IAMPoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_policies"
}

func (d *IAMPoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all customer-managed (scope `Local`) IAM policies in the calling account. Policy documents are not included in the list — use `data.scality_iam_policy` for per-policy drill-down. Empty account returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account whose policies to list.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account whose policies to list.",
				Required:            true,
				Sensitive:           true,
			},
			"policies": schema.ListNestedAttribute{
				MarkdownDescription: "List of managed policies in the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"policy_id": schema.StringAttribute{
							MarkdownDescription: "Stable unique identifier for the policy.",
							Computed:            true,
						},
						"policy_name": schema.StringAttribute{
							MarkdownDescription: "Policy name.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the policy.",
							Computed:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "IAM path of the policy.",
							Computed:            true,
						},
						"default_version_id": schema.StringAttribute{
							MarkdownDescription: "Default version identifier for the policy document.",
							Computed:            true,
						},
						"attachment_count": schema.Int64Attribute{
							MarkdownDescription: "Number of principals (users, groups, roles) the policy is attached to.",
							Computed:            true,
						},
						"create_date": schema.StringAttribute{
							MarkdownDescription: "Policy creation date.",
							Computed:            true,
						},
						"update_date": schema.StringAttribute{
							MarkdownDescription: "Last update date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *IAMPoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_iam_policies data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *IAMPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IAMPoliciesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Listing IAM managed policies via data source", nil)

	policies, err := d.client.ListPolicies(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policies: %s", err))
		return
	}

	data.ID = types.StringValue("scality_iam_policies")
	data.Policies = make([]PoliciesListItemModel, 0, len(policies))
	for _, p := range policies {
		data.Policies = append(data.Policies, PoliciesListItemModel{
			PolicyID:         types.StringValue(p.PolicyId),
			PolicyName:       types.StringValue(p.PolicyName),
			ARN:              types.StringValue(p.Arn),
			Path:             types.StringValue(p.Path),
			DefaultVersionID: types.StringValue(p.DefaultVersionId),
			AttachmentCount:  types.Int64Value(int64(p.AttachmentCount)),
			CreateDate:       types.StringValue(p.CreateDate),
			UpdateDate:       types.StringValue(p.UpdateDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
