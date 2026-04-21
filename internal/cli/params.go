package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dixa-public/dixa-cli-public/internal/spec"
)

func collectParams(cmd *cobra.Command, args []string, op spec.Operation, stdin io.Reader) (map[string]any, error) {
	params := map[string]any{}
	inputValues, err := readInputValues(cmd, stdin)
	if err != nil {
		return nil, err
	}

	for _, param := range op.Parameters {
		if raw, ok := lookupInputValue(param, inputValues); ok {
			value, err := normalizeParamValue(param, raw)
			if err != nil {
				return nil, fmt.Errorf("parse input value for %s: %w", param.Name, err)
			}
			params[param.Name] = value
		}
	}

	for _, param := range op.Parameters {
		raw, ok, err := readFlagValue(cmd, param)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		value, err := normalizeParamValue(param, raw)
		if err != nil {
			return nil, fmt.Errorf("parse flag value for --%s: %w", param.Flag, err)
		}
		params[param.Name] = value
	}

	pathParams := op.PathParameters()
	sort.Slice(pathParams, func(i, j int) bool { return pathParams[i].Position < pathParams[j].Position })
	if len(args) > len(pathParams) {
		return nil, fmt.Errorf("%s accepts at most %d positional arguments", op.Name, len(pathParams))
	}
	for index, arg := range args {
		value, err := normalizeParamValue(pathParams[index], arg)
		if err != nil {
			return nil, fmt.Errorf("parse positional argument %d: %w", index+1, err)
		}
		params[pathParams[index].Name] = value
	}

	for _, param := range op.Parameters {
		if param.Required {
			if _, ok := params[param.Name]; !ok {
				return nil, fmt.Errorf("missing required parameter %q", param.Name)
			}
		}
	}

	return params, nil
}

func readInputValues(cmd *cobra.Command, stdin io.Reader) (map[string]any, error) {
	inputPath, err := cmd.Flags().GetString("input")
	if err != nil || strings.TrimSpace(inputPath) == "" {
		return map[string]any{}, err
	}

	var data []byte
	if inputPath == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(inputPath)
	}
	if err != nil {
		return nil, fmt.Errorf("read input %q: %w", inputPath, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse input JSON %q: %w", inputPath, err)
	}
	return parsed, nil
}

func lookupInputValue(param spec.Parameter, values map[string]any) (any, bool) {
	candidates := []string{
		param.Name,
		param.Flag,
		param.APIName,
		strings.TrimPrefix(param.Name, "_"),
		strings.TrimPrefix(param.APIName, "_"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if value, ok := values[candidate]; ok {
			return value, true
		}
	}
	return nil, false
}

func readFlagValue(cmd *cobra.Command, param spec.Parameter) (any, bool, error) {
	for _, flagName := range append([]string{param.Flag}, param.FlagAliases...) {
		if !flagChanged(cmd, flagName) {
			continue
		}
		switch param.Type {
		case "int":
			value, err := cmd.Flags().GetInt(flagName)
			return value, true, err
		case "bool":
			value, err := cmd.Flags().GetBool(flagName)
			return value, true, err
		case "string_slice":
			value, err := cmd.Flags().GetStringSlice(flagName)
			return value, true, err
		default:
			value, err := cmd.Flags().GetString(flagName)
			return value, true, err
		}
	}
	return nil, false, nil
}

func normalizeParamValue(param spec.Parameter, raw any) (any, error) {
	switch param.Type {
	case "string":
		return fmt.Sprint(raw), nil
	case "int":
		switch value := raw.(type) {
		case int:
			return value, nil
		case int64:
			return int(value), nil
		case float64:
			return int(value), nil
		case json.Number:
			parsed, err := value.Int64()
			return int(parsed), err
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("unsupported int value %T", raw)
		}
	case "bool":
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			return strconv.ParseBool(strings.TrimSpace(value))
		default:
			return nil, fmt.Errorf("unsupported bool value %T", raw)
		}
	case "string_slice":
		switch value := raw.(type) {
		case []string:
			return value, nil
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				out = append(out, fmt.Sprint(item))
			}
			return out, nil
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				return []string{}, nil
			}
			if strings.HasPrefix(trimmed, "[") {
				var out []string
				if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
					return nil, err
				}
				return out, nil
			}
			return []string{trimmed}, nil
		default:
			return nil, fmt.Errorf("unsupported string slice value %T", raw)
		}
	case "int_slice":
		switch value := raw.(type) {
		case []int:
			return value, nil
		case []any:
			out := make([]int, 0, len(value))
			for _, item := range value {
				normalized, err := normalizeParamValue(spec.Parameter{Type: "int"}, item)
				if err != nil {
					return nil, err
				}
				out = append(out, normalized.(int))
			}
			return out, nil
		case string:
			var out []int
			if err := json.Unmarshal([]byte(value), &out); err != nil {
				return nil, err
			}
			return out, nil
		default:
			return nil, fmt.Errorf("unsupported int slice value %T", raw)
		}
	case "json":
		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				return nil, nil
			}
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
				return nil, err
			}
			return parsed, nil
		default:
			return raw, nil
		}
	default:
		return raw, nil
	}
}
