package iampolicy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &IAMPolicyDataSource{}
var _ datasource.DataSourceWithConfigure = &IAMPolicyDataSource{}

type IAMPolicyDataSource struct {
	client *client.IAMClient
}

func NewIAMPolicyDataSource() datasource.DataSource {
	return &IAMPolicyDataSource{}
}

type IAMPolicyDataSourceModel struct {
	AccountAccessKey types.String         `tfsdk:"account_access_key"`
	AccountSecretKey types.String         `tfsdk:"account_secret_key"`
	PolicyName       types.String         `tfsdk:"policy_name"`
	ID               types.String         `tfsdk:"id"`
	ARN              types.String         `tfsdk:"arn"`
	PolicyDocument   jsontypes.Normalized `tfsdk:"policy_document"`
}

func (d *IAMPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_policy"
}

func (d *IAMPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing customer-managed IAM policy by name within an account. Lookup walks `ListPolicies` (scope `Local`) and matches by name; returns the ARN and the default version's policy document.",

		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account that owns the policy.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account that owns the policy.",
				Required:            true,
				Sensitive:           true,
			},
			"policy_name": schema.StringAttribute{
				MarkdownDescription: "Name of the managed policy to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Same as `arn`.",
				Computed:            true,
			},
			"arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the managed policy.",
				Computed:            true,
			},
			"policy_document": schema.StringAttribute{
				MarkdownDescription: "JSON policy document of the default version.",
				Computed:            true,
				CustomType:          jsontypes.NormalizedType{},
			},
		},
	}
}

func (d *IAMPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An IAM endpoint must be configured to use the scality_iam_policy data source.",
		)
		return
	}

	d.client = clients.IAM
}

func (d *IAMPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IAMPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()
	policyName := data.PolicyName.ValueString()

	tflog.Debug(ctx, "Reading IAM managed policy via data source", map[string]any{
		"policy_name": policyName,
	})

	policies, err := d.client.ListPolicies(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list policies: %s", err))
		return
	}

	var match *client.PolicyListEntry
	for i := range policies {
		if policies[i].PolicyName == policyName {
			match = &policies[i]
			break
		}
	}

	if match == nil {
		resp.Diagnostics.AddError(
			"Policy Not Found",
			fmt.Sprintf("No managed policy named %q exists in this account.", policyName),
		)
		return
	}

	data.ID = types.StringValue(match.Arn)
	data.ARN = types.StringValue(match.Arn)

	if match.DefaultVersionId != "" {
		doc, err := d.client.GetManagedPolicyVersion(ctx, ak, sk, match.Arn, match.DefaultVersionId)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IAM policy document: %s", err))
			return
		}
		data.PolicyDocument = jsontypes.NewNormalizedValue(doc)
	} else {
		data.PolicyDocument = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
