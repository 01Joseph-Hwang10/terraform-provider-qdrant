package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Ensure QdrantProvider implements provider.Provider.
var _ provider.Provider = &QdrantProvider{}

// QdrantProvider defines the provider implementation.
type QdrantProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built locally.
	version string
}

// QdrantProviderModel describes the provider data model.
type QdrantProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Https    types.Bool   `tfsdk:"https"`
	Insecure types.Bool   `tfsdk:"insecure"`
	ApiKey   types.String `tfsdk:"api_key"`
}

func (p *QdrantProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "qdrant"
	resp.Version = p.version
}

func (p *QdrantProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Qdrant host. May also be provided via QDRANT_HOST environment variable.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Qdrant gRPC port. Defaults to 6334. May also be provided via QDRANT_PORT environment variable.",
				Optional:            true,
			},
			"https": schema.BoolAttribute{
				MarkdownDescription: "Whether to use HTTPS. Defaults to false. May also be provided via QDRANT_HTTPS environment variable.",
				Optional:            true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Whether to skip TLS verification. Defaults to false. May also be provided via QDRANT_INSECURE environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Qdrant API key. May also be provided via QDRANT_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

type QdrantClient struct {
	Collections qdrant.CollectionsClient
	Points      qdrant.PointsClient
	Conn        *grpc.ClientConn
	ApiKey      string
}

func (p *QdrantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data QdrantProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("QDRANT_HOST")
	port := int64(6334)
	if os.Getenv("QDRANT_PORT") != "" {
		fmt.Sscanf(os.Getenv("QDRANT_PORT"), "%d", &port)
	}
	https := os.Getenv("QDRANT_HTTPS") == "true"
	tlsInsecure := os.Getenv("QDRANT_INSECURE") == "true"
	apiKey := os.Getenv("QDRANT_API_KEY")

	if !data.Host.IsNull() {
		host = data.Host.ValueString()
	}
	if !data.Port.IsNull() {
		port = data.Port.ValueInt64()
	}
	if !data.Https.IsNull() {
		https = data.Https.ValueBool()
	}
	if !data.Insecure.IsNull() {
		tlsInsecure = data.Insecure.ValueBool()
	}
	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Qdrant Host",
			"The provider cannot create the Qdrant client as there is no host configured. "+
				"Set the host argument in the provider configuration or use the QDRANT_HOST environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	var dialOpts []grpc.DialOption
	if https {
		creds := credentials.NewClientTLSFromCert(nil, "")
		if tlsInsecure {
			creds = credentials.NewTLS(nil) // Simplified for now
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Unary interceptor for API Key
	if apiKey != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx = metadata.AppendToOutgoingContext(ctx, "api-key", apiKey)
			return invoker(ctx, method, req, reply, cc, opts...)
		}))
		dialOpts = append(dialOpts, grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			ctx = metadata.AppendToOutgoingContext(ctx, "api-key", apiKey)
			return streamer(ctx, desc, cc, method, opts...)
		}))
	}

	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Qdrant gRPC client",
			"An unexpected error occurred when creating the Qdrant gRPC client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}

	client := &QdrantClient{
		Collections: qdrant.NewCollectionsClient(conn),
		Points:      qdrant.NewPointsClient(conn),
		Conn:        conn,
		ApiKey:      apiKey,
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *QdrantProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCollectionResource,
		NewFieldIndexResource,
	}
}

func (p *QdrantProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCollectionsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &QdrantProvider{
			version: version,
		}
	}
}
