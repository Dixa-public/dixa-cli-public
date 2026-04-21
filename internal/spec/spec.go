package spec

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed operations.json
var manifestJSON []byte

type Manifest struct {
	Domains    []Domain    `json:"domains"`
	Operations []Operation `json:"operations"`
}

type Domain struct {
	Domain       string   `json:"domain"`
	Group        string   `json:"group"`
	GroupAliases []string `json:"group_aliases"`
	Label        string   `json:"label"`
}

type Operation struct {
	ID             string      `json:"id"`
	Domain         string      `json:"domain"`
	Group          string      `json:"group"`
	GroupAliases   []string    `json:"group_aliases"`
	Name           string      `json:"name"`
	Aliases        []string    `json:"aliases"`
	OriginalName   string      `json:"original_name"`
	Summary        string      `json:"summary"`
	Description    string      `json:"description"`
	Safety         string      `json:"safety"`
	Destructive    bool        `json:"destructive"`
	Mode           string      `json:"mode"`
	HTTPMethod     string      `json:"http_method"`
	PathTemplate   string      `json:"path_template"`
	SuccessMessage string      `json:"success_message"`
	Parameters     []Parameter `json:"parameters"`
}

type Parameter struct {
	Name        string   `json:"name"`
	Flag        string   `json:"flag"`
	FlagAliases []string `json:"flag_aliases"`
	Annotation  string   `json:"annotation"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     *string  `json:"default"`
	Location    string   `json:"location"`
	APIName     string   `json:"api_name"`
	Position    int      `json:"position"`
}

func Load() (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode embedded operation manifest: %w", err)
	}
	return manifest, nil
}

func MustLoad() Manifest {
	manifest, err := Load()
	if err != nil {
		panic(err)
	}
	return manifest
}

func (o Operation) PathParameters() []Parameter {
	var params []Parameter
	for _, param := range o.Parameters {
		if param.Location == "path" {
			params = append(params, param)
		}
	}
	return params
}

func (o Operation) HasWriteSideEffects() bool {
	return o.Safety == "write"
}
