package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/qdrant/go-client/qdrant"
)

// Ensure FieldIndexResource implements resource.Resource.
var _ resource.Resource = &FieldIndexResource{}
var _ resource.ResourceWithImportState = &FieldIndexResource{}

func NewFieldIndexResource() resource.Resource {
	return &FieldIndexResource{}
}

// FieldIndexResource defines the resource implementation.
type FieldIndexResource struct {
	client *QdrantClient
}

// FieldIndexResourceModel describes the resource data model.
type FieldIndexResourceModel struct {
	Id             types.String `tfsdk:"id"`
	CollectionName types.String `tfsdk:"collection_name"`
	FieldName      types.String `tfsdk:"field_name"`
	FieldType      types.String `tfsdk:"field_type"`
}

func (r *FieldIndexResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_field_index"
}

func (r *FieldIndexResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Qdrant field index.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal ID of the field index (collection_name/field_name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"collection_name": schema.StringAttribute{
				MarkdownDescription: "Collection name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"field_name": schema.StringAttribute{
				MarkdownDescription: "Field name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"field_type": schema.StringAttribute{
				MarkdownDescription: "Field type. Options: Keyword, Integer, Float, Geo, Text, Bool, Datetime.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *FieldIndexResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FieldIndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FieldIndexResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var fieldType *qdrant.FieldType
	if !data.FieldType.IsNull() {
		ft := r.mapFieldType(data.FieldType.ValueString())
		fieldType = &ft
	}

	createReq := &qdrant.CreateFieldIndexCollection{
		CollectionName: data.CollectionName.ValueString(),
		FieldName:      data.FieldName.ValueString(),
		FieldType:      fieldType,
	}

	_, err := r.client.Points.CreateFieldIndex(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create field index",
			"An unexpected error occurred when creating the field index. "+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s/%s", data.CollectionName.ValueString(), data.FieldName.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FieldIndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FieldIndexResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	qdrantResp, err := r.client.Collections.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: data.CollectionName.ValueString(),
	})
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	info := qdrantResp.GetResult()
	payloadSchema := info.GetPayloadSchema()

	found := false
	if schemaInfo, ok := payloadSchema[data.FieldName.ValueString()]; ok {
		// If it has data_type, it's indexed
		if schemaInfo.DataType != qdrant.PayloadSchemaType_UnknownType {
			found = true
			data.FieldType = types.StringValue(r.unmapFieldType(schemaInfo.DataType))
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FieldIndexResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Field indexes are usually replaced if changed
	resp.Diagnostics.AddError("Update not supported", "Update is not supported for this resource yet. Changes will trigger replacement.")
}

func (r *FieldIndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FieldIndexResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Points.DeleteFieldIndex(ctx, &qdrant.DeleteFieldIndexCollection{
		CollectionName: data.CollectionName.ValueString(),
		FieldName:      data.FieldName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete field index",
			"An unexpected error occurred when deleting the field index. "+
				"Qdrant Client Error: "+err.Error(),
		)
		return
	}
}

func (r *FieldIndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by "collection_name/field_name"
	idParts := strings.Split(req.ID, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: collection_name/field_name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection_name"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("field_name"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *FieldIndexResource) mapFieldType(t string) qdrant.FieldType {
	switch t {
	case "Keyword":
		return qdrant.FieldType_FieldTypeKeyword
	case "Integer":
		return qdrant.FieldType_FieldTypeInteger
	case "Float":
		return qdrant.FieldType_FieldTypeFloat
	case "Geo":
		return qdrant.FieldType_FieldTypeGeo
	case "Text":
		return qdrant.FieldType_FieldTypeText
	case "Bool":
		return qdrant.FieldType_FieldTypeBool
	case "Datetime":
		return qdrant.FieldType_FieldTypeDatetime
	default:
		return qdrant.FieldType_FieldTypeKeyword
	}
}

func (r *FieldIndexResource) unmapFieldType(t qdrant.PayloadSchemaType) string {
	switch t {
	case qdrant.PayloadSchemaType_Keyword:
		return "Keyword"
	case qdrant.PayloadSchemaType_Integer:
		return "Integer"
	case qdrant.PayloadSchemaType_Float:
		return "Float"
	case qdrant.PayloadSchemaType_Geo:
		return "Geo"
	case qdrant.PayloadSchemaType_Text:
		return "Text"
	case qdrant.PayloadSchemaType_Bool:
		return "Bool"
	case qdrant.PayloadSchemaType_Datetime:
		return "Datetime"
	default:
		return "Unknown"
	}
}
