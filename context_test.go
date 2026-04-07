package main

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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
	resources := m.Resources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	rd := resources[0]
	if rd.Category != "workflow" {
		t.Errorf("Category = %q, want %q", rd.Category, "workflow")
	}
	if len(rd.Tags) == 0 {
		t.Error("expected tags")
	}

	// Call the handler
	contents, err := rd.Handler(context.Background(), mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	tc, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("expected TextResourceContents, got %T", contents[0])
	}
	if tc.Text == "" {
		t.Error("resource text is empty")
	}
	if tc.URI != "systemd://workflows/unit-triage" {
		t.Errorf("URI = %q, want %q", tc.URI, "systemd://workflows/unit-triage")
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
	prompts := m.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}

	pd := prompts[0]
	if pd.Category != "workflow" {
		t.Errorf("Category = %q, want %q", pd.Category, "workflow")
	}
}

func TestSystemdPrompt_Handler(t *testing.T) {
	m := &systemdPromptModule{}
	pd := m.Prompts()[0]

	// Test with explicit arguments
	req := mcp.GetPromptRequest{}
	req.Params.Arguments = map[string]string{
		"unit":  "nginx.service",
		"scope": "system",
	}

	result, err := pd.Handler(context.Background(), req)
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

	// Test with no scope — should default to "user"
	req := mcp.GetPromptRequest{}
	req.Params.Arguments = map[string]string{
		"unit": "test.service",
	}

	result, err := pd.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	// The prompt content should mention "user-scoped"
	if len(result.Messages) > 0 {
		tc, ok := result.Messages[0].Content.(mcp.TextContent)
		if ok && tc.Text != "" {
			assertContains(t, tc.Text, "user-scoped")
		}
	}
}
