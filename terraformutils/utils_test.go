// Copyright 2018 The Terraformer Authors.
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
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestPrintTfStateWithProviderSourceWritesVersion4State(t *testing.T) {
	data, err := PrintTfStateWithProviderSource([]Resource{{
		ResourceName: "fixture",
		InstanceInfo: &terraform.InstanceInfo{Type: "aws_vpc", Id: "aws_vpc.fixture"},
		StateJSON:    []byte(`{"id":"vpc-123"}`),
	}}, "registry.terraform.io/hashicorp/aws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var state struct {
		Version   int `json:"version"`
		Resources []struct {
			Provider  string `json:"provider"`
			Instances []struct {
				Attributes map[string]string `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("state is not valid JSON: %v", err)
	}
	if state.Version != 4 {
		t.Fatalf("state version = %d, want 4", state.Version)
	}
	if got := state.Resources[0].Provider; got != `provider["registry.terraform.io/hashicorp/aws"]` {
		t.Fatalf("provider address = %q", got)
	}
	if got := state.Resources[0].Instances[0].Attributes["id"]; got != "vpc-123" {
		t.Fatalf("resource ID = %q", got)
	}
}
