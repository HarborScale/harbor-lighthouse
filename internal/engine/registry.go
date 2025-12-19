package engine

import (
	"encoding/json"
	"fmt"
)

type HarborDef struct {
	Mode           string `json:"mode"`            // "cargo" or "raw"
	EndpointSuffix string `json:"endpoint_suffix"`
}

var Registry map[string]HarborDef

func Load(data []byte) error {
	return json.Unmarshal(data, &Registry)
}

func Get(typeName string) (HarborDef, error) {
	if def, ok := Registry[typeName]; ok {
		return def, nil
	}
	return HarborDef{}, fmt.Errorf("unknown harbor type: %s", typeName)
}
