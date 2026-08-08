package systemd

import (
	"context"
	"fmt"

	"github.com/hairglasses-studio/mcpkit/prompts"
	"github.com/hairglasses-studio/mcpkit/registry"
	"github.com/hairglasses-studio/mcpkit/resources"
)

// newDocResource builds a registry-aliased Resource via mcpkit's compat
// constructor plus field assignment, instead of mcp-go's functional options
// (mcp.WithResourceDescription etc.), which the compat layer deliberately
// does not re-export. This file must not import mark3labs/mcp-go (or
// modelcontextprotocol/go-sdk) directly so it compiles under both build
// tags unchanged.
func newDocResource(uri, name, description, mimeType string) registry.Resource {
	r := registry.NewResource(uri, name)
	r.Description = description
	r.MIMEType = mimeType
	return r
}

type systemdResourceModule struct{}

func (m *systemdResourceModule) Name() string { return "systemd_context" }
func (m *systemdResourceModule) Description() string {
	return "Reusable systemd troubleshooting context"
}

func (m *systemdResourceModule) Resources() []resources.ResourceDefinition {
	return []resources.ResourceDefinition{
		{
			Resource: newDocResource(
				"systemd://workflows/unit-triage",
				"Systemd Unit Triage",
				"Compact workflow for diagnosing a failing or noisy systemd unit",
				"text/markdown",
			),
			Handler: resources.TextResourceHandler(func(_ context.Context, _ string) (string, string, error) {
				return "text/markdown", "1. Run `systemd_status` to confirm load, active, sub-state, pid, and fragment path.\n2. Run `systemd_logs` with a bounded line count for recent evidence.\n3. Run `systemd_failed` or `systemd_list_units` if the issue may be broader than one unit.\n4. Only reach for `systemd_restart`, `systemd_stop`, or `systemd_disable` after the read path explains the failure.", nil
			}),
			Category: "workflow",
			Tags:     []string{"triage", "debugging", "systemd"},
		},
		{
			Resource: newDocResource(
				"systemd://runtime/capabilities",
				"Systemd Runtime Capabilities",
				"Live backend capability report for user and system scope",
				"application/json",
			),
			Handler: resources.JSONResourceHandler(func(_ context.Context, _ string) (any, error) {
				return detectRuntimeCapabilities(), nil
			}),
			Category: "runtime",
			Tags:     []string{"systemd", "capabilities", "runtime"},
		},
	}
}

func (m *systemdResourceModule) Templates() []resources.TemplateDefinition { return nil }

type systemdPromptModule struct{}

func (m *systemdPromptModule) Name() string { return "systemd_prompts" }
func (m *systemdPromptModule) Description() string {
	return "Prompt workflows for systemd investigations"
}

func (m *systemdPromptModule) Prompts() []prompts.PromptDefinition {
	return []prompts.PromptDefinition{
		{
			Prompt: registry.MakePrompt(
				"systemd_triage_unit",
				"Guide a bounded investigation of a systemd unit before any write action",
				registry.PromptArgument{Name: "unit", Description: "Systemd unit name to investigate", Required: true},
				registry.PromptArgument{Name: "scope", Description: "user (default) or system"},
			),
			Handler: prompts.TextPromptHandler(func(_ context.Context, args map[string]string) (string, string, error) {
				unit := args["unit"]
				scope := args["scope"]
				if scope == "" {
					scope = "user"
				}
				return "Investigate systemd unit " + unit, fmt.Sprintf(
					"Investigate the %s-scoped unit %q. Start with `systemd_status`, then use `systemd_logs` with a bounded line count, and only suggest `systemd_restart`, `systemd_stop`, or `systemd_disable` if the evidence justifies a write action.",
					scope, unit,
				), nil
			}),
			Category: "workflow",
			Tags:     []string{"systemd", "triage", "debugging"},
		},
	}
}

func buildSystemdResourceRegistry() *resources.ResourceRegistry {
	reg := resources.NewResourceRegistry()
	reg.RegisterModule(&systemdResourceModule{})
	return reg
}

func buildSystemdPromptRegistry() *prompts.PromptRegistry {
	reg := prompts.NewPromptRegistry()
	reg.RegisterModule(&systemdPromptModule{})
	return reg
}
