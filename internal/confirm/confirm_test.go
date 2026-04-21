package confirm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Dixa-public/dixa-cli-public/internal/spec"
)

func TestEnsureReadDoesNotPrompt(t *testing.T) {
	t.Parallel()

	err := Ensure(spec.Operation{ID: "agents.list", Safety: "read"}, nil, false, strings.NewReader(""), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("read operation should not require confirmation: %v", err)
	}
}

func TestEnsureWriteRequiresYesInNonTTY(t *testing.T) {
	t.Parallel()

	err := Ensure(spec.Operation{ID: "agents.add_agent", Safety: "write"}, map[string]any{"display_name": "Alice"}, false, strings.NewReader(""), &bytes.Buffer{}, false)
	if err == nil {
		t.Fatalf("expected non-tty write command to require --yes")
	}
}

func TestEnsureWritePromptsInTTY(t *testing.T) {
	t.Parallel()

	err := Ensure(spec.Operation{ID: "agents.add_agent", Safety: "write"}, map[string]any{"display_name": "Alice"}, false, strings.NewReader("yes\n"), &bytes.Buffer{}, true)
	if err != nil {
		t.Fatalf("expected interactive confirmation to succeed: %v", err)
	}
}

func TestEnsureDestructiveAlwaysNeedsYes(t *testing.T) {
	t.Parallel()

	err := Ensure(spec.Operation{ID: "users.anonymize_end_user", Safety: "write", Destructive: true}, map[string]any{"user_id": "user-1"}, false, strings.NewReader("yes\n"), &bytes.Buffer{}, true)
	if err == nil {
		t.Fatalf("expected destructive command to require --yes")
	}
}
