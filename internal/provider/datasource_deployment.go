package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeploymentDataSource struct {
	data *providerData
}

func NewDeploymentDataSource() datasource.DataSource {
	return &DeploymentDataSource{}
}

var _ datasource.DataSource = &DeploymentDataSource{}
var _ datasource.DataSourceWithConfigure = &DeploymentDataSource{}

func (d *DeploymentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *DeploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic deployment by ID.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Required: true, Description: "Deployment ID (depl_...)."},
			"workspace_id": schema.StringAttribute{Optional: true, Description: "Workspace ID. Required when using WIF authentication."},
			"name":         schema.StringAttribute{Computed: true},
			"description":  schema.StringAttribute{Computed: true},
			"agent": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id":      schema.StringAttribute{Computed: true},
					"version": schema.Int64Attribute{Computed: true},
				},
			},
			"environment_id": schema.StringAttribute{Computed: true},
			"initial_events": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{Computed: true},
						"content": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{Computed: true},
									"text": schema.StringAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
			"schedule": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"expression":       schema.StringAttribute{Computed: true},
					"timezone":         schema.StringAttribute{Computed: true},
					"last_run_at":      schema.StringAttribute{Computed: true},
					"upcoming_runs_at": schema.ListAttribute{Computed: true, ElementType: types.StringType},
				},
			},
			"vault_ids": schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"metadata":  schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"paused":    schema.BoolAttribute{Computed: true},
			"status":    schema.StringAttribute{Computed: true},
			"paused_reason": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type":       schema.StringAttribute{Computed: true},
					"error_type": schema.StringAttribute{Computed: true},
				},
			},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *DeploymentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeploymentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_deployment", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	dep, err := client.NewDeploymentClient(auth.WithBeta(creds, auth.AgentsBeta)).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment: %s", err))
		return
	}
	if dep == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Deployment %q not found.", data.Id.ValueString()))
		return
	}
	resp.Diagnostics.Append(data.fill(*dep)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
