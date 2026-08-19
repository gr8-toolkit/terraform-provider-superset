package provider

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

var (
	_ resource.Resource               = &dashboardImportResource{}
	_ resource.ResourceWithConfigure  = &dashboardImportResource{}
	_ resource.ResourceWithModifyPlan = &dashboardImportResource{}
)

func NewDashboardImportResource() resource.Resource {
	return &dashboardImportResource{}
}

type dashboardImportResource struct {
	client *client.Client
}

type dashboardImportResourceModel struct {
	ID                types.String `tfsdk:"id"`
	SourceDir         types.String `tfsdk:"source_dir"`
	ForceOverwrite    types.Bool   `tfsdk:"force_overwrite"`
	DatabaseSecrets   types.Map    `tfsdk:"database_secrets"`
	DatabaseOverrides types.Map    `tfsdk:"database_overrides"`
	FileHashes        types.Map    `tfsdk:"file_hashes"`
	DashboardID       types.Int64  `tfsdk:"dashboard_id"`
	Roles             types.List   `tfsdk:"roles"`
	SkipFiles         types.List   `tfsdk:"skip_files"`
	CSSOverride       types.String `tfsdk:"css_override"`
}

func (r *dashboardImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dashboard_import"
}

func (r *dashboardImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports a Superset dashboard from an export directory (the result of dashboard export).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Dashboard UUID from the export.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_dir": schema.StringAttribute{
				Description: "Path to the dashboard export directory containing metadata.yaml, dashboards/, charts/, databases/, datasets/ etc.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"force_overwrite": schema.BoolAttribute{
				Description: "Whether to overwrite existing dashboards on import. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"database_secrets": schema.MapAttribute{
				Description: "Map of database UUID to database password/secret. Used to provide credentials for databases referenced in the export.",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"database_overrides": schema.MapAttribute{
				Description: "Map of database UUID to a JSON-encoded object of YAML field overrides. " +
					"Allows overriding any fields (including nested) in database export files before import. " +
					"Example: {\"<uuid>\" = jsonencode({sqlalchemy_uri = \"...\", extra = {cost_estimate_enabled = false}})}",
				Optional:    true,
				ElementType: types.StringType,
			},
			"file_hashes": schema.MapAttribute{
				Description: "Map of relative file path to SHA256 hash. Changes to individual files trigger re-import.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"dashboard_id": schema.Int64Attribute{
				Description: "Numeric ID of the imported dashboard in Superset.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"roles": schema.ListAttribute{
				Description: "List of role IDs to assign to the dashboard. Applied after every import.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"skip_files": schema.ListAttribute{
				Description: "List of regex patterns to exclude files from hashing and import. " +
					"Matched against both the file name and relative path.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"css_override": schema.StringAttribute{
				Description: "CSS content to override the inline css field in the dashboard YAML before import. " +
					"When set, replaces the css field in the dashboard export file.",
				Optional: true,
			},
		},
	}
}

func (r *dashboardImportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dashboardImportResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan dashboardImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := plan.SourceDir.ValueString()
	if sourceDir == "" {
		return
	}

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)
	skipFilePatterns := parseSkipFiles(ctx, plan.SkipFiles)
	skipPatterns := compileSkipPatterns(skipFilePatterns)
	if len(skipFilePatterns) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("skip_files patterns configured: %v", skipFilePatterns))
	}

	cssOverride := ""
	if !plan.CSSOverride.IsNull() && !plan.CSSOverride.IsUnknown() {
		cssOverride = plan.CSSOverride.ValueString()
	}

	newHashes, err := computeFileHashesWithOverrides(sourceDir, overrides, skipPatterns, cssOverride)
	if err != nil {
		resp.Diagnostics.AddWarning("Cannot compute file hashes", err.Error())
		return
	}

	// On create (no prior state), always set hashes
	if req.State.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(newHashes))...)
		return
	}

	// Compare against state — only touch the plan if files actually changed
	var state dashboardImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldHashes := fromStringMap(state.FileHashes)
	changed := !mapsEqual(oldHashes, newHashes)

	if changed {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), toStringMap(newHashes))...)
	} else {
		// No changes — preserve state values in plan
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_hashes"), state.FileHashes)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("dashboard_id"), state.DashboardID)...)
	}
}

