package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/qdrant/go-client/qdrant"
)

// Ensure CollectionsDataSource implements datasource.DataSource.
var _ datasource.DataSource = &CollectionsDataSource{}

func NewCollectionsDataSource() datasource.DataSource {
	return &CollectionsDataSource{}
}

// CollectionsDataSource defines the data source implementation.
type CollectionsDataSource struct {
	client *QdrantClient
}

// CollectionsDataSourceModel describes the data source data model.
type CollectionsDataSourceModel struct {
	Collections []types.String `tfsdk:"collections"`
}

func (d *CollectionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collections"
}

func (d *CollectionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of all collection names.",

		Attributes: map[string]schema.Attribute{
			"collections": schema.ListAttribute{
				MarkdownDescription: "List of collection names.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *CollectionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*QdrantClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *QdrantClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *CollectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CollectionsDataSourceModel

	qdrantResp, err := d.client.Collections.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list collections",
			"An unexpected error occurred when listing collections. "+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}

	for _, collection := range qdrantResp.GetCollections() {
		data.Collections = append(data.Collections, types.StringValue(collection.GetName()))
	}

	// Write logs using the tflog package
	// tflog.Trace(ctx, "read collections data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
