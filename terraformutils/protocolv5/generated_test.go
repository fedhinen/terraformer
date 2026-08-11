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
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestCanonicalProtocolFiles(t *testing.T) {
	expected := map[string]string{
		"tfplugin5.proto":      "3d9975526164cce8479755469220d54e7400e461b24a21997ccccef79e3f1e90",
		"tfplugin5.pb.go":      "b36b11ab6ea0ebe8ecbdad93a26b9026cf7298460d6182120a6ef6473290d083",
		"tfplugin5_grpc.pb.go": "1b572a4c9eb0edc01c3c609f86c0d270509f886418d6c9fdaa0fdd3751027c7d",
	}
	for name, want := range expected {
		contents, err := os.ReadFile("internal/tfplugin5/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(contents)); got != want {
			t.Fatalf("%s checksum = %s, want %s", name, got, want)
		}
	}
}
