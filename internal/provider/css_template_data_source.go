package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-superset/internal/client"
)

var (
	_ datasource.DataSource              = &cssTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &cssTemplateDataSource{}
)

func NewCSSTemplateDataSource() datasource.DataSource {
	return &cssTemplateDataSource{}
}

type cssTemplateDataSource struct {
	client *client.Client
}

type cssTemplateDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	ID           types.Int64  `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	CSS          types.String `tfsdk:"css"`
}

func (d *cssTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_css_template"
}

func (d *cssTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a CSS template by name from Superset.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the CSS template to look up (exact, case-sensitive).",
				Required:    true,
			},
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the CSS template.",
				Computed:    true,
			},
			"template_name": schema.StringAttribute{
				Description: "Name of the CSS template.",
				Computed:    true,
			},
			"css": schema.StringAttribute{
				Description: "CSS content of the template.",
				Computed:    true,
			},
		},
	}
}

func (d *cssTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cssTemplateDataSourceModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	tmpl, err := d.client.FindCSSTemplateByName(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Superset CSS Template",
			fmt.Sprintf("no CSS template with name '%s' found: %s", name, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(int64(tmpl.ID))
	data.TemplateName = types.StringValue(tmpl.TemplateName)
	data.CSS = types.StringValue(tmpl.CSS)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d *cssTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}