func (r *dashboardImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dashboardImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.importDashboard(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to import dashboard", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dashboardImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.DashboardID.IsNull() && !state.DashboardID.IsUnknown() {
		exists, err := r.client.DashboardExistsByID(state.DashboardID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Failed to check dashboard existence", err.Error())
			return
		}
		if !exists {
			tflog.Warn(ctx, fmt.Sprintf("Dashboard ID %d not found, removing from state", state.DashboardID.ValueInt64()))
			resp.State.RemoveResource(ctx)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dashboardImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dashboardImportResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dashboardImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	plan.DashboardID = state.DashboardID

	if err := r.importDashboard(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to re-import dashboard", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dashboardImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dashboardImportResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.DashboardID.IsNull() && !state.DashboardID.IsUnknown() {
		if err := r.client.DeleteDashboard(state.DashboardID.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Failed to delete dashboard", err.Error())
			return
		}
	}
}

type dashboardExportMeta struct {
	UUID  string `yaml:"uuid"`
	Slug  string `yaml:"slug"`
	Title string `yaml:"dashboard_title"`
}

func (r *dashboardImportResource) importDashboard(ctx context.Context, plan *dashboardImportResourceModel) error {
	sourceDir := plan.SourceDir.ValueString()

	meta, err := readDashboardMeta(sourceDir)
	if err != nil {
		return fmt.Errorf("reading dashboard metadata: %w", err)
	}
	plan.ID = types.StringValue(meta.UUID)

	overrides := parseDatabaseOverrides(ctx, plan.DatabaseOverrides)
	skipFilePatterns := parseSkipFiles(ctx, plan.SkipFiles)
	skipPatterns := compileSkipPatterns(skipFilePatterns)
	if len(skipFilePatterns) > 0 {
		tflog.Info(ctx, fmt.Sprintf("Skipping files matching patterns: %v", skipFilePatterns))
	}

	cssOverride := ""
	if !plan.CSSOverride.IsNull() && !plan.CSSOverride.IsUnknown() {
		cssOverride = plan.CSSOverride.ValueString()
	}

	fileHashes, err := computeFileHashesWithOverrides(sourceDir, overrides, skipPatterns, cssOverride)
	if err != nil {
		return fmt.Errorf("computing file hashes: %w", err)
	}
	plan.FileHashes = toStringMap(fileHashes)

	zipData, err := zipDirectoryWithOverrides(sourceDir, overrides, skipPatterns, cssOverride)
	if err != nil {
		return fmt.Errorf("creating ZIP: %w", err)
	}

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

	overwrite := plan.ForceOverwrite.ValueBool()
	tflog.Info(ctx, fmt.Sprintf("Importing dashboard from %s (overwrite=%v)", sourceDir, overwrite))

	// If dashboard already exists, unlink all charts and clear layout before importing
	existingID := plan.DashboardID.ValueInt64()
	if existingID == 0 {
		existingID, _ = r.client.GetDashboardIDByUUID(meta.UUID)
	}
	if existingID > 0 {
		// Get all chart IDs on the dashboard
		chartUUIDMap, err := r.client.GetDashboardChartUUIDs(existingID)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to get dashboard chart UUIDs: %s", err))
		} else {
			var allChartIDs []int64
			for _, chartID := range chartUUIDMap {
				allChartIDs = append(allChartIDs, chartID)
			}
			// Unlink all charts from dashboard
			if len(allChartIDs) > 0 {
				if err := r.client.UnlinkChartsFromDashboard(allChartIDs, existingID); err != nil {
					tflog.Warn(ctx, fmt.Sprintf("Failed to unlink charts: %s", err))
				}
			}
		}
		// Clear position_json and json_metadata
		if err := r.client.ClearDashboardLayout(existingID); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to clear dashboard layout: %s", err))
		}
		tflog.Info(ctx, fmt.Sprintf("Cleared all charts and layout from dashboard %d", existingID))
	}

	// Import dashboard
	if err := r.client.ImportDashboard(zipData, overwrite, passwords); err != nil {
		return err
	}

	var dashID int64
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		dashID, err = r.client.GetDashboardIDByUUID(meta.UUID)
		if err == nil {
			break
		}
		tflog.Debug(ctx, fmt.Sprintf("Dashboard lookup attempt %d failed: %s", attempt+1, err))
	}
	if err != nil {
		return fmt.Errorf("dashboard imported but could not find it after retries: %w", err)
	}
	plan.DashboardID = types.Int64Value(dashID)

	// Apply roles if configured
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roleIDs []int64
		diags := plan.Roles.ElementsAs(ctx, &roleIDs, false)
		if diags.HasError() {
			return fmt.Errorf("reading roles")
		}
		if err := r.client.SetDashboardRoles(dashID, roleIDs); err != nil {
			return fmt.Errorf("setting dashboard roles: %w", err)
		}
	}

	return nil
}

// --- helpers ---

func readDashboardMeta(sourceDir string) (*dashboardExportMeta, error) {
	dashDir := filepath.Join(sourceDir, "dashboards")
	entries, err := os.ReadDir(dashDir)
	if err != nil {
		return nil, fmt.Errorf("reading dashboards directory: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dashDir, e.Name()))
		if err != nil {
			return nil, err
		}
		var meta dashboardExportMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return nil, err
		}
		if meta.UUID != "" {
			return &meta, nil
		}
	}
	return nil, fmt.Errorf("no dashboard YAML found in %s", dashDir)
}

func buildPasswordMap(sourceDir string, secrets map[string]string) (map[string]string, error) {
	dbDir := filepath.Join(sourceDir, "databases")
	entries, err := os.ReadDir(dbDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	result := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		key := "databases/" + e.Name()
		result[key] = ""
		data, err := os.ReadFile(filepath.Join(dbDir, e.Name()))
		if err != nil {
			return nil, err
		}
		var db struct {
			UUID string `yaml:"uuid"`
		}
		if err := yaml.Unmarshal(data, &db); err != nil {
			continue
		}
		if pw, ok := secrets[db.UUID]; ok {
			result[key] = pw
		}
	}
	return result, nil
}

