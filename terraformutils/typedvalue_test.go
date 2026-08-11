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
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProjectTerraformValuePreservesNullUnknownAndNesting(t *testing.T) {
	typeObject := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"null":    tftypes.String,
		"unknown": tftypes.Bool,
		"empty":   tftypes.List{ElementType: tftypes.String},
		"nested":  tftypes.Map{ElementType: tftypes.Number},
	}}
	value := tftypes.NewValue(typeObject, map[string]tftypes.Value{
		"null":    tftypes.NewValue(tftypes.String, nil),
		"unknown": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"empty":   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		"nested": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Number}, map[string]tftypes.Value{
			"answer": tftypes.NewValue(tftypes.Number, 42),
		}),
	})

	projected, err := ProjectTerraformValue(value)
	if err != nil {
		t.Fatal(err)
	}
	got := projected.(map[string]interface{})
	if got["null"] != nil {
		t.Fatalf("null projected as %#v", got["null"])
	}
	if _, ok := got["unknown"].(UnknownValue); !ok {
		t.Fatalf("unknown projected as %T", got["unknown"])
	}
	if !reflect.DeepEqual(got["empty"], []interface{}{}) {
		t.Fatalf("empty list projected as %#v", got["empty"])
	}
	if number, ok := got["nested"].(map[string]interface{})["answer"].(json.Number); !ok || number != "42" {
		t.Fatalf("number projected as %#v", got["nested"])
	}
}

func TestCleanupProjectionAppliesIgnoreAndEmptyRules(t *testing.T) {
	input := map[string]interface{}{
		"computed": "drop",
		"empty":    []interface{}{},
		"tags":     map[string]interface{}{},
		"unknown":  UnknownValue{},
	}
	got, _ := cleanupProjection(input, "", []*regexp.Regexp{regexp.MustCompile(`^computed$`)}, []*regexp.Regexp{regexp.MustCompile(`^tags`)})
	want := map[string]interface{}{"tags": map[string]interface{}{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanupProjection() = %#v, want %#v", got, want)
	}
}

func TestTypedValueJSONCollapsesAllNullNestedObject(t *testing.T) {
	nestedType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"create": tftypes.String, "delete": tftypes.String}}
	rootType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String, "timeouts": nestedType}}
	value := tftypes.NewValue(rootType, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "fixture"),
		"timeouts": tftypes.NewValue(nestedType, map[string]tftypes.Value{
			"create": tftypes.NewValue(tftypes.String, nil),
			"delete": tftypes.NewValue(tftypes.String, nil),
		}),
	})
	data, err := typedValueJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"id":"fixture","timeouts":null}` {
		t.Fatalf("state JSON = %s", data)
	}
}
