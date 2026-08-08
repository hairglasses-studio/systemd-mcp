package systemd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hairglasses-studio/mcpkit/registry"
	"github.com/hairglasses-studio/mcpkit/resources"
)

// ---------------------------------------------------------------------------
// Resource registry
// ---------------------------------------------------------------------------

func TestSystemdResourceModule_Metadata(t *testing.T) {
	m := &systemdResourceModule{}
	if m.Name() != "systemd_context" {
		t.Errorf("Name() = %q, want %q", m.Name(), "systemd_context")
	}
	if m.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestSystemdResourceModule_Resources(t *testing.T) {
	m := &systemdResourceModule{}
	defs := m.Resources()
	if len(defs) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(defs))
	}

	rd := defs[0]
	if rd.Category != "workflow" {
		t.Errorf("Category = %q, want %q", rd.Category, "workflow")
	}
	if len(rd.Tags) == 0 {
		t.Error("expected tags")
	}

	mimeType, text, err := resources.CallHandlerText(context.Background(), rd.Handler, "systemd://workflows/unit-triage")
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if mimeType != "text/markdown" {
		t.Errorf("mimeType = %q, want %q", mimeType, "text/markdown")
	}
	if text == "" {
		t.Error("resource text is empty")
	}
}

func TestSystemdRuntimeCapabilitiesResource(t *testing.T) {
	m := &systemdResourceModule{}
	defs := m.Resources()

	var runtimeDef resources.ResourceDefinition
	found := false
	for _, rd := range defs {
		if rd.Resource.URI == "systemd://runtime/capabilities" {
			runtimeDef = rd
			found = true
			break
		}
	}
	if !found {
		t.Fatal("runtime capabilities resource not found")
	}

	mimeType, text, err := resources.CallHandlerText(context.Background(), runtimeDef.Handler, "systemd://runtime/capabilities")
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if mimeType != "application/json" {
		t.Errorf("mimeType = %q, want %q", mimeType, "application/json")
	}

	var caps RuntimeCapabilities
	if err := json.Unmarshal([]byte(text), &caps); err != nil {
		t.Fatalf("invalid runtime capabilities json: %v", err)
	}
	if caps.User.Scope != "user" {
		t.Fatalf("unexpected user scope %q", caps.User.Scope)
	}
	if caps.System.Scope != "system" {
		t.Fatalf("unexpected system scope %q", caps.System.Scope)
	}
}

func TestSystemdResourceModule_NilTemplates(t *testing.T) {
	m := &systemdResourceModule{}
	if m.Templates() != nil {
		t.Error("expected nil templates")
	}
}

// ---------------------------------------------------------------------------
// Prompt registry
// ---------------------------------------------------------------------------

func TestSystemdPromptModule_Metadata(t *testing.T) {
	m := &systemdPromptModule{}
	if m.Name() != "systemd_prompts" {
		t.Errorf("Name() = %q, want %q", m.Name(), "systemd_prompts")
	}
	if m.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestSystemdPromptModule_Prompts(t *testing.T) {
	m := &systemdPromptModule{}
	defs := m.Prompts()
	if len(defs) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(defs))
	}

	pd := defs[0]
	if pd.Category != "workflow" {
		t.Errorf("Category = %q, want %q", pd.Category, "workflow")
	}
}

func TestSystemdPrompt_Handler(t *testing.T) {
	m := &systemdPromptModule{}
	pd := m.Prompts()[0]

	result, err := callPrompt(pd.Handler, map[string]string{
		"unit":  "nginx.service",
		"scope": "system",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.Description == "" {
		t.Error("Description is empty")
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
}

func TestSystemdPrompt_DefaultScope(t *testing.T) {
	m := &systemdPromptModule{}
	pd := m.Prompts()[0]

	result, err := callPrompt(pd.Handler, map[string]string{
		"unit": "test.service",
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Messages) > 0 {
		if text, ok := registry.ExtractTextContent(result.Messages[0].Content); ok && text != "" {
			assertContains(t, text, "user-scoped")
		}
	}
}
