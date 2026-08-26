// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &chartResource{}
	_ resource.ResourceWithConfigure   = &chartResource{}
	_ resource.ResourceWithImportState = &chartResource{}
)

// NewChartResource is a helper function to simplify the provider implementation.
func NewChartResource() resource.Resource {
	return &chartResource{}
}

type chartResource struct {
	client *client.Client
}

type chartResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	SliceName      types.String `tfsdk:"slice_name"`
	Description    types.String `tfsdk:"description"`
	VizType        types.String `tfsdk:"viz_type"`
	DatasourceID   types.Int64  `tfsdk:"datasource_id"`
	DatasourceType types.String `tfsdk:"datasource_type"`
	DatasourceName types.String `tfsdk:"datasource_name"`
	Params         types.String `tfsdk:"params"`
	QueryContext   types.String `tfsdk:"query_context"`
	CacheTimeout   types.Int64  `tfsdk:"cache_timeout"`
	URL            types.String `tfsdk:"url"`
}

func (r *chartResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart"
}

func (r *chartResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a chart in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the chart.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "UUID of the chart assigned by Superset.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slice_name": schema.StringAttribute{
				Description: "Display name of the chart.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the chart.",
				Optional:    true,
			},
			"viz_type": schema.StringAttribute{
				Description: "Visualisation type (e.g. 'table', 'bar', 'line').",
				Required:    true,
			},
			"datasource_id": schema.Int64Attribute{
				Description: "ID of the dataset this chart is built on.",
				Required:    true,
			},
			"datasource_type": schema.StringAttribute{
				Description: "Type of the datasource. Defaults to 'table'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("table"),
			},
			"datasource_name": schema.StringAttribute{
				Description: "Name of the datasource (read from API).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"params": schema.StringAttribute{
				Description: "JSON string of chart form data / configuration.",
				Required:    true,
			},
			"query_context": schema.StringAttribute{
				Description: "JSON string of the query context. Optional — when omitted Superset auto-generates it.",
				Optional:    true,
				Computed:    true,
			},
			"cache_timeout": schema.Int64Attribute{
				Description: "Cache timeout in seconds. 0 means no cache expiry; -1 falls back to the datasource default.",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL path to the chart in the Superset UI.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *chartResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildChartPayload converts a plan model into the Superset API payload.
func buildChartPayload(plan chartResourceModel) map[string]interface{} {
	payload := map[string]interface{}{
		"slice_name":      plan.SliceName.ValueString(),
		"viz_type":        plan.VizType.ValueString(),
		"datasource_id":   plan.DatasourceID.ValueInt64(),
		"datasource_type": plan.DatasourceType.ValueString(),
		"params":          plan.Params.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		payload["description"] = plan.Description.ValueString()
	}
	if !plan.QueryContext.IsNull() && !plan.QueryContext.IsUnknown() && plan.QueryContext.ValueString() != "" {
		payload["query_context"] = plan.QueryContext.ValueString()
	}
	if !plan.CacheTimeout.IsNull() && !plan.CacheTimeout.IsUnknown() {
		payload["cache_timeout"] = plan.CacheTimeout.ValueInt64()
	}
	return payload
}

// applyChartToState writes API response fields back into the state model.
func applyChartToState(ch *client.Chart, state *chartResourceModel) {
	state.UUID = types.StringValue(ch.UUID)
	state.SliceName = types.StringValue(ch.SliceName)
	state.Description = types.StringValue(ch.Description)
	state.VizType = types.StringValue(ch.VizType)
	state.DatasourceID = types.Int64Value(ch.DatasourceID)
	state.DatasourceType = types.StringValue(ch.DatasourceType)
	state.DatasourceName = types.StringValue(ch.DatasourceName)
	state.Params = types.StringValue(ch.Params)
	state.QueryContext = types.StringValue(ch.QueryContext)
	if ch.CacheTimeout != nil {
		state.CacheTimeout = types.Int64Value(*ch.CacheTimeout)
	} else if state.CacheTimeout.IsUnknown() {
		state.CacheTimeout = types.Int64Value(0)
	}
	state.URL = types.StringValue(ch.URL)
}

func (r *chartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting chart Create")

	var plan chartResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.client.CreateChart(buildChartPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Superset Chart",
			fmt.Sprintf("CreateChart failed: %s", err.Error()))
		return
	}

	plan.ID = types.Int64Value(ch.ID)

	// The POST response only returns a subset of fields (no uuid, url, datasource_name, etc.).
	// Do a GET to populate the full state.
	full, err := r.client.GetChart(ch.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Chart After Create",
			fmt.Sprintf("GetChart failed: %s", err.Error()))
		return
	}
	applyChartToState(full, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, fmt.Sprintf("Created chart: ID=%d, Name=%s", ch.ID, ch.SliceName))
}

func (r *chartResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting chart Read")

	var state chartResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := r.client.GetChart(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			tflog.Warn(ctx, "Chart not found, removing from state",
				map[string]interface{}{"id": state.ID.ValueInt64()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading chart",
			fmt.Sprintf("Could not read chart ID %d: %s", state.ID.ValueInt64(), err.Error()))
		return
	}

	applyChartToState(ch, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *chartResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting chart Update")

	var plan chartResourceModel
	var state chartResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateChart(state.ID.ValueInt64(), buildChartPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Superset Chart",
			fmt.Sprintf("UpdateChart failed: %s", err.Error()))
		return
	}

	// PUT response is also a partial payload — re-fetch the full record.
	full, err := r.client.GetChart(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Chart After Update",
			fmt.Sprintf("GetChart failed: %s", err.Error()))
		return
	}
	applyChartToState(full, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, fmt.Sprintf("Updated chart: ID=%d", state.ID.ValueInt64()))
}

func (r *chartResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting chart Delete")

	var state chartResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteChart(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Delete Superset Chart",
			fmt.Sprintf("DeleteChart failed: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Deleted chart: ID=%d", state.ID.ValueInt64()))
}

func (r *chartResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID",
			fmt.Sprintf("'%s' is not a valid int64: %s", req.ID, err.Error()))
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), id)
}
