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
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDiscoveryPriorValueUsesTypedDiscoveryData(t *testing.T) {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":      tftypes.String,
		"enabled": tftypes.Bool,
		"ports":   tftypes.List{ElementType: tftypes.Number},
		"missing": tftypes.String,
	}}
	value, err := discoveryPriorValue(typ, "fixture-id", nil, map[string]interface{}{
		"enabled": true,
		"ports":   []int{80, 443},
	})
	if err != nil {
		t.Fatal(err)
	}
	var attributes map[string]tftypes.Value
	if err := value.As(&attributes); err != nil {
		t.Fatal(err)
	}
	if attributes["id"].IsNull() || attributes["enabled"].IsNull() || attributes["ports"].IsNull() {
		t.Fatalf("discovery value lost attributes: %#v", attributes)
	}
	if !attributes["missing"].IsNull() {
		t.Fatalf("missing attribute = %v, want null", attributes["missing"])
	}
}
