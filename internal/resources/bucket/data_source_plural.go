package bucket

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/scality/terraform-provider-scality/internal/client"
)

var _ datasource.DataSource = &BucketsDataSource{}
var _ datasource.DataSourceWithConfigure = &BucketsDataSource{}

type BucketsDataSource struct {
	client *client.S3Client
}

func NewBucketsDataSource() datasource.DataSource {
	return &BucketsDataSource{}
}

type BucketsDataSourceModel struct {
	ID               types.String           `tfsdk:"id"`
	AccountAccessKey types.String           `tfsdk:"account_access_key"`
	AccountSecretKey types.String           `tfsdk:"account_secret_key"`
	Buckets          []BucketsListItemModel `tfsdk:"buckets"`
}

type BucketsListItemModel struct {
	Name         types.String `tfsdk:"name"`
	ARN          types.String `tfsdk:"arn"`
	CreationDate types.String `tfsdk:"creation_date"`
}

func (d *BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buckets"
}

func (d *BucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all S3 buckets owned by the calling account. Useful for inventory, audit, and dashboard tooling — `for_each` over the result and drill down with `data.scality_bucket` per-bucket. The list does not include versioning, tags, or object-lock state (those require per-bucket calls; use `data.scality_bucket` for the drill-down). Empty account returns an empty list, not an error.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier for the data source instance.",
				Computed:            true,
			},
			"account_access_key": schema.StringAttribute{
				MarkdownDescription: "Access key of the account whose buckets to list.",
				Required:            true,
				Sensitive:           true,
			},
			"account_secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret key of the account whose buckets to list.",
				Required:            true,
				Sensitive:           true,
			},
			"buckets": schema.ListNestedAttribute{
				MarkdownDescription: "List of buckets owned by the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Bucket name.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the bucket (`arn:aws:s3:::<bucket>`).",
							Computed:            true,
						},
						"creation_date": schema.StringAttribute{
							MarkdownDescription: "Bucket creation date.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *BucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	if clients.S3 == nil {
		resp.Diagnostics.AddError(
			"Missing S3 Client Configuration",
			"An IAM endpoint must be configured to use the scality_buckets data source.",
		)
		return
	}

	d.client = clients.S3
}

func (d *BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := data.AccountAccessKey.ValueString()
	sk := data.AccountSecretKey.ValueString()

	tflog.Debug(ctx, "Listing S3 buckets via data source", nil)

	buckets, err := d.client.ListBuckets(ctx, ak, sk)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list buckets: %s", err))
		return
	}

	data.ID = types.StringValue("scality_buckets")
	data.Buckets = make([]BucketsListItemModel, 0, len(buckets))
	for _, b := range buckets {
		data.Buckets = append(data.Buckets, BucketsListItemModel{
			Name:         types.StringValue(b.Name),
			ARN:          types.StringValue(fmt.Sprintf("arn:aws:s3:::%s", b.Name)),
			CreationDate: types.StringValue(b.CreationDate),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