// toStringMap converts map[string]string to types.Map.
func toStringMap(m map[string]string) types.Map {
	elements := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elements[k] = types.StringValue(v)
	}
	result, _ := types.MapValue(types.StringType, elements)
	return result
}

// fromStringMap extracts map[string]string from types.Map.
func fromStringMap(m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return map[string]string{}
	}
	result := make(map[string]string)
	for k, v := range m.Elements() {
		if sv, ok := v.(types.String); ok {
			result[k] = sv.ValueString()
		}
	}
	return result
}

// mapsEqual returns true if two string maps have identical keys and values.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// parseDatabaseOverrides extracts database UUID -> arbitrary override map from JSON strings.
func parseDatabaseOverrides(ctx context.Context, m types.Map) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	if m.IsNull() || m.IsUnknown() {
		return result
	}
	raw := make(map[string]string)
	if diags := m.ElementsAs(ctx, &raw, false); diags.HasError() {
		return result
	}
	for uuid, jsonStr := range raw {
		var fields map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
			continue
		}
		if len(fields) > 0 {
			result[uuid] = fields
		}
	}
	return result
}

// parseSkipFiles extracts a list of skip file patterns from a types.List attribute.
func parseSkipFiles(ctx context.Context, l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var patterns []string
	if diags := l.ElementsAs(ctx, &patterns, false); diags.HasError() {
		return nil
	}
	return patterns
}

// deepMerge recursively merges src into dst. Values in src override dst.
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for k, srcVal := range src {
		if dstVal, ok := dst[k]; ok {
			if dstMap, ok := dstVal.(map[string]interface{}); ok {
				if srcMap, ok := srcVal.(map[string]interface{}); ok {
					dst[k] = deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		dst[k] = srcVal
	}
	return dst
}

// applyDatabaseOverrides patches a database YAML file content by deep-merging overrides.
// It matches by the "uuid" field in the YAML.
func applyDatabaseOverrides(data []byte, overrides map[string]map[string]interface{}) ([]byte, error) {
	if len(overrides) == 0 {
		return data, nil
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return data, err
	}

	uuid, _ := doc["uuid"].(string)
	if uuid == "" {
		return data, nil
	}

	fields, ok := overrides[uuid]
	if !ok {
		return data, nil
	}

	doc = deepMerge(doc, fields)

	out, err := yaml.Marshal(doc)
	if err != nil {
		return data, err
	}
	return out, nil
}

// computeFileHashesWithOverrides computes SHA256 hashes for all files in dir,
// applying database overrides to databases/*.yaml files before hashing.
// If cssOverride is non-empty, it replaces the css field in dashboards/*.yaml files.
// Files matching skipPatterns are excluded.
func computeFileHashesWithOverrides(dir string, overrides map[string]map[string]interface{}, skipPatterns []*regexp.Regexp, cssOverride string) (map[string]string, error) {
	hashes := make(map[string]string)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		// Skip files matching user-provided patterns
		if shouldSkipFile(d.Name(), rel, skipPatterns) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, "databases/") && strings.HasSuffix(rel, ".yaml") {
			data, _ = applyDatabaseOverrides(data, overrides)
		}
		if strings.HasPrefix(rel, "dashboards/") && strings.HasSuffix(rel, ".yaml") && cssOverride != "" {
			data, _ = applyCSSOverride(data, cssOverride)
		}
		h := sha256.Sum256(data)
		hashes[rel] = fmt.Sprintf("%x", h)
		return nil
	})
	return hashes, err
}

// zipDirectoryWithOverrides creates a ZIP of sourceDir, applying database overrides to databases/*.yaml.
// If cssOverride is non-empty, it replaces the css field in dashboards/*.yaml files.
// Files matching skipPatterns are excluded.
func zipDirectoryWithOverrides(sourceDir string, overrides map[string]map[string]interface{}, skipPatterns []*regexp.Regexp, cssOverride string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	base := filepath.Base(sourceDir)
	err := filepath.WalkDir(sourceDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, p)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		zipPath := filepath.ToSlash(filepath.Join(base, rel))
		if d.IsDir() {
			_, err := w.Create(zipPath + "/")
			return err
		}
		// Skip files matching user-provided patterns
		if shouldSkipFile(d.Name(), relSlash, skipPatterns) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relSlash, "databases/") && strings.HasSuffix(relSlash, ".yaml") {
			data, _ = applyDatabaseOverrides(data, overrides)
		}
		if strings.HasPrefix(relSlash, "dashboards/") && strings.HasSuffix(relSlash, ".yaml") && cssOverride != "" {
			data, _ = applyCSSOverride(data, cssOverride)
		}
		f, err := w.Create(zipPath)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// applyCSSOverride replaces the css field in a dashboard YAML file with the override value.
func applyCSSOverride(data []byte, cssOverride string) ([]byte, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return data, err
	}

	doc["css"] = cssOverride

	out, err := yaml.Marshal(doc)
	if err != nil {
		return data, err
	}
	return out, nil
}
