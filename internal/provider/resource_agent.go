package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type agentCoreModel struct {
	Id          types.String `tfsdk:"id"`
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

func (m *agentCoreModel) fill(a client.AgentResponse) {
	m.Id = types.StringValue(a.ID)
	m.Name = types.StringValue(a.Name)
	m.Model = types.StringValue(a.Model.ID)
	m.ModelSpeed = types.StringValue(a.Model.Speed)
	m.Version = types.Int64Value(int64(a.Version))
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)
	m.System = nullableString(a.System)
	m.Description = nullableString(a.Description)
	m.ArchivedAt = nullableString(a.ArchivedAt)
	m.Tools = marshalJSONList(a.Tools)
	m.MCPServers = marshalJSONList(a.MCPServers)
	m.Skills = marshalJSONList(a.Skills)
	if a.Multiagent != nil && string(*a.Multiagent) != "null" {
		m.Multiagent = types.StringValue(string(*a.Multiagent))
	} else {
		m.Multiagent = types.StringNull()
	}
	m.Metadata = fillMetadata(a.Metadata)
}

func buildAgentBody(data agentCoreModel) (map[string]any, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
		"model": map[string]string{
			"id":    data.Model.ValueString(),
			"speed": data.ModelSpeed.ValueString(),
		},
	}
	if !data.System.IsNull() && !data.System.IsUnknown() {
		body["system"] = data.System.ValueString()
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.Tools.IsNull() && !data.Tools.IsUnknown() {
		var tools []interface{}
		if err := json.Unmarshal([]byte(data.Tools.ValueString()), &tools); err != nil {
			return nil, fmt.Errorf("invalid tools JSON: %w", err)
		}
		body["tools"] = tools
	}
	if !data.MCPServers.IsNull() && !data.MCPServers.IsUnknown() {
		var mcpServers []interface{}
		if err := json.Unmarshal([]byte(data.MCPServers.ValueString()), &mcpServers); err != nil {
			return nil, fmt.Errorf("invalid mcp_servers JSON: %w", err)
		}
		body["mcp_servers"] = mcpServers
	}
	if !data.Skills.IsNull() && !data.Skills.IsUnknown() {
		var skills []interface{}
		if err := json.Unmarshal([]byte(data.Skills.ValueString()), &skills); err != nil {
			return nil, fmt.Errorf("invalid skills JSON: %w", err)
		}
		body["skills"] = skills
	}
	if !data.Multiagent.IsNull() && !data.Multiagent.IsUnknown() {
		var multiagent interface{}
		if err := json.Unmarshal([]byte(data.Multiagent.ValueString()), &multiagent); err != nil {
			return nil, fmt.Errorf("invalid multiagent JSON: %w", err)
		}
		body["multiagent"] = multiagent
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() && len(data.Metadata.Elements()) > 0 {
		meta := make(map[string]string, len(data.Metadata.Elements()))
		data.Metadata.ElementsAs(context.Background(), &meta, false)
		body["metadata"] = meta
	}
	return body, nil
}

func agentCoreSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{Required: true},
		"model": schema.StringAttribute{
			Required:    true,
			Description: "Model ID, e.g. claude-opus-4-7 or claude-sonnet-4-6.",
		},
		"model_speed": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("standard"),
			Description: "Inference speed: standard (default) or fast.",
		},
		"system":      schema.StringAttribute{Optional: true, Computed: true},
		"description": schema.StringAttribute{Optional: true, Computed: true},
		"tools": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description:   `JSON-encoded tools array. Example: [{"type":"agent_toolset_20260401"}]`,
		},
		"mcp_servers": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description:   `JSON-encoded MCP servers array. Example: [{"name":"my-server","type":"url","url":"https://..."}]. Maximum 20, names must be unique.`,
		},
		"skills": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description:   `JSON-encoded skills array. Example: [{"type":"anthropic","skill_id":"xlsx"}]. Maximum 20.`,
		},
		"multiagent": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description:   `JSON-encoded multiagent coordinator config. Example: {"type":"coordinator","agents":["agent_id_1","agent_id_2"]}.`,
		},
		"metadata": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Arbitrary string key-value pairs attached to the agent.",
		},
		"version": schema.Int64Attribute{
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"created_at": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"updated_at": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"archived_at": schema.StringAttribute{Computed: true},
	}
}

func agentUserFieldsChanged(plan, state agentCoreModel) bool {
	return !plan.Name.Equal(state.Name) ||
		!plan.System.Equal(state.System) ||
		!plan.Description.Equal(state.Description) ||
		!plan.Model.Equal(state.Model) ||
		!plan.ModelSpeed.Equal(state.ModelSpeed) ||
		!plan.Tools.Equal(state.Tools) ||
		!plan.MCPServers.Equal(state.MCPServers) ||
		!plan.Skills.Equal(state.Skills) ||
		!plan.Multiagent.Equal(state.Multiagent) ||
		!plan.Metadata.Equal(state.Metadata)
}

// AgentResource manages anthropic_agent using a workspace API key.
type AgentResource struct {
	data *providerData
}

func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}
var _ resource.ResourceWithModifyPlan = &AgentResource{}

func (r *AgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *AgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic agent using a workspace API key.",
		Attributes:  agentCoreSchemaAttrs(),
	}
}

func (r *AgentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan, state agentCoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !agentUserFieldsChanged(plan, state) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("version"), types.Int64Unknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())...)
}

func (r *AgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.data = data
}

func (r *AgentResource) requireWorkspaceKey(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.wsKey == "" {
		diags.AddError("Missing API key",
			"Set api_key in the provider block or ANTHROPIC_API_KEY environment variable. Required for anthropic_agent.")
		return false
	}
	return true
}

func (r *AgentResource) agentClient() *client.AgentClient {
	return client.NewAgentClient(auth.AdminAPIKey{Key: r.data.wsKey, Beta: auth.AgentsBeta})
}

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data agentCoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWorkspaceKey(&resp.Diagnostics) {
		return
	}

	body, err := buildAgentBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	agent, err := r.agentClient().Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data agentCoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWorkspaceKey(&resp.Diagnostics) {
		return
	}

	agent, err := r.agentClient().Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if agent == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data agentCoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state agentCoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWorkspaceKey(&resp.Diagnostics) {
		return
	}

	body, err := buildAgentBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	body["version"] = state.Version.ValueInt64()

	agent, err := r.agentClient().Update(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent: %s", err))
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data agentCoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWorkspaceKey(&resp.Diagnostics) {
		return
	}

	if err := r.agentClient().Delete(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive agent: %s", err))
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
