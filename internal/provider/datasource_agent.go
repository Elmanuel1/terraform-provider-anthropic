package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

func (m *AgentDataModel) fill(a client.AgentResponse) {
	var core AgentCoreModel
	core.fill(a)
	m.Id = core.Id
	m.Name = core.Name
	m.Model = core.Model
	m.ModelSpeed = core.ModelSpeed
	m.System = core.System
	m.Description = core.Description
	m.Tools = core.Tools
	m.MCPServers = core.MCPServers
	m.Skills = core.Skills
	m.Multiagent = core.Multiagent
	m.Metadata = core.Metadata
	m.Version = core.Version
	m.CreatedAt = core.CreatedAt
	m.UpdatedAt = core.UpdatedAt
	m.ArchivedAt = core.ArchivedAt
	// WorkspaceId is caller-supplied; not present in AgentResponse — leave unchanged.
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
	// All fields mirror agentCoreSchemaAttrs() but use datasource/schema types
	// (plan modifiers are not applicable to data sources) and id/workspace_id differ.
	resp.Schema = dsschema.Schema{
		Description: "Reads an existing Anthropic agent by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{Required: true, Description: "Agent ID (agt_...)."},
			"workspace_id": dsschema.StringAttribute{Optional: true, Description: "Workspace ID. Required when using WIF authentication."},
			"name":         dsschema.StringAttribute{Computed: true},
			"model":        dsschema.StringAttribute{Computed: true, Description: "Model ID, e.g. claude-opus-4-7 or claude-sonnet-4-6."},
			"model_speed":  dsschema.StringAttribute{Computed: true, Description: "Inference speed: standard or fast."},
			"system":       dsschema.StringAttribute{Computed: true},
			"description":  dsschema.StringAttribute{Computed: true},
			"tools":        dsschema.StringAttribute{Computed: true, Description: `JSON-encoded tools array. Example: [{"type":"agent_toolset_20260401"}]`},
			"mcp_servers":  dsschema.StringAttribute{Computed: true, Description: `JSON-encoded MCP servers array. Example: [{"name":"my-server","type":"url","url":"https://..."}].`},
			"skills":       dsschema.StringAttribute{Computed: true, Description: `JSON-encoded skills array. Example: [{"type":"anthropic","skill_id":"xlsx"}].`},
			"multiagent":   dsschema.StringAttribute{Computed: true, Description: `JSON-encoded multiagent coordinator config. Example: {"type":"coordinator","agents":["agent_id_1","agent_id_2"]}.`},
			"metadata":     dsschema.MapAttribute{Computed: true, ElementType: types.StringType, Description: "Arbitrary string key-value pairs attached to the agent."},
			"version":      dsschema.Int64Attribute{Computed: true},
			"created_at":   dsschema.StringAttribute{Computed: true},
			"updated_at":   dsschema.StringAttribute{Computed: true},
			"archived_at":  dsschema.StringAttribute{Computed: true},
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

	data.fill(*a)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
