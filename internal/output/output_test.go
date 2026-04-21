package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	if got := ResolveFormat("auto", true); got != "table" {
		t.Fatalf("expected table for tty auto, got %q", got)
	}
	if got := ResolveFormat("auto", false); got != "json" {
		t.Fatalf("expected json for non-tty auto, got %q", got)
	}
	if got := ResolveFormat("yaml", false); got != "yaml" {
		t.Fatalf("expected yaml to pass through, got %q", got)
	}
}

func TestRenderJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := Render(&buf, "json", map[string]any{"hello": "world"}); err != nil {
		t.Fatalf("render json: %v", err)
	}
	if !strings.Contains(buf.String(), "\"hello\": \"world\"") {
		t.Fatalf("expected indented json output, got %q", buf.String())
	}
}

func TestRenderTable(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	value := map[string]any{
		"data": []any{
			map[string]any{"id": "agent-1", "name": "Alice"},
			map[string]any{"id": "agent-2", "name": "Bob"},
		},
	}
	if err := Render(&buf, "table", value); err != nil {
		t.Fatalf("render table: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "agent-1") || !strings.Contains(output, "Alice") {
		t.Fatalf("expected tabular output, got %q", output)
	}
}
