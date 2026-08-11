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

package providerwrapper //nolint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/compatibility"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/protocolv5"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils/terraformerstring"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

// DefaultPluginVendorDir is the location in the config directory to look for
// user-added plugin binaries. Terraform only reads from this path if it
// exists, it is never created by terraform.
const DefaultPluginVendorDirV12 = "terraform.d/plugins/" + pluginMachineName

// pluginMachineName is the directory name used in new plugin paths.
const pluginMachineName = runtime.GOOS + "_" + runtime.GOARCH

type ProviderWrapper struct {
	client       protocolv5.ProviderClient
	providerName string
	config       cty.Value
	schema       *tfprotov5.GetProviderSchemaResponse
}

func NewProviderWrapper(providerName string, providerConfig cty.Value, verbose bool, options ...map[string]int) (*ProviderWrapper, error) {
	p := &ProviderWrapper{}
	p.providerName = providerName
	p.config = providerConfig

	// Kept in the signature while command-line retry flags are retired. Protocol
	// errors are not assumed to be transient and are never retried blindly.
	_ = options

	err := p.initProvider(verbose)

	return p, err
}

func (p *ProviderWrapper) Kill() {
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			log.Printf("closing provider %s: %v", p.providerName, err)
		}
	}
}

func (p *ProviderWrapper) GetSchema() *tfprotov5.GetProviderSchemaResponse {
	return p.schema
}

func (p *ProviderWrapper) GetReadOnlyAttributes(resourceTypes []string) (map[string][]string, error) {
	r := p.GetSchema()

	if err := diagnosticError("get provider schema", r.Diagnostics); err != nil {
		return nil, err
	}
	readOnlyAttributes := map[string][]string{}
	for resourceName, obj := range r.ResourceSchemas {
		if terraformerstring.ContainsString(resourceTypes, resourceName) {
			readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^id$")
			for _, v := range obj.Block.Attributes {
				if !v.Optional && !v.Required {
					if v.Type.Is(tftypes.List{}) || v.Type.Is(tftypes.Set{}) {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+v.Name+"\\.(.*)")
					} else {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+v.Name+"$")
					}
				}
			}
			readOnlyAttributes[resourceName] = p.readObjBlocks(obj.Block.BlockTypes, readOnlyAttributes[resourceName], "-1")
		}
	}
	return readOnlyAttributes, nil
}

func (p *ProviderWrapper) readObjBlocks(block []*tfprotov5.SchemaNestedBlock, readOnlyAttributes []string, parent string) []string {
	for _, v := range block {
		k := v.TypeName
		if len(v.Block.BlockTypes) > 0 {
			if parent == "-1" {
				readOnlyAttributes = p.readObjBlocks(v.Block.BlockTypes, readOnlyAttributes, k)
			} else {
				readOnlyAttributes = p.readObjBlocks(v.Block.BlockTypes, readOnlyAttributes, parent+"\\.[0-9]+\\."+k)
			}
		}
		fieldCount := 0
		for _, l := range v.Block.Attributes {
			if !l.Optional && !l.Required {
				fieldCount++
				key := l.Name
				switch v.Nesting {
				case tfprotov5.SchemaNestedBlockNestingModeList:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"($|\\.[0-9]+|\\.#)")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"$")
					}
				case tfprotov5.SchemaNestedBlockNestingModeSet:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"$")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"($|\\.(.*))")
					}
				case tfprotov5.SchemaNestedBlockNestingModeMap:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key)
				default:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key+"$")
				}
			}
		}
		if fieldCount == len(v.Block.Attributes) && fieldCount > 0 && len(v.Block.BlockTypes) == 0 {
			readOnlyAttributes = append(readOnlyAttributes, "^"+k)
		}
	}
	return readOnlyAttributes
}

