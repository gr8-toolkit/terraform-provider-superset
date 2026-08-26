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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &savedQueryResource{}
	_ resource.ResourceWithConfigure   = &savedQueryResource{}
	_ resource.ResourceWithImportState = &savedQueryResource{}
)

// NewSavedQueryResource is a helper function to simplify the provider implementation.
func NewSavedQueryResource() resource.Resource {
	return &savedQueryResource{}
}

type savedQueryResource struct {
	client *client.Client
}

type savedQueryResourceModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	DatabaseID         types.Int64  `tfsdk:"database_id"`
	DatabaseName       types.String `tfsdk:"database_name"`
	Label              types.String `tfsdk:"label"`
	Description        types.String `tfsdk:"description"`
	Catalog            types.String `tfsdk:"catalog"`
	Schema             types.String `tfsdk:"schema"`
	SQL                types.String `tfsdk:"sql"`
	TemplateParameters types.String `tfsdk:"template_parameters"`
	ExtraJSON          types.String `tfsdk:"extra_json"`
}

func (r *savedQueryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_query"
}

func (r *savedQueryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a saved SQL query in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the saved query.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.Int64Attribute{
				Description: "ID of the database this query belongs to.",
				Required:    true,
			},
			"database_name": schema.StringAttribute{
				Description: "Name of the database this query belongs to (read from API).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"label": schema.StringAttribute{
				Description: "Display name of the saved query.",
				Required:    true,
			},
			"sql": schema.StringAttribute{
				Description: "SQL text of the query.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the saved query.",
				Optional:    true,
				Computed:    true, // API returns "" when not set.
			},
			"catalog": schema.StringAttribute{
				Description: "Optional catalog name for the query.",
				Optional:    true,
				Computed:    true, // API returns "" when not set.
			},
			"schema": schema.StringAttribute{
				Description: "Optional schema name for the query.",
				Optional:    true,
				Computed:    true, // API returns "" when not set.
			},
			"template_parameters": schema.StringAttribute{
				Description: "JSON string of Jinja template parameters used in the SQL.",
				Optional:    true,
				Computed:    true,
			},
			"extra_json": schema.StringAttribute{
				Description: "JSON string of extra metadata for the saved query.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *savedQueryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildSavedQueryPayload converts the model to the API wire format.
// The API uses "db_id" not "database_id".
func buildSavedQueryPayload(plan savedQueryResourceModel) map[string]interface{} {
	payload := map[string]interface{}{
		"db_id": plan.DatabaseID.ValueInt64(),
		"label": plan.Label.ValueString(),
		"sql":   plan.SQL.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		payload["description"] = plan.Description.ValueString()
	}
	if !plan.Catalog.IsNull() && !plan.Catalog.IsUnknown() {
		payload["catalog"] = plan.Catalog.ValueString()
	}
	if !plan.Schema.IsNull() && !plan.Schema.IsUnknown() {
		payload["schema"] = plan.Schema.ValueString()
	}
	if !plan.TemplateParameters.IsNull() && !plan.TemplateParameters.IsUnknown() {
		payload["template_parameters"] = plan.TemplateParameters.ValueString()
	}
	if !plan.ExtraJSON.IsNull() && !plan.ExtraJSON.IsUnknown() {
		payload["extra_json"] = plan.ExtraJSON.ValueString()
	}
	return payload
}

// applySavedQueryToState writes API response fields back to the model.
func applySavedQueryToState(sq *client.SavedQuery, state *savedQueryResourceModel) {
	state.DatabaseID = types.Int64Value(sq.DatabaseID)
	state.DatabaseName = types.StringValue(sq.DatabaseName)
	state.Label = types.StringValue(sq.Label)
	state.SQL = types.StringValue(sq.SQL)
	state.Description = types.StringValue(sq.Description)
	state.Catalog = types.StringValue(sq.Catalog)
	state.Schema = types.StringValue(sq.Schema)
	state.TemplateParameters = types.StringValue(sq.TemplateParameters)
	state.ExtraJSON = types.StringValue(sq.ExtraJSON)
}

func (r *savedQueryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting saved query Create")

	var plan savedQueryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sq, err := r.client.CreateSavedQuery(buildSavedQueryPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Superset Saved Query",
			fmt.Sprintf("CreateSavedQuery failed: %s", err.Error()))
		return
	}

	plan.ID = types.Int64Value(sq.ID)

	// POST response does not include the nested database object (database_name).
	// Re-fetch via GET to get the full record.
	full, err := r.client.GetSavedQuery(sq.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Saved Query After Create",
			fmt.Sprintf("GetSavedQuery failed: %s", err.Error()))
		return
	}
	applySavedQueryToState(full, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, fmt.Sprintf("Created saved query: ID=%d, Label=%s", sq.ID, sq.Label))
}

func (r *savedQueryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting saved query Read")

	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sq, err := r.client.GetSavedQuery(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			tflog.Warn(ctx, "Saved query not found, removing from state",
				map[string]interface{}{"id": state.ID.ValueInt64()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading saved query",
			fmt.Sprintf("Could not read saved query ID %d: %s", state.ID.ValueInt64(), err.Error()))
		return
	}

	applySavedQueryToState(sq, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *savedQueryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting saved query Update")

	var plan savedQueryResourceModel
	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateSavedQuery(state.ID.ValueInt64(), buildSavedQueryPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Superset Saved Query",
			fmt.Sprintf("UpdateSavedQuery failed: %s", err.Error()))
		return
	}

	// PUT response does not include the nested database object — re-fetch.
	full, err := r.client.GetSavedQuery(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Saved Query After Update",
			fmt.Sprintf("GetSavedQuery failed: %s", err.Error()))
		return
	}
	applySavedQueryToState(full, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, fmt.Sprintf("Updated saved query: ID=%d", state.ID.ValueInt64()))
}

func (r *savedQueryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting saved query Delete")

	var state savedQueryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSavedQuery(state.ID.ValueInt64())
	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Delete Superset Saved Query",
			fmt.Sprintf("DeleteSavedQuery failed: %s", err.Error()))
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Deleted saved query: ID=%d", state.ID.ValueInt64()))
}

func (r *savedQueryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID",
			fmt.Sprintf("'%s' is not a valid int64: %s", req.ID, err.Error()))
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), id)
}
