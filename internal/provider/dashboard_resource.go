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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
)

// NewDashboardResource is a helper function to simplify the provider implementation.
func NewDashboardResource() resource.Resource {
	return &dashboardResource{}
}

type dashboardResource struct {
	client *client.Client
}

type dashboardResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	DashboardTitle types.String `tfsdk:"dashboard_title"`
	Slug           types.String `tfsdk:"slug"`
	CSS            types.String `tfsdk:"css"`
	Published      types.Bool   `tfsdk:"published"`
	PositionJSON   types.String `tfsdk:"position_json"`
	JSONMetadata   types.String `tfsdk:"json_metadata"`
	URL            types.String `tfsdk:"url"`
}

func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard"
}

func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a dashboard in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the dashboard.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "UUID of the dashboard assigned by Superset.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dashboard_title": schema.StringAttribute{
				Description: "Title of the dashboard.",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "Optional URL-friendly slug for the dashboard.",
				Optional:    true,
				Computed:    true, // API returns "" when not set; Computed prevents null→"" inconsistency.
			},
			"css": schema.StringAttribute{
				Description: "Optional custom CSS applied to the dashboard.",
				Optional:    true,
				Computed:    true, // API returns "" when not set; Computed prevents null→"" inconsistency.
			},
			"published": schema.BoolAttribute{
				Description: "Whether the dashboard is published (visible to all). Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"position_json": schema.StringAttribute{
				Description: "JSON string describing the dashboard grid layout.",
				Optional:    true,
				Computed:    true,
			},
			"json_metadata": schema.StringAttribute{
				Description: "JSON string of dashboard metadata (native filters, filter bar settings, etc.).",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL path to the dashboard in the Superset UI.",
				Computed:    true,
			},
		},
	}
}

func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildDashboardCreatePayload builds the POST payload (subset of fields).
func buildDashboardCreatePayload(plan dashboardResourceModel) map[string]interface{} {
	payload := map[string]interface{}{
		"dashboard_title": plan.DashboardTitle.ValueString(),
		"published":       plan.Published.ValueBool(),
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		payload["slug"] = plan.Slug.ValueString()
	}
	if !plan.CSS.IsNull() && !plan.CSS.IsUnknown() {
		payload["css"] = plan.CSS.ValueString()
	}
	if !plan.PositionJSON.IsNull() && !plan.PositionJSON.IsUnknown() && plan.PositionJSON.ValueString() != "" {
		payload["position_json"] = plan.PositionJSON.ValueString()
	}
	if !plan.JSONMetadata.IsNull() && !plan.JSONMetadata.IsUnknown() && plan.JSONMetadata.ValueString() != "" {
		payload["json_metadata"] = plan.JSONMetadata.ValueString()
	}
	return payload
}

// applyDashboardToState writes API response fields back into the state model.
func applyDashboardToState(d *client.Dashboard, state *dashboardResourceModel) {
	state.UUID = types.StringValue(d.UUID)
	state.DashboardTitle = types.StringValue(d.DashboardTitle)
	state.Slug = types.StringValue(d.Slug)
	state.CSS = types.StringValue(d.CSS)
	state.Published = types.BoolValue(d.Published)
	state.PositionJSON = types.StringValue(d.PositionJSON)
	state.JSONMetadata = types.StringValue(d.JSONMetadata)
	state.URL = types.StringValue(d.URL)
}

func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting dashboard Create")

	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	d, err := r.client.CreateDashboard(buildDashboardCreatePayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Superset Dashboard",
			fmt.Sprintf("CreateDashboard failed: %s", err.Error()))
		return
	}

	plan.ID = types.Int64Value(d.ID)

	// POST response only returns a partial payload (no uuid, url, etc.) — re-fetch.
	full, err := r.client.GetDashboard(d.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Dashboard After Create",
			fmt.Sprintf("GetDashboard failed: %s", err.Error()))
		return
	}
	applyDashboardToState(full, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, fmt.Sprintf("Created dashboard: ID=%d, Title=%s", d.ID, d.DashboardTitle))
}

func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting dashboard Read")

	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	d, err := r.client.GetDashboard(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			tflog.Warn(ctx, "Dashboard not found, removing from state",
				map[string]interface{}{"id": state.ID.ValueInt64()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading dashboard",
			fmt.Sprintf("Could not read dashboard ID %d: %s", state.ID.ValueInt64(), err.Error()))
		return
	}

	applyDashboardToState(d, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting dashboard Update")

	var plan dashboardResourceModel
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateDashboard(state.ID.ValueInt64(), buildDashboardCreatePayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Superset Dashboard",
			fmt.Sprintf("UpdateDashboard failed: %s", err.Error()))
		return
	}

	// PUT response is also a partial payload — re-fetch the full record.
	full, err := r.client.GetDashboard(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Dashboard After Update",
			fmt.Sprintf("GetDashboard failed: %s", err.Error()))
		return
	}
	applyDashboardToState(full, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, fmt.Sprintf("Updated dashboard: ID=%d", state.ID.ValueInt64()))
}

func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting dashboard Delete")

	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDashboardByID(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Delete Superset Dashboard",
			fmt.Sprintf("DeleteDashboard failed: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Deleted dashboard: ID=%d", state.ID.ValueInt64()))
}

func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID",
			fmt.Sprintf("'%s' is not a valid int64: %s", req.ID, err.Error()))
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), id)
}