// RefreshValue reads existing state once and falls back to the provider's
// import RPC only when the read returns error diagnostics.
func (p *ProviderWrapper) RefreshValue(resourceType, id string, prior tftypes.Value, private []byte) (tftypes.Value, []byte, error) {
	resourceSchema, ok := p.schema.ResourceSchemas[resourceType]
	if !ok {
		return tftypes.Value{}, nil, fmt.Errorf("provider %s has no schema for %s", p.providerName, resourceType)
	}
	dynamic, err := tfprotov5.NewDynamicValue(resourceSchema.ValueType(), prior)
	if err != nil {
		return tftypes.Value{}, nil, fmt.Errorf("encode prior state for %s: %w", resourceType, err)
	}
	resp, err := p.client.ReadResource(context.Background(), &tfprotov5.ReadResourceRequest{TypeName: resourceType, CurrentState: &dynamic, Private: private})
	if err != nil {
		return tftypes.Value{}, nil, fmt.Errorf("read %s %q: %w", resourceType, id, err)
	}
	if diagnosticError("read resource", resp.Diagnostics) != nil {
		importResponse, importErr := p.client.ImportResourceState(context.Background(), &tfprotov5.ImportResourceStateRequest{TypeName: resourceType, ID: id})
		if importErr != nil {
			return tftypes.Value{}, nil, fmt.Errorf("import %s %q after read diagnostics: %w", resourceType, id, importErr)
		}
		if err := diagnosticError("import resource", importResponse.Diagnostics); err != nil {
			return tftypes.Value{}, nil, err
		}
		if len(importResponse.ImportedResources) == 0 {
			return tftypes.Value{}, nil, errors.New("provider returned no resources for the given import ID")
		}
		if len(importResponse.ImportedResources) != 1 {
			return tftypes.Value{}, nil, fmt.Errorf("provider returned %d resources for %s %q; Terraformer's resource model cannot represent a multi-resource import", len(importResponse.ImportedResources), resourceType, id)
		}
		imported := importResponse.ImportedResources[0]
		value, err := imported.State.Unmarshal(resourceSchema.ValueType())
		return value, imported.Private, err
	}
	if err := diagnosticError("read resource", resp.Diagnostics); err != nil {
		return tftypes.Value{}, nil, err
	}
	value, err := resp.NewState.Unmarshal(resourceSchema.ValueType())
	if err != nil {
		return tftypes.Value{}, nil, fmt.Errorf("decode state for %s %q: %w", resourceType, id, err)
	}
	if value.IsNull() {
		return tftypes.Value{}, nil, fmt.Errorf("read resource response is null for %s %q", resourceType, id)
	}
	return value, resp.Private, nil
}

func (p *ProviderWrapper) initProvider(verbose bool) error {
	providerFilePath, err := getProviderFileName(p.providerName)
	if err != nil {
		return err
	}
	p.client, err = protocolv5.LaunchWithLogging(context.Background(), providerFilePath, verbose)
	if err != nil {
		return err
	}
	p.schema, err = p.client.GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		return err
	}
	if err := diagnosticError("get provider schema", p.schema.Diagnostics); err != nil {
		return err
	}
	config, err := providerConfigValue(p.config, p.schema.Provider.ValueType())
	if err != nil {
		return err
	}
	dynamic, err := tfprotov5.NewDynamicValue(p.schema.Provider.ValueType(), config)
	if err != nil {
		return err
	}
	prepared, err := p.client.PrepareProviderConfig(context.Background(), &tfprotov5.PrepareProviderConfigRequest{Config: &dynamic})
	if err != nil {
		return err
	}
	if err := diagnosticError("prepare provider config", prepared.Diagnostics); err != nil {
		return err
	}
	preparedConfig := prepared.PreparedConfig
	if preparedConfig == nil {
		preparedConfig = &dynamic
	}
	configured, err := p.client.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{TerraformVersion: compatibility.TerraformVersion, Config: preparedConfig})
	if err != nil {
		return err
	}
	return diagnosticError("configure provider", configured.Diagnostics)
}

func providerConfigValue(config cty.Value, typ tftypes.Type) (tftypes.Value, error) {
	encoded, err := ctyjson.Marshal(config, config.Type())
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("encode provider config: %w", err)
	}
	var supplied map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &supplied); err != nil {
		return tftypes.Value{}, fmt.Errorf("provider config must be an object: %w", err)
	}
	objectType, ok := typ.(tftypes.Object)
	if !ok {
		return tftypes.Value{}, fmt.Errorf("provider schema must be an object, got %T", typ)
	}
	complete := make(map[string]json.RawMessage, len(objectType.AttributeTypes))
	for name := range objectType.AttributeTypes {
		complete[name] = json.RawMessage("null")
	}
	for name, value := range supplied {
		if _, ok := complete[name]; ok {
			complete[name] = value
		}
	}
	encoded, err = json.Marshal(complete)
	if err != nil {
		return tftypes.Value{}, err
	}
	return tftypes.ValueFromJSON(encoded, typ) //nolint:staticcheck
}

