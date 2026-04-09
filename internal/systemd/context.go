package systemd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hairglasses-studio/mcpkit/prompts"
	"github.com/hairglasses-studio/mcpkit/resources"
	"github.com/mark3labs/mcp-go/mcp"
)

type systemdResourceModule struct{}

func (m *systemdResourceModule) Name() string { return "systemd_context" }
func (m *systemdResourceModule) Description() string {
	return "Reusable systemd troubleshooting context"
}

func (m *systemdResourceModule) Resources() []resources.ResourceDefinition {
	return []resources.ResourceDefinition{
		{
			Resource: mcp.NewResource(
				"systemd://workflows/unit-triage",
				"Systemd Unit Triage",
				mcp.WithResourceDescription("Compact workflow for diagnosing a failing or noisy systemd unit"),
				mcp.WithMIMEType("text/markdown"),
			),
			Handler: func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      "systemd://workflows/unit-triage",
						MIMEType: "text/markdown",
						Text:     "1. Run `systemd_status` to confirm load, active, sub-state, pid, and fragment path.\n2. Run `systemd_logs` with a bounded line count for recent evidence.\n3. Run `systemd_failed` or `systemd_list_units` if the issue may be broader than one unit.\n4. Only reach for `systemd_restart`, `systemd_stop`, or `systemd_disable` after the read path explains the failure.",
					},
				}, nil
			},
			Category: "workflow",
			Tags:     []string{"triage", "debugging", "systemd"},
		},
		{
			Resource: mcp.NewResource(
				"systemd://runtime/capabilities",
				"Systemd Runtime Capabilities",
				mcp.WithResourceDescription("Live backend capability report for user and system scope"),
				mcp.WithMIMEType("application/json"),
			),
			Handler: func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				body, err := json.MarshalIndent(detectRuntimeCapabilities(), "", "  ")
				if err != nil {
					return nil, err
				}
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      "systemd://runtime/capabilities",
						MIMEType: "application/json",
						Text:     string(body),
					},
				}, nil
			},
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
			Prompt: mcp.NewPrompt(
				"systemd_triage_unit",
				mcp.WithPromptDescription("Guide a bounded investigation of a systemd unit before any write action"),
				mcp.WithArgument("unit", mcp.RequiredArgument(), mcp.ArgumentDescription("Systemd unit name to investigate")),
				mcp.WithArgument("scope", mcp.ArgumentDescription("user (default) or system")),
			),
			Handler: func(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				unit := req.Params.Arguments["unit"]
				scope := req.Params.Arguments["scope"]
				if scope == "" {
					scope = "user"
				}
				return &mcp.GetPromptResult{
					Description: "Investigate systemd unit " + unit,
					Messages: []mcp.PromptMessage{
						mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(fmt.Sprintf(
							"Investigate the %s-scoped unit %q. Start with `systemd_status`, then use `systemd_logs` with a bounded line count, and only suggest `systemd_restart`, `systemd_stop`, or `systemd_disable` if the evidence justifies a write action.",
							scope, unit,
						))),
					},
				}, nil
			},
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
