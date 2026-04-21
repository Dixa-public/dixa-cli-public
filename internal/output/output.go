package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

func ResolveFormat(requested string, stdoutTTY bool) string {
	switch strings.ToLower(strings.TrimSpace(requested)) {
	case "", "auto":
		if stdoutTTY {
			return "table"
		}
		return "json"
	case "json", "table", "yaml":
		return strings.ToLower(strings.TrimSpace(requested))
	default:
		if stdoutTTY {
			return "table"
		}
		return "json"
	}
}

func Render(w io.Writer, format string, value any) error {
	switch ResolveFormat(format, true) {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case "yaml":
		encoded, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal yaml output: %w", err)
		}
		_, err = fmt.Fprintln(w, string(encoded))
		return err
	default:
		return renderTable(w, value)
	}
}

func renderTable(w io.Writer, value any) error {
	if rows, extras, ok := extractRows(value); ok {
		tw := table.NewWriter()
		tw.SetOutputMirror(w)
		tw.AppendHeader(rows.header)
		for _, row := range rows.rows {
			tw.AppendRow(row)
		}
		tw.Render()
		if extras != nil {
			encoded, _ := json.MarshalIndent(extras, "", "  ")
			_, _ = fmt.Fprintf(w, "\n%s\n", string(encoded))
		}
		return nil
	}

	kv := table.NewWriter()
	kv.SetOutputMirror(w)
	kv.AppendHeader(table.Row{"Field", "Value"})
	for _, entry := range keyValueRows(value) {
		kv.AppendRow(entry)
	}
	kv.Render()
	return nil
}

type tabular struct {
	header table.Row
	rows   []table.Row
}

func extractRows(value any) (tabular, any, bool) {
	switch typed := value.(type) {
	case []any:
		return buildRows(typed), nil, true
	case map[string]any:
		if data, ok := typed["data"]; ok {
			switch rows := data.(type) {
			case []any:
				extras := map[string]any{}
				for key, item := range typed {
					if key != "data" {
						extras[key] = item
					}
				}
				if len(extras) == 0 {
					extras = nil
				}
				return buildRows(rows), extras, true
			case map[string]any:
				return tabular{}, nil, false
			}
		}
	}
	return tabular{}, nil, false
}

func buildRows(items []any) tabular {
	keys := map[string]struct{}{}
	for _, item := range items {
		rowMap, ok := item.(map[string]any)
		if !ok {
			keys["value"] = struct{}{}
			continue
		}
		for key := range rowMap {
			keys[key] = struct{}{}
		}
	}

	columns := make([]string, 0, len(keys))
	for key := range keys {
		columns = append(columns, key)
	}
	sort.Strings(columns)

	header := make(table.Row, 0, len(columns))
	for _, column := range columns {
		header = append(header, column)
	}

	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rowMap, ok := item.(map[string]any)
		if !ok {
			rows = append(rows, table.Row{displayValue(item)})
			continue
		}
		row := make(table.Row, 0, len(columns))
		for _, column := range columns {
			row = append(row, displayValue(rowMap[column]))
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		rows = append(rows, table.Row{"(no rows)"})
		header = table.Row{"Result"}
	}

	return tabular{header: header, rows: rows}
}

func keyValueRows(value any) []table.Row {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		rows := make([]table.Row, 0, len(keys))
		for _, key := range keys {
			rows = append(rows, table.Row{key, displayValue(typed[key])})
		}
		if len(rows) == 0 {
			return []table.Row{{"result", "(empty)"}}
		}
		return rows
	default:
		return []table.Row{{"result", displayValue(value)}}
	}
}

func displayValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return ""
	case string, bool, int, int32, int64, float32, float64:
		return typed
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	}
}