func diagnosticError(operation string, diagnostics []*tfprotov5.Diagnostic) error {
	messages := make([]string, 0)
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != tfprotov5.DiagnosticSeverityError {
			continue
		}
		message := diagnostic.Summary
		if diagnostic.Detail != "" {
			message += ": " + diagnostic.Detail
		}
		if diagnostic.Attribute != nil {
			message += fmt.Sprintf(" (attribute %v)", diagnostic.Attribute)
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", operation, strings.Join(messages, "; "))
}

func getProviderFileName(providerName string) (string, error) {
	defaultDataDir := os.Getenv("TF_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = DefaultDataDir
	}
	providerFilePath, err := getProviderFileNameV13andV14(defaultDataDir, providerName)
	if err != nil || providerFilePath == "" {
		providerFilePath, err = getProviderFileNameV13andV14(os.Getenv("HOME")+string(os.PathSeparator)+
			".terraform.d", providerName)
	}
	if err != nil || providerFilePath == "" {
		return getProviderFileNameV12(providerName)
	}
	return providerFilePath, nil
}

func getProviderFileNameV13andV14(prefix, providerName string) (string, error) {
	type candidate struct{ version, path string }
	var selected candidate
	foundRegistry := false
	for _, layout := range []string{"providers", "plugins", "plugin-cache"} {
		registryDir := filepath.Join(prefix, layout, "registry.terraform.io")
		namespaces, err := os.ReadDir(registryDir)
		if err != nil {
			continue
		}
		foundRegistry = true
		for _, namespace := range namespaces {
			versions, err := os.ReadDir(filepath.Join(registryDir, namespace.Name(), providerName))
			if err != nil {
				continue
			}
			for _, version := range versions {
				if !version.IsDir() {
					continue
				}
				machineDir := filepath.Join(registryDir, namespace.Name(), providerName, version.Name(), pluginMachineName)
				files, err := os.ReadDir(machineDir)
				if err != nil {
					continue
				}
				for _, file := range files {
					if file.IsDir() || !strings.HasPrefix(file.Name(), "terraform-provider-"+providerName) {
						continue
					}
					current := candidate{version: version.Name(), path: filepath.Join(machineDir, file.Name())}
					if selected.path == "" || compareProviderVersions(current.version, selected.version) > 0 {
						selected = current
					}
				}
			}
		}
	}
	if selected.path != "" {
		return selected.path, nil
	}
	if !foundRegistry {
		return "", os.ErrNotExist
	}
	return "", nil
}

func compareProviderVersions(left, right string) int {
	leftParts, rightParts := strings.Split(strings.TrimPrefix(left, "v"), "."), strings.Split(strings.TrimPrefix(right, "v"), ".")
	length := len(leftParts)
	if len(rightParts) > length {
		length = len(rightParts)
	}
	for i := 0; i < length; i++ {
		var leftNumber, rightNumber int
		if i < len(leftParts) {
			leftNumber, _ = strconv.Atoi(strings.SplitN(leftParts[i], "-", 2)[0])
		}
		if i < len(rightParts) {
			rightNumber, _ = strconv.Atoi(strings.SplitN(rightParts[i], "-", 2)[0])
		}
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
	}
	return strings.Compare(left, right)
}

func getProviderFileNameV12(providerName string) (string, error) {
	defaultDataDir := os.Getenv("TF_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = DefaultDataDir
	}
	pluginPath := defaultDataDir + string(os.PathSeparator) + "plugins" + string(os.PathSeparator) + runtime.GOOS + "_" + runtime.GOARCH
	files, err := os.ReadDir(pluginPath)
	if err != nil {
		pluginPath = os.Getenv("HOME") + string(os.PathSeparator) + "." + DefaultPluginVendorDirV12
		files, err = os.ReadDir(pluginPath)
		if err != nil {
			return "", err
		}
	}
	providerFilePath := ""
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasPrefix(file.Name(), "terraform-provider-"+providerName) {
			providerFilePath = pluginPath + string(os.PathSeparator) + file.Name()
		}
	}
	return providerFilePath, nil
}

func GetProviderVersion(providerName string) string {
	providerFilePath, err := getProviderFileName(providerName)
	if err != nil {
		log.Println("Can't find provider file path. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#third-party-plugins.")
		return ""
	}
	t := strings.Split(providerFilePath, string(os.PathSeparator))
	providerFileName := t[len(t)-1]
	providerFileNameParts := strings.Split(providerFileName, "_")
	if len(providerFileNameParts) < 2 {
		log.Println("Can't find provider version. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#plugin-names-and-versions.")
		return ""
	}
	providerVersion := providerFileNameParts[1]
	return "~> " + strings.TrimPrefix(providerVersion, "v")
}
