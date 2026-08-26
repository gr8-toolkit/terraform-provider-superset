// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-superset/internal/client"
)

var (
	_ resource.Resource                = &annotationLayerResource{}
	_ resource.ResourceWithConfigure   = &annotationLayerResource{}
	_ resource.ResourceWithImportState = &annotationLayerResource{}
)

// NewAnnotationLayerResource is a helper function to simplify the provider implementation.
func NewAnnotationLayerResource() resource.Resource {
	return &annotationLayerResource{}
}

type annotationLayerResource struct {
	client *client.Client
}

type annotationLayerResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *annotationLayerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_annotation_layer"
}

func (r *annotationLayerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an annotation layer in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the annotation layer.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the annotation layer.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the annotation layer. Maps to the 'descr' API field.",
				Optional:    true,
			},
		},
	}
}

func (r *annotationLayerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *annotationLayerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting annotation layer Create")

	var plan annotationLayerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	description := ""
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description = plan.Description.ValueString()
	}

	al, err := r.client.CreateAnnotationLayer(plan.Name.ValueString(), description)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Superset Annotation Layer",
			fmt.Sprintf("CreateAnnotationLayer failed: %s", err.Error()))
		return
	}

	plan.ID = types.Int64Value(al.ID)
	plan.Name = types.StringValue(al.Name)
	if plan.Description.IsNull() || plan.Description.IsUnknown() {
		plan.Description = types.StringValue(al.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, fmt.Sprintf("Created annotation layer: ID=%d, Name=%s", al.ID, al.Name))
}

func (r *annotationLayerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting annotation layer Read")

	var state annotationLayerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	al, err := r.client.GetAnnotationLayer(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			tflog.Warn(ctx, "Annotation layer not found, removing from state",
				map[string]interface{}{"id": state.ID.ValueInt64()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading annotation layer",
			fmt.Sprintf("Could not read annotation layer ID %d: %s", state.ID.ValueInt64(), err.Error()))
		return
	}

	state.Name = types.StringValue(al.Name)
	state.Description = types.StringValue(al.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *annotationLayerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting annotation layer Update")

	var plan annotationLayerResourceModel
	var state annotationLayerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	description := ""
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description = plan.Description.ValueString()
	}

	al, err := r.client.UpdateAnnotationLayer(state.ID.ValueInt64(), plan.Name.ValueString(), description)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Superset Annotation Layer",
			fmt.Sprintf("UpdateAnnotationLayer failed: %s", err.Error()))
		return
	}

	state.Name = types.StringValue(al.Name)
	state.Description = types.StringValue(al.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, fmt.Sprintf("Updated annotation layer: ID=%d", state.ID.ValueInt64()))
}

func (r *annotationLayerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting annotation layer Delete")

	var state annotationLayerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAnnotationLayer(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Delete Superset Annotation Layer",
			fmt.Sprintf("DeleteAnnotationLayer failed: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Deleted annotation layer: ID=%d", state.ID.ValueInt64()))
}

func (r *annotationLayerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID",
			fmt.Sprintf("'%s' is not a valid int64: %s", req.ID, err.Error()))
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), id)
}
