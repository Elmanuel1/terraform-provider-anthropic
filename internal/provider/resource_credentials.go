package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// resolveWorkspaceCredentials returns the credentials for any workspace-scoped resource.
// WIF is used when workspace_id is set and WIF is fully configured.
// workspace_api_key is used otherwise. When both are configured, WIF takes precedence.
func resolveWorkspaceCredentials(ctx context.Context, data *providerData, resourceName, workspaceID string, diags interface{ AddError(string, string) }) auth.Credentials {
	if data == nil {
		diags.AddError("Provider not configured", "No provider data available.")
		return nil
	}

	if data.wif != nil && workspaceID != "" {
		tflog.Debug(ctx, fmt.Sprintf("%s: using WIF authentication", resourceName), map[string]any{"workspace_id": workspaceID})
		return auth.WIFBearer{Config: data.wif, WorkspaceID: workspaceID}
	}

	if data.workspaceAPIKey != "" {
		tflog.Debug(ctx, fmt.Sprintf("%s: using workspace API key authentication", resourceName))
		return auth.WorkspaceAPIKey{Key: data.workspaceAPIKey}
	}

	if workspaceID != "" && data.wifErr != nil {
		diags.AddError("Invalid WIF configuration", data.wifErr.Error())
	} else if workspaceID != "" {
		diags.AddError("Missing credentials",
			fmt.Sprintf("workspace_id is set but WIF is not fully configured and workspace_api_key is not set. "+
				"Set federation_rule_id, organization_id, service_account_id in the provider block, "+
				"or set workspace_api_key. (resource: %s)", resourceName))
	} else {
		diags.AddError("Missing credentials",
			fmt.Sprintf("No authentication method is configured for %s. "+
				"Set workspace_api_key in the provider block, "+
				"or set workspace_id together with WIF credentials (federation_rule_id, organization_id, service_account_id).", resourceName))
	}
	return nil
}

// validateWorkspaceCredentials is called from ModifyPlan to catch misconfigurations at plan time.
// workspaceID may be empty-string when the value is unknown at plan time (referencing a
// not-yet-created resource) — in that case WIF path validation is deferred to apply.
func validateWorkspaceCredentials(data *providerData, resourceName, workspaceID string, diags interface{ AddError(string, string) }) {
	if data == nil {
		return
	}
	wifConfigured := data.wif != nil
	apiKeyConfigured := data.workspaceAPIKey != ""
	if !wifConfigured && !apiKeyConfigured {
		diags.AddError(
			"Missing credentials",
			fmt.Sprintf("No authentication method is configured for %s. "+
				"Set workspace_id together with WIF credentials (federation_rule_id, organization_id, service_account_id), "+
				"or set workspace_api_key in the provider block.", resourceName),
		)
		return
	}
	// WIF is configured but workspace_id is definitively absent (not just unknown).
	if wifConfigured && !apiKeyConfigured && workspaceID == "" {
		diags.AddError(
			"Missing workspace_id",
			fmt.Sprintf("WIF authentication is configured for %s but workspace_id is not set. "+
				"Set workspace_id, or configure workspace_api_key instead.", resourceName),
		)
	}
}
