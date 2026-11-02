package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The acceptance tests run real applies against the API at ANTHROPIC_BASE_URL
// (an Anthropic-compatible server such as agentd, or the live API when it is
// unset), with ANTHROPIC_API_KEY and ANTHROPIC_ADMIN_API_KEY. They run only
// with TF_ACC=1:
//
//	TF_ACC=1 ANTHROPIC_BASE_URL=http://127.0.0.1:8080 ANTHROPIC_API_KEY=k ANTHROPIC_ADMIN_API_KEY=k go test ./internal/provider -run TestAcc

var accProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"anthropic": providerserver.NewProtocol6WithError(New()),
}

func accPreCheck(t *testing.T) {
	t.Helper()
	for _, v := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_ADMIN_API_KEY"} {
		if os.Getenv(v) == "" {
			t.Fatalf("%s must be set for acceptance tests", v)
		}
	}
}

func accName(prefix string) string {
	return fmt.Sprintf("tfacc-%s-%d", prefix, time.Now().UnixNano())
}

func TestAccEnvironmentAgentDeployment(t *testing.T) {
	name := accName("stack")
	config := func(system string) string {
		return fmt.Sprintf(`
provider "anthropic" {}

resource "anthropic_environment" "e" {
  name            = "%[1]s-env"
  networking_type = "unrestricted"
}

resource "anthropic_agent" "a" {
  name   = "%[1]s-agent"
  model  = "claude-sonnet-4-6"
  system = %[2]q
}

resource "anthropic_deployment" "d" {
  name           = "%[1]s-deploy"
  environment_id = anthropic_environment.e.id
  agent = {
    id = anthropic_agent.a.id
  }
  initial_events = [{
    type    = "user.message"
    content = [{ type = "text", text = "Run triage." }]
  }]
  schedule = {
    expression = "0 9 * * 1-5"
    timezone   = "UTC"
  }
}
`, name, system)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: config("first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("anthropic_environment.e", "id", envID),
					resource.TestMatchResourceAttr("anthropic_agent.a", "id", agentID),
					resource.TestCheckResourceAttr("anthropic_agent.a", "system", "first"),
					resource.TestCheckResourceAttrSet("anthropic_deployment.d", "id"),
					resource.TestCheckResourceAttrSet("anthropic_deployment.d", "schedule.upcoming_runs_at.0"),
				),
			},
			{
				Config: config("second"),
				Check:  resource.TestCheckResourceAttr("anthropic_agent.a", "system", "second"),
			},
		},
	})
}

func TestAccMemoryStore(t *testing.T) {
	name := accName("mem")
	config := func(n string) string {
		return fmt.Sprintf("provider \"anthropic\" {}\n\nresource \"anthropic_memory_store\" \"m\" {\n  name = %q\n}\n", n)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{
			{
				Config: config(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("anthropic_memory_store.m", "id", memstoreID),
					resource.TestCheckResourceAttr("anthropic_memory_store.m", "name", name),
				),
			},
			{
				Config: config(name + "-renamed"),
				Check:  resource.TestCheckResourceAttr("anthropic_memory_store.m", "name", name+"-renamed"),
			},
		},
	})
}

func TestAccVault(t *testing.T) {
	name := accName("vault")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "anthropic" {}

resource "anthropic_vault" "v" {
  display_name = %q
  metadata     = { team = "platform" }
}
`, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("anthropic_vault.v", "id"),
				resource.TestCheckResourceAttr("anthropic_vault.v", "display_name", name),
				resource.TestCheckResourceAttr("anthropic_vault.v", "metadata.team", "platform"),
			),
		}},
	})
}

func TestAccSkill(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviders,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "anthropic" {}

resource "anthropic_skill" "s" {
  display_title = %q
  source_dir    = "testdata/acc-skill"
}
`, accName("skill")),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("anthropic_skill.s", "id"),
				resource.TestCheckResourceAttrSet("anthropic_skill.s", "source_hash"),
			),
		}},
	})
}

var (
	envID      = regexp.MustCompile(`^env_`)
	agentID    = regexp.MustCompile(`^agent_`)
	memstoreID = regexp.MustCompile(`^memstore_`)
)
