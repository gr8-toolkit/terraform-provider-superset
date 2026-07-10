package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &chartImportResource{}
	_ resource.ResourceWithConfigure   = &chartImportResource{}
	_ resource.ResourceWithModifyPlan  = &chartImportResource{}
	_ resource.ResourceWithImportState = &chartImportResource{}
)

func NewChartImportResource() resource.Resource {
	return &chartImportResource{}
}

type chartImportResource struct {
	client *client.Client
}

type chartImportResourceModel struct {
	ID                types.String `tfsdk:"id"`
	SourceDir         types.String `tfsdk:"source_dir"`
	ForceOverwrite    types.Bool   `tfsdk:"force_overwrite"`
	DatabaseSecrets   types.Map    `tfsdk:"database_secrets"`
	DatabaseOverrides types.Map    `tfsdk:"database_overrides"`
	FileHashes        types.Map    `tfsdk:"file_hashes"`
}

// Prefixes included in the chart import ZIP.
var chartImportPrefixes = []string{"charts/", "datasets/", "databases/"}

func (r *chartImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chart_import"
}

func (r *chartImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports charts from a dashboard export directory via POST /api/v1/chart/import/. " +
			"This endpoint properly respects overwrite=true for charts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this resource (derived from source_dir).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_dir": schema.StringAttribute{
				Description: "Path to a dashboard export directory containing charts/, datasets/, databases/, and metadata.yaml.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"force_overwrite": schema.BoolAttribute{
				Description: "Whether to overwrite existing charts on import. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"database_secrets": schema.MapAttribute{
				Description: "Map of database UUID to database password/secret.",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"database_overrides": schema.MapAttribute{
				Description: "Map of database UUID to a JSON-encoded object of YAML field overrides.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"file_hashes": schema.MapAttribute{
				Description: "Map of file path to SHA256 hash. Changes trigger re-import.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *chartImportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *chartImportResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan chartImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := plan.SourceDir.ValueString()
	if sourceDir == "" {
		return
	}

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)
	newHashes, err := computeFilteredFileHashes(sourceDir, chartImportPrefixes, overrides)
	if err != nil {
		resp.Diagnostics.AddWarning("Cannot compute file hashes", err.Error())
		return
	}

	if req.State.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(newHashes))...)
		return
	}

	var state chartImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldHashes := fromStringMap(state.FileHashes)
	if !mapsEqual(oldHashes, newHashes) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(newHashes))...)
	} else {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), state.FileHashes)...)
	}
}

func (r *chartImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan chartImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.doImport(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to import charts", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *chartImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state chartImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *chartImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan chartImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state chartImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if err := r.doImport(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to re-import charts", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *chartImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state chartImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := state.SourceDir.ValueString()

	// Find the dashboard ID from this source_dir to know which dashboard we're unlinking from
	var ownerDashboardID int64
	dashUUIDs, _ := readUUIDsFromDir(sourceDir, "dashboards/")
	if len(dashUUIDs) > 0 {
		ownerDashboardID, _ = r.client.GetDashboardIDByUUID(dashUUIDs[0])
	}

	// Read chart UUIDs from the source directory
	uuids, err := readUUIDsFromDir(sourceDir, "charts/")
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to read chart UUIDs for deletion: %s", err))
		return
	}

	for _, uuid := range uuids {
		id, err := r.client.GetChartIDByUUID(uuid)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to look up chart UUID %s: %s", uuid, err))
			continue
		}
		if id == 0 {
			continue
		}

		// Check how many dashboards reference this chart
		dashCount, err := r.client.GetChartDashboardCount(id)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to check chart %d references: %s", id, err))
			// Can't determine — try to delete anyway
			dashCount = 0
		}

		// Always unlink from this resource's dashboard first
		if ownerDashboardID > 0 && dashCount > 0 {
			tflog.Info(ctx, fmt.Sprintf("Unlinking chart %d (UUID %s) from dashboard %d", id, uuid, ownerDashboardID))
			if err := r.client.UnlinkChartsFromDashboard([]int64{id}, ownerDashboardID); err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to unlink chart %d from dashboard %d: %s", id, ownerDashboardID, err))
			}
		}

		// Re-check: if no dashboards reference it now, delete it
		newDashCount, err := r.client.GetChartDashboardCount(id)
		if err != nil {
			newDashCount = 0 // assume safe to delete
		}
		if newDashCount == 0 {
			tflog.Info(ctx, fmt.Sprintf("Deleting chart %d (UUID %s) — no longer referenced", id, uuid))
			if err := r.client.DeleteChart(id); err != nil {
				resp.Diagnostics.AddWarning("Failed to delete chart",
					fmt.Sprintf("Chart %d (UUID %s): %s", id, uuid, err))
			}
		} else {
			tflog.Info(ctx, fmt.Sprintf("Chart %d (UUID %s) still referenced by %d dashboard(s), keeping", id, uuid, newDashCount))
		}
	}
}

func (r *chartImportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceDir := req.ID

	hashes, err := computeFilteredFileHashes(sourceDir, chartImportPrefixes, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to compute file hashes", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("chart-import:%s", sourceDir))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_dir"), sourceDir)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("force_overwrite"), true)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(hashes))...)
}

func (r *chartImportResource) doImport(ctx context.Context, plan *chartImportResourceModel) error {
	sourceDir := plan.SourceDir.ValueString()
	plan.ID = types.StringValue(fmt.Sprintf("chart-import:%s", sourceDir))

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)

	hashes, err := computeFilteredFileHashes(sourceDir, chartImportPrefixes, overrides)
	if err != nil {
		return fmt.Errorf("computing file hashes: %w", err)
	}
	plan.FileHashes = toStringMap(hashes)

	secrets := make(map[string]string)
	if !plan.DatabaseSecrets.IsNull() && !plan.DatabaseSecrets.IsUnknown() {
		diags := plan.DatabaseSecrets.ElementsAs(ctx, &secrets, false)
		if diags.HasError() {
			return fmt.Errorf("reading database_secrets")
		}
	}
	passwordMap, err := buildPasswordMap(sourceDir, secrets)
	if err != nil {
		return fmt.Errorf("building password map: %w", err)
	}
	passwords := ""
	if len(passwordMap) > 0 {
		b, _ := json.Marshal(passwordMap)
		passwords = string(b)
	}

	zipData, err := zipDirectoryFiltered(sourceDir, overrides, chartImportPrefixes, "Slice")
	if err != nil {
		return fmt.Errorf("creating ZIP: %w", err)
	}

	overwrite := plan.ForceOverwrite.ValueBool()
	tflog.Info(ctx, fmt.Sprintf("Importing charts from %s (overwrite=%v)", sourceDir, overwrite))

	if err := r.client.ImportChart(zipData, overwrite, passwords); err != nil {
		return err
	}

	return nil
}
