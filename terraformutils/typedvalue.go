// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformutils

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// UnknownValue marks an unknown Terraform value in the cleanup projection.
// It deliberately differs from nil, which represents Terraform null.
type UnknownValue struct{}

// ProjectTerraformValue recursively converts a typed Terraform value into the
// map/slice view consumed by filters and provider cleanup hooks. Collection
// kinds remain available from ResourceState.Value; this projection preserves
// null and unknown as distinct Go values.
func ProjectTerraformValue(value tftypes.Value) (interface{}, error) {
	if !value.IsKnown() {
		return UnknownValue{}, nil
	}
	if value.IsNull() {
		return nil, nil
	}
	if value.Type().Is(tftypes.String) {
		var result string
		return result, value.As(&result)
	}
	if value.Type().Is(tftypes.Bool) {
		var result bool
		return result, value.As(&result)
	}
	if value.Type().Is(tftypes.Number) {
		result := new(big.Float)
		if err := value.As(&result); err != nil {
			return nil, err
		}
		return json.Number(result.Text('f', -1)), nil
	}

	switch value.Type().(type) {
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		var elements []tftypes.Value
		if err := value.As(&elements); err != nil {
			return nil, err
		}
		projected := make([]interface{}, len(elements))
		for i, element := range elements {
			var err error
			projected[i], err = ProjectTerraformValue(element)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
		}
		return projected, nil
	case tftypes.Map, tftypes.Object:
		var attributes map[string]tftypes.Value
		if err := value.As(&attributes); err != nil {
			return nil, err
		}
		projected := make(map[string]interface{}, len(attributes))
		for name, attribute := range attributes {
			var err error
			projected[name], err = ProjectTerraformValue(attribute)
			if err != nil {
				return nil, fmt.Errorf("attribute %q: %w", name, err)
			}
		}
		return projected, nil
	default:
		return nil, fmt.Errorf("unsupported Terraform type %T", value.Type())
	}
}

func typedValueJSON(value tftypes.Value) ([]byte, error) {
	projected, err := ProjectTerraformValue(value)
	if err != nil {
		return nil, err
	}
	if !value.IsFullyKnown() {
		return nil, fmt.Errorf("cannot serialize unknown value to state JSON")
	}
	return json.Marshal(collapseNullObjects(projected, true))
}

func collapseNullObjects(value interface{}, root bool) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		allNull := len(typed) > 0
		for name, child := range typed {
			typed[name] = collapseNullObjects(child, false)
			if typed[name] != nil {
				allNull = false
			}
		}
		if allNull && !root {
			return nil
		}
		return typed
	case []interface{}:
		for i, child := range typed {
			typed[i] = collapseNullObjects(child, false)
		}
		return typed
	default:
		return value
	}
}

func cleanupProjection(value interface{}, path string, ignored, allowedEmpty []*regexp.Regexp) (interface{}, bool) {
	for _, pattern := range ignored {
		if pattern.MatchString(path) {
			return nil, false
		}
	}
	allowEmpty := false
	for _, pattern := range allowedEmpty {
		if pattern.MatchString(path) {
			allowEmpty = true
			break
		}
	}

	switch typed := value.(type) {
	case UnknownValue, nil:
		return nil, false
	case string:
		return typed, typed != "" || allowEmpty
	case []interface{}:
		cleaned := make([]interface{}, 0, len(typed))
		for i, element := range typed {
			child, keep := cleanupProjection(element, joinProjectionPath(path, strconv.Itoa(i)), ignored, allowedEmpty)
			if keep {
				cleaned = append(cleaned, child)
			}
		}
		return cleaned, len(cleaned) > 0 || allowEmpty
	case map[string]interface{}:
		cleaned := make(map[string]interface{}, len(typed))
		for name, attribute := range typed {
			child, keep := cleanupProjection(attribute, joinProjectionPath(path, name), ignored, allowedEmpty)
			if keep {
				cleaned[name] = child
			}
		}
		return cleaned, len(cleaned) > 0 || path == "" || allowEmpty
	default:
		return value, true
	}
}

func joinProjectionPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
