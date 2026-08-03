package provider

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &databaseResource{}
	_ resource.ResourceWithConfigure   = &databaseResource{}
	_ resource.ResourceWithImportState = &databaseResource{}
)

// NewDatabaseResource is a helper function to simplify the provider implementation.
func NewDatabaseResource() resource.Resource {
	return &databaseResource{}
}

// databaseResource is the resource implementation.
type databaseResource struct {
	client *client.Client
}

// sshTunnelModel maps the ssh_tunnel nested block.
type sshTunnelModel struct {
	ServerAddress      types.String `tfsdk:"server_address"`
	ServerPort         types.Int64  `tfsdk:"server_port"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	PrivateKey         types.String `tfsdk:"private_key"`
	PrivateKeyPassword types.String `tfsdk:"private_key_password"`
}

// databaseResourceModel maps the resource schema data.
type databaseResourceModel struct {
	ID                   types.Int64     `tfsdk:"id"`
	ConnectionName       types.String    `tfsdk:"connection_name"`
	DBEngine             types.String    `tfsdk:"db_engine"`
	SQLAlchemyURI        types.String    `tfsdk:"sqlalchemy_uri"`
	DBUser               types.String    `tfsdk:"db_user"`
	DBPass               types.String    `tfsdk:"db_pass"`
	DBHost               types.String    `tfsdk:"db_host"`
	DBPort               types.Int64     `tfsdk:"db_port"`
	DBName               types.String    `tfsdk:"db_name"`
	AllowCTAS            types.Bool      `tfsdk:"allow_ctas"`
	AllowCVAS            types.Bool      `tfsdk:"allow_cvas"`
	AllowDML             types.Bool      `tfsdk:"allow_dml"`
	AllowFileUpload      types.Bool      `tfsdk:"allow_file_upload"`
	AllowRunAsync        types.Bool      `tfsdk:"allow_run_async"`
	ExposeInSQLLab       types.Bool      `tfsdk:"expose_in_sqllab"`
	CacheTimeout         types.Int64     `tfsdk:"cache_timeout"`
	Extra                types.String    `tfsdk:"extra"`
	ForceCTASSchema      types.String    `tfsdk:"force_ctas_schema"`
	ImpersonateUser      types.Bool      `tfsdk:"impersonate_user"`
	MaskedEncryptedExtra types.String    `tfsdk:"masked_encrypted_extra"`
	ServerCert           types.String    `tfsdk:"server_cert"`
	SSHTunnel            *sshTunnelModel `tfsdk:"ssh_tunnel"`
}

// Metadata returns the resource type name.
func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the resource.
func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database connection in Superset.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the database connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"connection_name": schema.StringAttribute{
				Description: "Name of the database connection.",
				Required:    true,
			},
			"db_engine": schema.StringAttribute{
				Description: "Database engine (e.g., postgresql, mysql, awsathena).",
				Required:    true,
			},
			"sqlalchemy_uri": schema.StringAttribute{
				Description: "Full SQLAlchemy URI for the database connection. When set, db_user/db_pass/db_host/db_port/db_name are ignored. " +
					"Use this for engines like Athena: awsathena+rest://{access_key}:{secret_key}@athena.{region}.amazonaws.com/{schema}?s3_staging_dir={s3_staging_dir}",
				Optional:  true,
				Sensitive: true,
			},
			"db_user": schema.StringAttribute{
				Description: "Database username. Not required when sqlalchemy_uri is set.",
				Optional:    true,
			},
			"db_pass": schema.StringAttribute{
				Description: "Database password. Not required when sqlalchemy_uri is set.",
				Optional:    true,
				Sensitive:   true,
			},
			"db_host": schema.StringAttribute{
				Description: "Database host. Not required when sqlalchemy_uri is set.",
				Optional:    true,
			},
			"db_port": schema.Int64Attribute{
				Description: "Database port. Not required when sqlalchemy_uri is set.",
				Optional:    true,
			},
			"db_name": schema.StringAttribute{
				Description: "Database name. Not required when sqlalchemy_uri is set.",
				Optional:    true,
			},
			"allow_ctas": schema.BoolAttribute{
				Description: "Allow CTAS.",
				Required:    true,
			},
			"allow_cvas": schema.BoolAttribute{
				Description: "Allow CVAS.",
				Required:    true,
			},
			"allow_dml": schema.BoolAttribute{
				Description: "Allow DML.",
				Required:    true,
			},
			"allow_file_upload": schema.BoolAttribute{
				Description: "Allow file (CSV/Excel/Columnar) upload into this database. If selected, set the schemas allowed for file upload in Extra.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"allow_run_async": schema.BoolAttribute{
				Description: "Allow run async.",
				Required:    true,
			},
			"expose_in_sqllab": schema.BoolAttribute{
				Description: "Expose in SQL Lab.",
				Required:    true,
			},
			"cache_timeout": schema.Int64Attribute{
				Description: "Duration (in seconds) of the caching timeout for charts of this database. A timeout of 0 indicates that the cache never expires, and -1 falls back to the default timeout.",
				Optional:    true,
				Computed:    true,
			},
			"extra": schema.StringAttribute{
				Description: "JSON string containing extra configuration elements. Supports client_encoding, cost_estimate_enabled, schemas_allowed_for_csv_upload, etc.",
				Optional:    true,
				Computed:    true,
			},
			"force_ctas_schema": schema.StringAttribute{
				Description: "When using CTAS, the default target schema (where the table is created) if not defined in the SQL query.",
				Optional:    true,
			},
			"impersonate_user": schema.BoolAttribute{
				Description: "If enabled, the connection string is impersonated using the name of the logged-in user.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"masked_encrypted_extra": schema.StringAttribute{
				Description: "JSON string containing additional connection configuration, such as credentials for OAuth2 or GCP service accounts. Sensitive fields are masked.",
				Optional:    true,
				Sensitive:   true,
			},
			"server_cert": schema.StringAttribute{
				Description: "Optional CA_BUNDLE contents to validate HTTPS requests. Only available on certain database engines.",
				Optional:    true,
			},
			"ssh_tunnel": schema.SingleNestedAttribute{
				Description: "SSH tunnel configuration for connecting to the database through a bastion host.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"server_address": schema.StringAttribute{
						Description: "SSH tunnel server address (hostname or IP).",
						Required:    true,
					},
					"server_port": schema.Int64Attribute{
						Description: "SSH tunnel server port.",
						Required:    true,
					},
					"username": schema.StringAttribute{
						Description: "SSH tunnel username.",
						Required:    true,
					},
					"password": schema.StringAttribute{
						Description: "SSH tunnel password. Mutually exclusive with private_key.",
						Optional:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: "SSH tunnel private key (PEM format). Mutually exclusive with password.",
						Optional:    true,
						Sensitive:   true,
					},
					"private_key_password": schema.StringAttribute{
						Description: "Passphrase for the SSH tunnel private key.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Starting Create method")
	var plan databaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Create due to error in retrieving plan", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	sqlalchemyURI := buildSQLAlchemyURI(plan)
	payload := buildDatabasePayload(plan, sqlalchemyURI)

	result, err := r.client.CreateDatabase(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Database Connection",
			fmt.Sprintf("CreateDatabase failed: %s", err.Error()),
		)
		return
	}

	// Type assertion with error handling
	idFloat, ok := result["id"].(float64)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The 'id' field in the response is not a float64",
		)
		return
	}
	plan.ID = types.Int64Value(int64(idFloat))

	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain the expected 'result' field",
		)
		return
	}

	// Handle type assertions with error handling
	if val, ok := resultData["database_name"].(string); ok {
		plan.ConnectionName = types.StringValue(val)
	} else {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain a valid 'database_name' field",
		)
		return
	}
	if val, ok := resultData["allow_ctas"].(bool); ok {
		plan.AllowCTAS = types.BoolValue(val)
	}
	if val, ok := resultData["allow_cvas"].(bool); ok {
		plan.AllowCVAS = types.BoolValue(val)
	}
	if val, ok := resultData["allow_dml"].(bool); ok {
		plan.AllowDML = types.BoolValue(val)
	}
	if val, ok := resultData["allow_file_upload"].(bool); ok {
		plan.AllowFileUpload = types.BoolValue(val)
	}
	if val, ok := resultData["allow_run_async"].(bool); ok {
		plan.AllowRunAsync = types.BoolValue(val)
	}
	if val, ok := resultData["expose_in_sqllab"].(bool); ok {
		plan.ExposeInSQLLab = types.BoolValue(val)
	}
	if val, ok := resultData["cache_timeout"].(float64); ok {
		plan.CacheTimeout = types.Int64Value(int64(val))
	} else if plan.CacheTimeout.IsUnknown() {
		plan.CacheTimeout = types.Int64Value(0)
	}
	if val, ok := resultData["extra"].(string); ok {
		plan.Extra = types.StringValue(val)
	} else if plan.Extra.IsUnknown() {
		plan.Extra = types.StringValue("")
	}
	if val, ok := resultData["impersonate_user"].(bool); ok {
		plan.ImpersonateUser = types.BoolValue(val)
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Create due to error in setting state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created database connection: ID=%d, ConnectionName=%s", plan.ID.ValueInt64(), plan.ConnectionName.ValueString()))
}

// Read refreshes the Terraform state with the latest data from Superset.
func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Starting Read method")
	var state databaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Read due to error in getting state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	db, err := r.client.GetDatabaseConnectionByID(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading database connection",
			fmt.Sprintf("Could not read database ID %d: %s", state.ID.ValueInt64(), err.Error()),
		)
		return
	}

	result, ok := db["result"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain the expected 'result' field",
		)
		return
	}

	if val, ok := result["database_name"].(string); ok {
		state.ConnectionName = types.StringValue(val)
	} else {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain a valid 'database_name' field",
		)
		return
	}
	if val, ok := result["allow_ctas"].(bool); ok {
		state.AllowCTAS = types.BoolValue(val)
	}
	if val, ok := result["allow_cvas"].(bool); ok {
		state.AllowCVAS = types.BoolValue(val)
	}
	if val, ok := result["allow_dml"].(bool); ok {
		state.AllowDML = types.BoolValue(val)
	}
	if val, ok := result["allow_file_upload"].(bool); ok {
		state.AllowFileUpload = types.BoolValue(val)
	}
	if val, ok := result["allow_run_async"].(bool); ok {
		state.AllowRunAsync = types.BoolValue(val)
	}
	if val, ok := result["expose_in_sqllab"].(bool); ok {
		state.ExposeInSQLLab = types.BoolValue(val)
	}
	if val, ok := result["cache_timeout"].(float64); ok {
		state.CacheTimeout = types.Int64Value(int64(val))
	} else if state.CacheTimeout.IsUnknown() {
		state.CacheTimeout = types.Int64Value(0)
	}
	if val, ok := result["extra"].(string); ok {
		state.Extra = types.StringValue(val)
	} else if state.Extra.IsUnknown() {
		state.Extra = types.StringValue("")
	}
	if val, ok := result["impersonate_user"].(bool); ok {
		state.ImpersonateUser = types.BoolValue(val)
	}
	if val, ok := result["server_cert"].(string); ok {
		state.ServerCert = types.StringValue(val)
	}
	if val, ok := result["backend"].(string); ok {
		state.DBEngine = types.StringValue(val)
	}
	// Only populate individual fields if sqlalchemy_uri is not used
	if state.SQLAlchemyURI.IsNull() || state.SQLAlchemyURI.ValueString() == "" {
		if params, ok := result["parameters"].(map[string]interface{}); ok {
			if val, ok := params["host"].(string); ok {
				state.DBHost = types.StringValue(val)
			}
			if val, ok := params["username"].(string); ok {
				state.DBUser = types.StringValue(val)
			}
			if val, ok := params["port"].(float64); ok {
				state.DBPort = types.Int64Value(int64(val))
			}
			if val, ok := params["database"].(string); ok {
				state.DBName = types.StringValue(val)
			}
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Read due to error in setting state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting Update method")
	var plan databaseResourceModel
	var state databaseResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Update due to error in retrieving plan", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Update due to error in retrieving state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	sqlalchemyURI := buildSQLAlchemyURI(plan)
	payload := buildDatabasePayload(plan, sqlalchemyURI)

	result, err := r.client.UpdateDatabase(state.ID.ValueInt64(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Superset Database Connection",
			fmt.Sprintf("UpdateDatabase failed: %s", err.Error()),
		)
		return
	}

	resultData, ok := result["result"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain the expected 'result' field",
		)
		return
	}

	// Update state attributes with the values from the response
	if val, ok := resultData["database_name"].(string); ok {
		state.ConnectionName = types.StringValue(val)
	} else {
		resp.Diagnostics.AddError(
			"Invalid Response",
			"The response from the API does not contain a valid 'database_name' field",
		)
		return
	}
	if val, ok := resultData["allow_ctas"].(bool); ok {
		state.AllowCTAS = types.BoolValue(val)
	}
	if val, ok := resultData["allow_cvas"].(bool); ok {
		state.AllowCVAS = types.BoolValue(val)
	}
	if val, ok := resultData["allow_dml"].(bool); ok {
		state.AllowDML = types.BoolValue(val)
	}
	if val, ok := resultData["allow_file_upload"].(bool); ok {
		state.AllowFileUpload = types.BoolValue(val)
	}
	if val, ok := resultData["allow_run_async"].(bool); ok {
		state.AllowRunAsync = types.BoolValue(val)
	}
	if val, ok := resultData["expose_in_sqllab"].(bool); ok {
		state.ExposeInSQLLab = types.BoolValue(val)
	}
	if val, ok := resultData["cache_timeout"].(float64); ok {
		state.CacheTimeout = types.Int64Value(int64(val))
	} else if state.CacheTimeout.IsUnknown() {
		state.CacheTimeout = types.Int64Value(0)
	}
	if val, ok := resultData["extra"].(string); ok {
		state.Extra = types.StringValue(val)
	} else if state.Extra.IsUnknown() {
		state.Extra = types.StringValue("")
	}
	if val, ok := resultData["impersonate_user"].(bool); ok {
		state.ImpersonateUser = types.BoolValue(val)
	}

	state.DBEngine = plan.DBEngine
	state.SQLAlchemyURI = plan.SQLAlchemyURI
	state.DBUser = plan.DBUser
	state.DBPass = plan.DBPass
	state.DBHost = plan.DBHost
	state.DBPort = plan.DBPort
	state.DBName = plan.DBName
	state.ForceCTASSchema = plan.ForceCTASSchema
	state.MaskedEncryptedExtra = plan.MaskedEncryptedExtra
	state.ServerCert = plan.ServerCert
	state.SSHTunnel = plan.SSHTunnel

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Update due to error in setting state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Updated database connection: ID=%d, ConnectionName=%s", state.ID.ValueInt64(), state.ConnectionName.ValueString()))
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Starting Delete method")
	var state databaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "Exiting Delete due to error in getting state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	err := r.client.DeleteDatabase(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Superset Database Connection",
			fmt.Sprintf("DeleteDatabase failed: %s", err.Error()),
		)
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, fmt.Sprintf("Deleted database connection: ID=%d", state.ID.ValueInt64()))
}

// ImportState imports an existing resource.
func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Starting ImportState method", map[string]interface{}{
		"import_id": req.ID,
	})

	// Convert import ID to int64 and set it to the state
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("The provided import ID '%s' is not a valid int64: %s", req.ID, err.Error()),
		)
		return
	}

	// Set the ID in the state and call Read
	resp.State.SetAttribute(ctx, path.Root("id"), id)

	// Call Read to refresh the state with the latest data
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{
		State:       resp.State,
		Diagnostics: resp.Diagnostics,
	})

	tflog.Debug(ctx, "ImportState completed successfully", map[string]interface{}{
		"import_id": req.ID,
	})
}

// buildSQLAlchemyURI returns the SQLAlchemy URI from either the explicit field or constructed from parts.
func buildSQLAlchemyURI(plan databaseResourceModel) string {
	if !plan.SQLAlchemyURI.IsNull() && plan.SQLAlchemyURI.ValueString() != "" {
		return plan.SQLAlchemyURI.ValueString()
	}
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		plan.DBEngine.ValueString(), plan.DBUser.ValueString(), plan.DBPass.ValueString(),
		plan.DBHost.ValueString(), plan.DBPort.ValueInt64(), plan.DBName.ValueString())
}

// buildDatabasePayload constructs the API payload for create/update.
func buildDatabasePayload(plan databaseResourceModel, sqlalchemyURI string) map[string]interface{} {
	payload := map[string]interface{}{
		"allow_ctas":                        plan.AllowCTAS.ValueBool(),
		"allow_cvas":                        plan.AllowCVAS.ValueBool(),
		"allow_dml":                         plan.AllowDML.ValueBool(),
		"allow_file_upload":                 plan.AllowFileUpload.ValueBool(),
		"allow_multi_schema_metadata_fetch": true,
		"allow_run_async":                   plan.AllowRunAsync.ValueBool(),
		"database_name":                     plan.ConnectionName.ValueString(),
		"expose_in_sqllab":                  plan.ExposeInSQLLab.ValueBool(),
		"impersonate_user":                  plan.ImpersonateUser.ValueBool(),
		"sqlalchemy_uri":                    sqlalchemyURI,
	}

	if !plan.CacheTimeout.IsNull() && !plan.CacheTimeout.IsUnknown() {
		payload["cache_timeout"] = plan.CacheTimeout.ValueInt64()
	}

	if !plan.Extra.IsNull() && !plan.Extra.IsUnknown() {
		payload["extra"] = plan.Extra.ValueString()
	}

	if !plan.ForceCTASSchema.IsNull() && !plan.ForceCTASSchema.IsUnknown() {
		payload["force_ctas_schema"] = plan.ForceCTASSchema.ValueString()
	}

	if !plan.MaskedEncryptedExtra.IsNull() && !plan.MaskedEncryptedExtra.IsUnknown() {
		payload["masked_encrypted_extra"] = plan.MaskedEncryptedExtra.ValueString()
	}

	if !plan.ServerCert.IsNull() && !plan.ServerCert.IsUnknown() {
		payload["server_cert"] = plan.ServerCert.ValueString()
	}

	if plan.SSHTunnel != nil {
		tunnel := map[string]interface{}{
			"server_address": plan.SSHTunnel.ServerAddress.ValueString(),
			"server_port":    plan.SSHTunnel.ServerPort.ValueInt64(),
			"username":       plan.SSHTunnel.Username.ValueString(),
		}
		if !plan.SSHTunnel.Password.IsNull() && !plan.SSHTunnel.Password.IsUnknown() {
			tunnel["password"] = plan.SSHTunnel.Password.ValueString()
		}
		if !plan.SSHTunnel.PrivateKey.IsNull() && !plan.SSHTunnel.PrivateKey.IsUnknown() {
			tunnel["private_key"] = plan.SSHTunnel.PrivateKey.ValueString()
		}
		if !plan.SSHTunnel.PrivateKeyPassword.IsNull() && !plan.SSHTunnel.PrivateKeyPassword.IsUnknown() {
			tunnel["private_key_password"] = plan.SSHTunnel.PrivateKeyPassword.ValueString()
		}
		payload["ssh_tunnel"] = tunnel
	}

	return payload
}

// Configure adds the provider configured client to the resource.
func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
