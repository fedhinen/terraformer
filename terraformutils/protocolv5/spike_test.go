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

package protocolv5

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBlockTypeIncludesNestedCollectionKinds(t *testing.T) {
	typ, err := blockType(&schemaBlock{
		Attributes: []*schemaAttribute{{Name: "name", Type: []byte(`"string"`)}},
		BlockTypes: []*schemaNestedBlock{
			{TypeName: "items", Nesting: 2, Block: &schemaBlock{Attributes: []*schemaAttribute{{Name: "enabled", Type: []byte(`"bool"`)}}}},
			{TypeName: "rules", Nesting: 3, Block: &schemaBlock{Attributes: []*schemaAttribute{{Name: "port", Type: []byte(`"number"`)}}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	object := typ.(tftypes.Object)
	if !object.AttributeTypes["items"].Is(tftypes.List{}) {
		t.Fatalf("items type = %s", object.AttributeTypes["items"])
	}
	if !object.AttributeTypes["rules"].Is(tftypes.Set{}) {
		t.Fatalf("rules type = %s", object.AttributeTypes["rules"])
	}
}

func TestSpikeContractProvider(t *testing.T) {
	if os.Getenv("TERRAFORMER_PROTOCOLV5_CONTRACT") == "" {
		t.Skip("set TERRAFORMER_PROTOCOLV5_CONTRACT=1 to run the subprocess contract test")
	}
	binary := filepath.Join(t.TempDir(), "terraform-provider-contract")
	command := exec.Command("go", "build", "-o", binary, "./testdata/provider")
	command.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("building contract provider: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := Launch(ctx, binary)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	result, err := client.Run(ctx, "test_resource", "fixture-id")
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedCount != 2 || string(result.Private) != "private-one-read" || result.SchemaVersion != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDiagnosticsErrorPreservesProviderDetails(t *testing.T) {
	err := diagnosticsError([]*diagnostic{{Severity: 2, Summary: "warning"}, {Severity: 1, Summary: "invalid config", Detail: "missing region"}})
	if err == nil || err.Error() != "invalid config: missing region" {
		t.Fatalf("diagnosticsError() = %v", err)
	}
}
