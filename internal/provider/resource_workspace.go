package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkspaceResource struct {
	data *providerData
}

type WorkspaceModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CreatedAt  types.String `tfsdk:"created_at"`
	ArchivedAt types.String `tfsdk:"archived_at"`
}

type workspaceAPIResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	ArchivedAt *string `json:"archived_at"`
}

func (m *WorkspaceModel) fill(w workspaceAPIResponse) {
	m.Id = types.StringValue(w.ID)
	m.Name = types.StringValue(w.Name)
	m.CreatedAt = types.StringValue(w.CreatedAt)
	m.ArchivedAt = nullableString(w.ArchivedAt)
}

func NewWorkspaceResource() resource.Resource {
	return &WorkspaceResource{}
}

var _ resource.Resource = &WorkspaceResource{}
var _ resource.ResourceWithImportState = &WorkspaceResource{}

func (r *WorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *WorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Workspace name as it appears in the Anthropic Console.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *WorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkspaceResource) creds() auth.Credentials {
	return auth.AdminAPIKey{Key: r.data.client.APIKey}
}

func (r *WorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{"name": data.Name.ValueString()}
	raw, status, err := client.DoAdminRequest(ctx, r.data.client, r.creds(), http.MethodPost, "/v1/organizations/workspaces", body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workspace: %s", err))
		return
	}
	if status != http.StatusOK && status != http.StatusCreated {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workspace, status %d: %s", status, raw))
		return
	}

	var w workspaceAPIResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse workspace response: %s", err))
		return
	}
	data.fill(w)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, status, err := client.DoAdminRequest(ctx, r.data.client, r.creds(), http.MethodGet, "/v1/organizations/workspaces/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workspace: %s", err))
		return
	}
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workspace, status %d: %s", status, raw))
		return
	}

	var w workspaceAPIResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse workspace response: %s", err))
		return
	}
	data.fill(w)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// name carries RequiresReplace; Update is never called.
}

func (r *WorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, status, err := client.DoAdminRequest(ctx, r.data.client, r.creds(), http.MethodDelete, "/v1/organizations/workspaces/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workspace: %s", err))
		return
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workspace, status %d", status))
	}
}

func (r *WorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name — resolve to ID, then let Read populate the rest.
	id, err := auth.ResolveWorkspaceID(ctx, r.creds(), req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to resolve workspace %q: %s", req.ID, err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
