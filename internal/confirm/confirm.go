package confirm

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Dixa-public/dixa-cli-public/internal/spec"
)

func Ensure(op spec.Operation, params map[string]any, yes bool, stdin io.Reader, stderr io.Writer, interactive bool) error {
	if !op.HasWriteSideEffects() {
		return nil
	}

	printSummary(op, params, stderr)

	if op.Destructive {
		if !yes {
			return fmt.Errorf("%s is destructive and requires --yes", op.ID)
		}
		return nil
	}

	if yes {
		return nil
	}
	if !interactive {
		return fmt.Errorf("non-interactive write commands require --yes for %s", op.ID)
	}

	if stderr != nil {
		fmt.Fprint(stderr, "Proceed? [y/N]: ")
	}
	reader := bufio.NewReader(stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted")
	}
	return nil
}

func printSummary(op spec.Operation, params map[string]any, stderr io.Writer) {
	if stderr == nil {
		return
	}
	prefix := "write"
	if op.Destructive {
		prefix = "danger"
	}
	fmt.Fprintf(stderr, "[%s] %s\n", prefix, op.ID)

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(stderr, "  %s=%v\n", key, params[key])
	}
}
