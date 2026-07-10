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
	_ resource.Resource                = &datasetImportResource{}
	_ resource.ResourceWithConfigure   = &datasetImportResource{}
	_ resource.ResourceWithModifyPlan  = &datasetImportResource{}
	_ resource.ResourceWithImportState = &datasetImportResource{}
)

func NewDatasetImportResource() resource.Resource {
	return &datasetImportResource{}
}

type datasetImportResource struct {
	client *client.Client
}

type datasetImportResourceModel struct {
	ID                types.String `tfsdk:"id"`
	SourceDir         types.String `tfsdk:"source_dir"`
	ForceOverwrite    types.Bool   `tfsdk:"force_overwrite"`
	DatabaseSecrets   types.Map    `tfsdk:"database_secrets"`
	DatabaseOverrides types.Map    `tfsdk:"database_overrides"`
	FileHashes        types.Map    `tfsdk:"file_hashes"`
}

// Prefixes included in the dataset import ZIP.
var datasetImportPrefixes = []string{"datasets/", "databases/"}

func (r *datasetImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_import"
}

func (r *datasetImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports datasets from a dashboard export directory via POST /api/v1/dataset/import/. " +
			"This endpoint properly respects overwrite=true for datasets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this resource (derived from source_dir).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_dir": schema.StringAttribute{
				Description: "Path to a dashboard export directory containing datasets/, databases/, and metadata.yaml.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"force_overwrite": schema.BoolAttribute{
				Description: "Whether to overwrite existing datasets on import. Defaults to true.",
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

func (r *datasetImportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *datasetImportResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan datasetImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := plan.SourceDir.ValueString()
	if sourceDir == "" {
		return
	}

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)
	newHashes, err := computeFilteredFileHashes(sourceDir, datasetImportPrefixes, overrides)
	if err != nil {
		resp.Diagnostics.AddWarning("Cannot compute file hashes", err.Error())
		return
	}

	if req.State.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(newHashes))...)
		return
	}

	var state datasetImportResourceModel
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

func (r *datasetImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datasetImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.doImport(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to import datasets", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *datasetImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datasetImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *datasetImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan datasetImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state datasetImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if err := r.doImport(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to re-import datasets", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *datasetImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datasetImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := state.SourceDir.ValueString()

	// Read dataset UUIDs from the source directory and delete them from Superset
	uuids, err := readUUIDsFromDir(sourceDir, "datasets/")
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to read dataset UUIDs for deletion: %s", err))
		return
	}

	for _, uuid := range uuids {
		id, err := r.client.GetDatasetIDByUUID(uuid)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to look up dataset UUID %s: %s", uuid, err))
			continue
		}
		if id == 0 {
			continue
		}
		// Skip deletion if dataset is still used by charts
		chartCount, err := r.client.GetDatasetChartCount(id)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to check dataset %d references: %s", id, err))
		} else if chartCount > 0 {
			tflog.Info(ctx, fmt.Sprintf("Skipping dataset %d (UUID %s) — still referenced by %d chart(s)", id, uuid, chartCount))
			continue
		}
		tflog.Info(ctx, fmt.Sprintf("Deleting dataset %d (UUID %s)", id, uuid))
		if err := r.client.DeleteDataset(id); err != nil {
			resp.Diagnostics.AddWarning("Failed to delete dataset",
				fmt.Sprintf("Dataset %d (UUID %s): %s", id, uuid, err))
		}
	}
}

func (r *datasetImportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceDir := req.ID

	hashes, err := computeFilteredFileHashes(sourceDir, datasetImportPrefixes, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to compute file hashes", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("dataset-import:%s", sourceDir))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_dir"), sourceDir)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("force_overwrite"), true)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(hashes))...)
}

func (r *datasetImportResource) doImport(ctx context.Context, plan *datasetImportResourceModel) error {
	sourceDir := plan.SourceDir.ValueString()
	plan.ID = types.StringValue(fmt.Sprintf("dataset-import:%s", sourceDir))

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)

	hashes, err := computeFilteredFileHashes(sourceDir, datasetImportPrefixes, overrides)
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

	zipData, err := zipDirectoryFiltered(sourceDir, overrides, datasetImportPrefixes, "SqlaTable")
	if err != nil {
		return fmt.Errorf("creating ZIP: %w", err)
	}

	overwrite := plan.ForceOverwrite.ValueBool()
	tflog.Info(ctx, fmt.Sprintf("Importing datasets from %s (overwrite=%v)", sourceDir, overwrite))

	if err := r.client.ImportDataset(zipData, overwrite, passwords); err != nil {
		return err
	}

	return nil
}
