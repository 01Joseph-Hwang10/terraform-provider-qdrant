package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/qdrant/go-client/qdrant"
)

// Ensure CollectionResource implements resource.Resource.
var _ resource.Resource = &CollectionResource{}
var _ resource.ResourceWithImportState = &CollectionResource{}

func NewCollectionResource() resource.Resource {
	return &CollectionResource{}
}

// CollectionResource defines the resource implementation.
type CollectionResource struct {
	client *QdrantClient
}

// CollectionResourceModel describes the resource data model.
type CollectionResourceModel struct {
	Name    types.String  `tfsdk:"name"`
	Vectors []VectorModel `tfsdk:"vectors"`
}

type VectorModel struct {
	Size     types.Int64  `tfsdk:"size"`
	Distance types.String `tfsdk:"distance"`
	Name     types.String `tfsdk:"name"`
}

func (r *CollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *CollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Qdrant collection.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Collection name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vectors": schema.ListNestedAttribute{
				MarkdownDescription: "Vector configuration.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							MarkdownDescription: "Vector size.",
							Required:            true,
						},
						"distance": schema.StringAttribute{
							MarkdownDescription: "Distance metric. Options: Cosine, Euclidian, Dot.",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Vector name. Optional if only one vector is defined.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (r *CollectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*QdrantClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *QdrantClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *CollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare Vectors Config
	var vectorsConfig *qdrant.VectorsConfig
	if len(data.Vectors) == 1 && data.Vectors[0].Name.IsNull() {
		// Single unnamed vector
		vectorsConfig = &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(data.Vectors[0].Size.ValueInt64()),
					Distance: r.mapDistance(data.Vectors[0].Distance.ValueString()),
				},
			},
		}
	} else {
		// Multiple or named vectors
		mapParams := make(map[string]*qdrant.VectorParams)
		for _, v := range data.Vectors {
			name := v.Name.ValueString()
			if name == "" {
				name = "" // Should we error if multiple and no name?
			}
			mapParams[name] = &qdrant.VectorParams{
				Size:     uint64(v.Size.ValueInt64()),
				Distance: r.mapDistance(v.Distance.ValueString()),
			}
		}
		vectorsConfig = &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_ParamsMap{
				ParamsMap: &qdrant.VectorParamsMap{
					Map: mapParams,
				},
			},
		}
	}

	createReq := &qdrant.CreateCollection{
		CollectionName: data.Name.ValueString(),
		VectorsConfig:  vectorsConfig,
	}

	_, err := r.client.Collections.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create collection",
			"An unexpected error occurred when creating the collection. "+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	qdrantResp, err := r.client.Collections.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: data.Name.ValueString(),
	})
	if err != nil {
		// If collection not found, remove from state
		// Note: Checking gRPC error codes would be better
		resp.State.RemoveResource(ctx)
		return
	}

	info := qdrantResp.GetResult()
	config := info.GetConfig()
	vConfig := config.GetParams().GetVectorsConfig()

	data.Vectors = []VectorModel{}

	if params := vConfig.GetParams(); params != nil {
		data.Vectors = append(data.Vectors, VectorModel{
			Size:     types.Int64Value(int64(params.Size)),
			Distance: types.StringValue(r.unmapDistance(params.Distance)),
			Name:     types.StringNull(),
		})
	} else if paramsMap := vConfig.GetParamsMap(); paramsMap != nil {
		for name, params := range paramsMap.GetMap() {
			data.Vectors = append(data.Vectors, VectorModel{
				Size:     types.Int64Value(int64(params.Size)),
				Distance: types.StringValue(r.unmapDistance(params.Distance)),
				Name:     types.StringValue(name),
			})
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Qdrant allows updating some collection parameters, but vectors configuration is usually immutable.
	// For simplicity, we'll mark vectors as RequiresReplace (which is done in Schema for now via Name)
	// Actually, Qdrant's UpdateCollection API is for optimizers, etc.
	// Since we only have Name and Vectors, and Name is RequiresReplace, any change to Vectors will also require replace if not supported.
	
	// If we want to support updating other fields later, we'd implement it here.
	resp.Diagnostics.AddError("Update not supported", "Update is not supported for this resource yet. Changes will trigger replacement.")
}

func (r *CollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Collections.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete collection",
			"An unexpected error occurred when deleting the collection. "+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}
}

func (r *CollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *CollectionResource) mapDistance(d string) qdrant.Distance {
	switch d {
	case "Cosine":
		return qdrant.Distance_Cosine
	case "Euclidian":
		return qdrant.Distance_Euclid
	case "Dot":
		return qdrant.Distance_Dot
	case "Manhattan":
		return qdrant.Distance_Manhattan
	default:
		return qdrant.Distance_UnknownDistance
	}
}

func (r *CollectionResource) unmapDistance(d qdrant.Distance) string {
	switch d {
	case qdrant.Distance_Cosine:
		return "Cosine"
	case qdrant.Distance_Euclid:
		return "Euclidian"
	case qdrant.Distance_Dot:
		return "Dot"
	case qdrant.Distance_Manhattan:
		return "Manhattan"
	default:
		return "Unknown"
	}
}
