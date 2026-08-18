package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-superset/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &cssTemplateResource{}
	_ resource.ResourceWithConfigure   = &cssTemplateResource{}
	_ resource.ResourceWithImportState = &cssTemplateResource{}
)

// NewCSSTemplateResource is a helper function to simplify the provider implementation.
func NewCSSTemplateResource() resource.Resource {
	return &cssTemplateResource{}
}

// cssTemplateResource is the resource implementation.
type cssTemplateResource struct {
	client *client.Client
}

// cssTemplateResourceModel maps the resource schema data.
type cssTemplateResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	CSS          types.String `tfsdk:"css"`
}

// notWhitespaceOnlyValidator validates that a string is not composed entirely of whitespace.
type notWhitespaceOnlyValidator struct{}

func (v notWhitespaceOnlyValidator) Description(_ context.Context) string {
	return "value must not be empty or contain only whitespace characters"
}

func (v notWhitespaceOnlyValidator) MarkdownDescription(_ context.Context) string {
	return "value must not be empty or contain only whitespace characters"
}

func (v notWhitespaceOnlyValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if strings.TrimSpace(value) == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			"Value must not be empty or contain only whitespace characters.",
		)
	}
}

// stringLengthBetweenValidator validates that a string has length between min and max.
type stringLengthBetweenValidator struct {
	min int
	max int
}

func (v stringLengthBetweenValidator) Description(_ context.Context) string {
	return fmt.Sprintf("string length must be between %d and %d", v.min, v.max)
}

func (v stringLengthBetweenValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("string length must be between %d and %d", v.min, v.max)
}

func (v stringLengthBetweenValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if len(value) < v.min || len(value) > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid String Length",
			fmt.Sprintf("String length must be between %d and %d, got: %d.", v.min, v.max, len(value)),
		)
	}
}

// stringLengthAtLeastValidator validates that a string has at least min characters.
type stringLengthAtLeastValidator struct {
	min int
}

func (v stringLengthAtLeastValidator) Description(_ context.Context) string {
	return fmt.Sprintf("string length must be at least %d", v.min)
}

func (v stringLengthAtLeastValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("string length must be at least %d", v.min)
}

func (v stringLengthAtLeastValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if len(value) < v.min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid String Length",
			fmt.Sprintf("String length must be at least %d, got: %d.", v.min, len(value)),
		)
	}
}

// Metadata returns the resource type name.
func (r *cssTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_css_template"
}

// Schema defines the schema for the resource.
func (r *cssTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CSS template in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the CSS template (stored as string).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
				Description: "Name of the CSS template (1-255 characters, must not be whitespace-only).",
				Required:    true,
				Validators: []validator.String{
					stringLengthBetweenValidator{min: 1, max: 255},
					notWhitespaceOnlyValidator{},
				},
			},
			"css": schema.StringAttribute{
				Description: "CSS content of the template (must not be empty or whitespace-only).",
				Required:    true,
				Validators: []validator.String{
					stringLengthAtLeastValidator{min: 1},
					notWhitespaceOnlyValidator{},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *cssTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting CSS template Create method")
	var plan cssTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check for existing templates with the same name
	existing, err := r.client.FindCSSTemplatesByName(plan.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Check CSS Template Uniqueness",
			fmt.Sprintf("Failed to check for existing CSS templates: %s", err.Error()),
		)
		return
	}
	if len(existing) > 0 {
		resp.Diagnostics.AddError(
			"CSS Template Name Already Exists",
			fmt.Sprintf("A CSS template with name %q already exists (ID: %d). Template names must be unique.", plan.TemplateName.ValueString(), existing[0].ID),
		)
		return
	}

	tmpl, err := r.client.CreateCSSTemplate(plan.TemplateName.ValueString(), plan.CSS.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset CSS Template",
			fmt.Sprintf("CreateCSSTemplate failed: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(tmpl.ID))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created CSS template: ID=%s, Name=%s", plan.ID.ValueString(), plan.TemplateName.ValueString()))
}

// Read refreshes the Terraform state with the latest data from Superset.
func (r *cssTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting CSS template Read method")
	var state cssTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid CSS Template ID",
			fmt.Sprintf("Could not parse ID '%s' as integer: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tmpl, err := r.client.GetCSSTemplate(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			tflog.Info(ctx, fmt.Sprintf("CSS template ID %d not found, removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading CSS template",
			fmt.Sprintf("Could not read CSS template ID %d: %s", id, err.Error()),
		)
		return
	}

	state.TemplateName = types.StringValue(tmpl.TemplateName)
	state.CSS = types.StringValue(tmpl.CSS)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *cssTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting CSS template Update method")
	var plan cssTemplateResourceModel
	var state cssTemplateResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid CSS Template ID",
			fmt.Sprintf("Could not parse ID '%s' as integer: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tmpl, err := r.client.UpdateCSSTemplate(id, plan.TemplateName.ValueString(), plan.CSS.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			tflog.Info(ctx, fmt.Sprintf("CSS template ID %d not found during update, removing from state", id))
			resp.State.RemoveResource(ctx)
			resp.Diagnostics.AddError(
				"CSS Template Not Found",
				fmt.Sprintf("CSS template ID %d no longer exists in Superset.", id),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Update Superset CSS Template",
			fmt.Sprintf("UpdateCSSTemplate failed: %s", err.Error()),
		)
		return
	}

	state.TemplateName = types.StringValue(tmpl.TemplateName)
	state.CSS = types.StringValue(tmpl.CSS)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updated CSS template: ID=%s, Name=%s", state.ID.ValueString(), state.TemplateName.ValueString()))
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *cssTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting CSS template Delete method")
	var state cssTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid CSS Template ID",
			fmt.Sprintf("Could not parse ID '%s' as integer: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	err = r.client.DeleteCSSTemplate(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset CSS Template",
			fmt.Sprintf("DeleteCSSTemplate failed: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Deleted CSS template: ID=%d", id))
}

// ImportState imports an existing resource.
func (r *cssTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Starting CSS template ImportState method", map[string]interface{}{
		"import_id": req.ID,
	})

	// Validate that the import ID is a valid integer
	_, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("The provided import ID '%s' is not a valid integer: %s", req.ID, err.Error()),
		)
		return
	}

	// Set the ID as a string in the state; the framework will call Read to populate the rest
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)

	tflog.Debug(ctx, "CSS template ImportState completed successfully", map[string]interface{}{
		"import_id": req.ID,
	})
}

// Configure adds the provider configured client to the resource.
func (r *cssTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}
