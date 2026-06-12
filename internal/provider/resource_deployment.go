package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type deploymentAgentModel struct {
	Id      types.String `tfsdk:"id"`
	Version types.Int64  `tfsdk:"version"`
}

type deploymentContentBlockModel struct {
	Type types.String `tfsdk:"type"`
	Text types.String `tfsdk:"text"`
}

type deploymentInitialEventModel struct {
	Type    types.String                  `tfsdk:"type"`
	Content []deploymentContentBlockModel `tfsdk:"content"`
}

type deploymentScheduleModel struct {
	Expression     types.String `tfsdk:"expression"`
	Timezone       types.String `tfsdk:"timezone"`
	LastRunAt      types.String `tfsdk:"last_run_at"`
	UpcomingRunsAt types.List   `tfsdk:"upcoming_runs_at"`
}

// pausedReasonAttrTypes describes the paused_reason object. It is modeled as a
// types.Object (not a Go struct pointer) because the attribute is fully Computed
// and therefore unknown at plan time, which a struct pointer cannot represent.
var pausedReasonAttrTypes = map[string]attr.Type{
	"type":       types.StringType,
	"error_type": types.StringType,
}

type DeploymentModel struct {
	Id            types.String                  `tfsdk:"id"`
	WorkspaceId   types.String                  `tfsdk:"workspace_id"`
	Name          types.String                  `tfsdk:"name"`
	Description   types.String                  `tfsdk:"description"`
	Agent         *deploymentAgentModel         `tfsdk:"agent"`
	EnvironmentId types.String                  `tfsdk:"environment_id"`
	InitialEvents []deploymentInitialEventModel `tfsdk:"initial_events"`
	Schedule      *deploymentScheduleModel      `tfsdk:"schedule"`
	VaultIds      types.List                    `tfsdk:"vault_ids"`
	Metadata      types.Map                     `tfsdk:"metadata"`
	Paused        types.Bool                    `tfsdk:"paused"`
	Status        types.String                  `tfsdk:"status"`
	PausedReason  types.Object                  `tfsdk:"paused_reason"`
	CreatedAt     types.String                  `tfsdk:"created_at"`
	UpdatedAt     types.String                  `tfsdk:"updated_at"`
	ArchivedAt    types.String                  `tfsdk:"archived_at"`
}

func (m *DeploymentModel) fill(d client.DeploymentResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Name)
	m.Description = nullableString(d.Description)
	m.EnvironmentId = types.StringValue(d.EnvironmentID)
	m.Agent = &deploymentAgentModel{
		Id:      types.StringValue(d.Agent.ID),
		Version: types.Int64Value(d.Agent.Version),
	}

	events := make([]deploymentInitialEventModel, 0, len(d.InitialEvents))
	for _, ev := range d.InitialEvents {
		blocks := make([]deploymentContentBlockModel, 0, len(ev.Content))
		for _, b := range ev.Content {
			blocks = append(blocks, deploymentContentBlockModel{
				Type: types.StringValue(b.Type),
				Text: types.StringValue(b.Text),
			})
		}
		events = append(events, deploymentInitialEventModel{
			Type:    types.StringValue(ev.Type),
			Content: blocks,
		})
	}
	m.InitialEvents = events

	if d.Schedule != nil {
		m.Schedule = &deploymentScheduleModel{
			Expression:     types.StringValue(d.Schedule.Expression),
			Timezone:       types.StringValue(d.Schedule.Timezone),
			LastRunAt:      nullableString(d.Schedule.LastRunAt),
			UpcomingRunsAt: stringList(d.Schedule.UpcomingRunsAt),
		}
	} else {
		m.Schedule = nil
	}

	m.VaultIds = stringList(d.VaultIDs)
	m.Metadata = fillMetadata(d.Metadata)
	m.Status = types.StringValue(d.Status)
	m.Paused = types.BoolValue(d.Status == "paused")

	if d.PausedReason != nil {
		errType := types.StringNull()
		if d.PausedReason.Error != nil {
			errType = types.StringValue(d.PausedReason.Error.Type)
		}
		obj, objDiags := types.ObjectValue(pausedReasonAttrTypes, map[string]attr.Value{
			"type":       types.StringValue(d.PausedReason.Type),
			"error_type": errType,
		})
		diags.Append(objDiags...)
		m.PausedReason = obj
	} else {
		m.PausedReason = types.ObjectNull(pausedReasonAttrTypes)
	}

	m.CreatedAt = types.StringValue(d.CreatedAt)
	m.UpdatedAt = types.StringValue(d.UpdatedAt)
	m.ArchivedAt = nullableString(d.ArchivedAt)
	return diags
}

