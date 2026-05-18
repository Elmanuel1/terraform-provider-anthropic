package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentDataSource struct {
	data *providerData
}

type AgentDataModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Name        types.String `tfsdk:"name"`
	Model       types.String `tfsdk:"model"`
	ModelSpeed  types.String `tfsdk:"model_speed"`
	System      types.String `tfsdk:"system"`
	Description types.String `tfsdk:"description"`
	Tools       types.String `tfsdk:"tools"`
	MCPServers  types.String `tfsdk:"mcp_servers"`
	Skills      types.String `tfsdk:"skills"`
	Multiagent  types.String `tfsdk:"multiagent"`
	Metadata    types.Map    `tfsdk:"metadata"`
	Version     types.Int64  `tfsdk:"version"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func NewAgentDataSource() datasource.DataSource {
	return &AgentDataSource{}
}

var _ datasource.DataSource = &AgentDataSource{}
var _ datasource.DataSourceWithConfigure = &AgentDataSource{}

func (d *AgentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *AgentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic agent by ID.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Required: true, Description: "Agent ID (agt_...)."},
			"workspace_id": schema.StringAttribute{Optional: true, Description: "Workspace ID. Required when using WIF authentication."},
			"name":         schema.StringAttribute{Computed: true},
			"model":        schema.StringAttribute{Computed: true},
			"model_speed":  schema.StringAttribute{Computed: true},
			"system":       schema.StringAttribute{Computed: true},
			"description":  schema.StringAttribute{Computed: true},
			"tools":        schema.StringAttribute{Computed: true, Description: "JSON-encoded tools array."},
			"mcp_servers":  schema.StringAttribute{Computed: true, Description: "JSON-encoded MCP servers array."},
			"skills":       schema.StringAttribute{Computed: true, Description: "JSON-encoded skills array."},
			"multiagent":   schema.StringAttribute{Computed: true, Description: "JSON-encoded multiagent config."},
			"metadata":     schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"version":      schema.Int64Attribute{Computed: true},
			"created_at":   schema.StringAttribute{Computed: true},
			"updated_at":   schema.StringAttribute{Computed: true},
			"archived_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *AgentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.data = pd
}

func (d *AgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_agent", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := client.NewAgentClient(creds).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if a == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Agent %q not found.", data.Id.ValueString()))
		return
	}

	var m AgentCoreModel
	m.fill(*a)
	data.Name = m.Name
	data.Model = m.Model
	data.ModelSpeed = m.ModelSpeed
	data.System = m.System
	data.Description = m.Description
	data.Tools = m.Tools
	data.MCPServers = m.MCPServers
	data.Skills = m.Skills
	data.Multiagent = m.Multiagent
	data.Metadata = m.Metadata
	data.Version = m.Version
	data.CreatedAt = m.CreatedAt
	data.UpdatedAt = m.UpdatedAt
	data.ArchivedAt = m.ArchivedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