func stringList(in []string) types.List {
	if len(in) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	vals := make([]attr.Value, len(in))
	for i, s := range in {
		vals[i] = types.StringValue(s)
	}
	return types.ListValueMust(types.StringType, vals)
}

type WIFDeploymentResource struct {
	data *providerData
}

func NewDeploymentResource() resource.Resource {
	return &WIFDeploymentResource{}
}

var _ resource.Resource = &WIFDeploymentResource{}
var _ resource.ResourceWithImportState = &WIFDeploymentResource{}
var _ resource.ResourceWithModifyPlan = &WIFDeploymentResource{}
var _ resource.ResourceWithConfigure = &WIFDeploymentResource{}

func (r *WIFDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *WIFDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic Managed Agents deployment: an agent bound to an environment with seed events and an optional cron schedule. Omit `schedule` for a manual (on-demand) deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Deployment ID (depl_...).",
			},
			"workspace_id": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the workspace this deployment belongs to. Required when using WIF authentication. Not needed when using workspace_api_key.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name for the deployment.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description of what the deployment does.",
			},
			"agent": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The agent to deploy.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:    true,
						Description: "Agent ID (agent_...).",
					},
					"version": schema.Int64Attribute{
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						Description:   "Agent version to pin. Omit to use the latest version at create time; the resolved version is then stored and not auto-updated.",
					},
				},
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the environment where sessions run.",
			},
			"initial_events": schema.ListNestedAttribute{
				Required:    true,
				Description: "Events sent to each session immediately after creation (1-50). Text-only this iteration: only user.message events with text content blocks.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: `Event type. Currently only "user.message" is supported.`,
						},
						"content": schema.ListNestedAttribute{
							Required:    true,
							Description: "Content blocks for the message.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Required:    true,
										Description: `Content block type. Currently only "text" is supported.`,
									},
									"text": schema.StringAttribute{
										Required:    true,
										Description: "The text content.",
									},
								},
							},
						},
					},
				},
			},
			"schedule": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Cron schedule. Omit for a manual (on-demand) deployment.",
				Attributes: map[string]schema.Attribute{
					"expression": schema.StringAttribute{
						Required:    true,
						Description: "5-field POSIX cron expression (e.g. \"0 9 * * 1-5\").",
					},
					"timezone": schema.StringAttribute{
						Required:    true,
						Description: "IANA timezone identifier (e.g. \"America/Los_Angeles\").",
					},
					"last_run_at": schema.StringAttribute{
						Computed:    true,
						Description: "Timestamp of the last run (RFC 3339).",
					},
					"upcoming_runs_at": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "Up to 5 upcoming run timestamps (RFC 3339).",
					},
				},
			},
			"vault_ids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Vault IDs supplying stored credentials for sessions.",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value metadata (max 16 pairs).",
			},
			"paused": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the deployment is paused. Toggling this calls the pause/unpause endpoints.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: `Lifecycle status: "active" or "paused".`,
			},
			"paused_reason": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Why the deployment is paused. Null when active.",
				Attributes: map[string]schema.Attribute{
					"type":       schema.StringAttribute{Computed: true},
					"error_type": schema.StringAttribute{Computed: true},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *WIFDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WIFDeploymentResource) buildBody(ctx context.Context, data DeploymentModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	agent := map[string]any{"id": data.Agent.Id.ValueString(), "type": "agent"}
	if !data.Agent.Version.IsNull() && !data.Agent.Version.IsUnknown() {
		agent["version"] = data.Agent.Version.ValueInt64()
	}

	events := make([]map[string]any, 0, len(data.InitialEvents))
	for _, ev := range data.InitialEvents {
		blocks := make([]map[string]any, 0, len(ev.Content))
		for _, b := range ev.Content {
			blocks = append(blocks, map[string]any{"type": b.Type.ValueString(), "text": b.Text.ValueString()})
		}
		events = append(events, map[string]any{"type": ev.Type.ValueString(), "content": blocks})
	}

	body := map[string]any{
		"name":           data.Name.ValueString(),
		"environment_id": data.EnvironmentId.ValueString(),
		"agent":          agent,
		"initial_events": events,
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() && len(data.Metadata.Elements()) > 0 {
		meta := make(map[string]string, len(data.Metadata.Elements()))
		diags.Append(data.Metadata.ElementsAs(ctx, &meta, false)...)
		body["metadata"] = meta
	}
	if !data.VaultIds.IsNull() && !data.VaultIds.IsUnknown() && len(data.VaultIds.Elements()) > 0 {
		var ids []string
		diags.Append(data.VaultIds.ElementsAs(ctx, &ids, false)...)
		body["vault_ids"] = ids
	}
	if data.Schedule != nil {
		body["schedule"] = map[string]any{
			"type":       "cron",
			"expression": data.Schedule.Expression.ValueString(),
			"timezone":   data.Schedule.Timezone.ValueString(),
		}
	}
	return body, diags
}

func (r *WIFDeploymentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan DeploymentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	workspaceID := ""
	if !plan.WorkspaceId.IsNull() && !plan.WorkspaceId.IsUnknown() {
		workspaceID = plan.WorkspaceId.ValueString()
	}
	validateWorkspaceCredentials(r.data, "anthropic_deployment", workspaceID, plan.WorkspaceId.IsUnknown(), &resp.Diagnostics)
}

func (r *WIFDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, r.data, "anthropic_deployment", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body, diags := r.buildBody(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := client.NewDeploymentClient(auth.WithBeta(creds, auth.AgentsBeta))
	dep, err := c.Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create deployment: %s", err))
		return
	}
	if data.Paused.ValueBool() {
		paused, pauseErr := c.Pause(ctx, dep.ID)
		if pauseErr != nil {
			// The deployment was created remotely. Persist it (active) so it is
			// tracked rather than orphaned, then surface the error. A re-apply
			// reconciles the paused state through the update path.
			resp.Diagnostics.Append(data.fill(*dep)...)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deployment %s created but pausing failed; it exists and is active. Re-apply to pause it: %s", dep.ID, pauseErr))
			return
		}
		dep = paused
	}
	resp.Diagnostics.Append(data.fill(*dep)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, r.data, "anthropic_deployment", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	dep, err := client.NewDeploymentClient(auth.WithBeta(creds, auth.AgentsBeta)).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment: %s", err))
		return
	}
	if dep == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(data.fill(*dep)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, r.data, "anthropic_deployment", plan.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body, diags := r.buildBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := client.NewDeploymentClient(auth.WithBeta(creds, auth.AgentsBeta))
	dep, err := c.Update(ctx, plan.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update deployment: %s", err))
		return
	}
	wantPaused := plan.Paused.ValueBool()
	if wantPaused != (dep.Status == "paused") {
		var toggled *client.DeploymentResponse
		if wantPaused {
			toggled, err = c.Pause(ctx, plan.Id.ValueString())
		} else {
			toggled, err = c.Unpause(ctx, plan.Id.ValueString())
		}
		if err != nil {
			// The field update already succeeded remotely (dep is the post-update
			// object). Persist it so state is not left on stale pre-update values,
			// then surface the error. A re-apply reconciles the paused state.
			resp.Diagnostics.Append(plan.fill(*dep)...)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deployment updated but changing pause state failed; re-apply to reconcile: %s", err))
			return
		}
		dep = toggled
	}
	resp.Diagnostics.Append(plan.fill(*dep)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WIFDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, r.data, "anthropic_deployment", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := client.NewDeploymentClient(auth.WithBeta(creds, auth.AgentsBeta)).Archive(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive deployment: %s", err))
	}
}

// ImportState supports two formats:
//   - workspace_id/deployment_id  (WIF path)
//   - deployment_id               (workspace_api_key path)
func (r *WIFDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/deployment_id or deployment_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "deployment_id must not be empty")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/deployment_id or deployment_id")
	}
}
