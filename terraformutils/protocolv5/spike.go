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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/compatibility"
	proto "github.com/GoogleCloudPlatform/terraformer/terraformutils/protocolv5/internal/tfplugin5"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/grpc"
)

var handshake = plugin.HandshakeConfig{
	ProtocolVersion:  4,
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

// Result summarizes a successful end-to-end protocol-v5 spike.
type Result struct {
	ResourceType  string
	SchemaVersion int64
	ImportedCount int
	Value         tftypes.Value
	Private       []byte
}

type grpcPlugin struct{ plugin.NetRPCUnsupportedPlugin }

func (*grpcPlugin) GRPCServer(*plugin.GRPCBroker, *grpc.Server) error {
	return errors.New("protocol-v5 spike is client-only")
}

func (*grpcPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return proto.NewProviderClient(conn), nil
}

// SpikeClient launches and calls one provider binary. It is intentionally
// separate from ProviderWrapper until the transport ADR is accepted.
type SpikeClient struct {
	client   *plugin.Client
	provider proto.ProviderClient
	once     sync.Once
	closeErr error
}

func Launch(ctx context.Context, providerBinary string) (*SpikeClient, error) {
	return LaunchWithLogging(ctx, providerBinary, false)
}

// LaunchWithLogging starts a protocol-v5 provider and optionally forwards its
// trace logs to stderr.
func LaunchWithLogging(ctx context.Context, providerBinary string, verbose bool) (*SpikeClient, error) {
	logger := hclog.NewNullLogger()
	if verbose {
		logger = hclog.New(&hclog.LoggerOptions{Name: "provider", Level: hclog.Trace, Output: os.Stderr})
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		Cmd:              exec.CommandContext(ctx, providerBinary),
		HandshakeConfig:  handshake,
		VersionedPlugins: map[int]plugin.PluginSet{5: {"provider": &grpcPlugin{}}},
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Managed:          true,
		AutoMTLS:         true,
		Logger:           logger,
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("starting protocol-v5 provider: %w", err)
	}
	raw, err := rpcClient.Dispense("provider")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("dispensing provider: %w", err)
	}
	provider, ok := raw.(proto.ProviderClient)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("unexpected provider client %T", raw)
	}
	return &SpikeClient{client: client, provider: provider}, nil
}

func (c *SpikeClient) Close() error {
	c.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		response, err := c.provider.Stop(ctx, &proto.Stop_Request{})
		if err != nil {
			c.closeErr = fmt.Errorf("stopping provider: %w", err)
		} else if response.Error != "" {
			c.closeErr = errors.New(response.Error)
		}
		c.client.Kill()
	})
	return c.closeErr
}

// Run performs schema, prepare, configure, import, read, and typed decode.
func (c *SpikeClient) Run(ctx context.Context, resourceType, id string) (*Result, error) {
	schemas := new(getSchemaResponse)
	var err error
	schemas, err = c.provider.GetSchema(ctx, &empty{})
	if err != nil {
		return nil, fmt.Errorf("GetSchema: %w", err)
	}
	if err := diagnosticsError(schemas.Diagnostics); err != nil {
		return nil, fmt.Errorf("GetSchema: %w", err)
	}
	resourceSchema, ok := schemas.ResourceSchemas[resourceType]
	if !ok || resourceSchema.Block == nil {
		return nil, fmt.Errorf("provider has no schema for %q", resourceType)
	}
	providerType, err := blockType(schemas.Provider.Block)
	if err != nil {
		return nil, fmt.Errorf("provider schema: %w", err)
	}
	config, err := nullObject(providerType)
	if err != nil {
		return nil, err
	}
	encodedConfig, err := tfprotov5.NewDynamicValue(providerType, config)
	if err != nil {
		return nil, err
	}
	wireConfig := fromDynamicValue(encodedConfig)
	prepared, err := c.provider.PrepareProviderConfig(ctx, &prepareRequest{Config: wireConfig})
	if err != nil {
		return nil, fmt.Errorf("PrepareProviderConfig: %w", err)
	}
	if err := diagnosticsError(prepared.Diagnostics); err != nil {
		return nil, fmt.Errorf("PrepareProviderConfig: %w", err)
	}
	if prepared.PreparedConfig != nil {
		wireConfig = prepared.PreparedConfig
	}
	configured, err := c.provider.Configure(ctx, &configureRequest{TerraformVersion: compatibility.TerraformVersion, Config: wireConfig})
	if err != nil {
		return nil, fmt.Errorf("Configure: %w", err)
	}
	if err := diagnosticsError(configured.Diagnostics); err != nil {
		return nil, fmt.Errorf("Configure: %w", err)
	}
	imported, err := c.provider.ImportResourceState(ctx, &importRequest{TypeName: resourceType, Id: id})
	if err != nil {
		return nil, fmt.Errorf("ImportResourceState: %w", err)
	}
	if err := diagnosticsError(imported.Diagnostics); err != nil {
		return nil, fmt.Errorf("ImportResourceState: %w", err)
	}
	if len(imported.ImportedResources) == 0 {
		return nil, errors.New("provider returned no imported resources")
	}
	first := imported.ImportedResources[0]
	read, err := c.provider.ReadResource(ctx, &readRequest{TypeName: first.TypeName, CurrentState: first.State, Private: first.Private})
	if err != nil {
		return nil, fmt.Errorf("ReadResource: %w", err)
	}
	if err := diagnosticsError(read.Diagnostics); err != nil {
		return nil, fmt.Errorf("ReadResource: %w", err)
	}
	resourceTFType, err := blockType(resourceSchema.Block)
	if err != nil {
		return nil, fmt.Errorf("resource schema: %w", err)
	}
	value, err := toDynamicValue(read.NewState).Unmarshal(resourceTFType)
	if err != nil {
		return nil, fmt.Errorf("decoding ReadResource state: %w", err)
	}
	return &Result{ResourceType: resourceType, SchemaVersion: resourceSchema.Version, ImportedCount: len(imported.ImportedResources), Value: value, Private: read.Private}, nil
}

func diagnosticsError(diagnostics []*diagnostic) error {
	var messages []error
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == proto.Diagnostic_ERROR {
			messages = append(messages, fmt.Errorf("%s: %s", diagnostic.Summary, diagnostic.Detail))
		}
	}
	return errors.Join(messages...)
}

func fromDynamicValue(value tfprotov5.DynamicValue) *dynamicValue {
	return &dynamicValue{Msgpack: value.MsgPack, Json: value.JSON}
}

func toDynamicValue(value *dynamicValue) tfprotov5.DynamicValue {
	if value == nil {
		return tfprotov5.DynamicValue{}
	}
	return tfprotov5.DynamicValue{MsgPack: value.Msgpack, JSON: value.Json}
}

func blockType(block *schemaBlock) (tftypes.Type, error) {
	if block == nil {
		return nil, errors.New("missing schema block")
	}
	attributes := make(map[string]tftypes.Type, len(block.Attributes)+len(block.BlockTypes))
	for _, attribute := range block.Attributes {
		typ, err := tftypes.ParseJSONType(attribute.Type)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", attribute.Name, err)
		}
		attributes[attribute.Name] = typ
	}
	for _, nested := range block.BlockTypes {
		element, err := blockType(nested.Block)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", nested.TypeName, err)
		}
		switch nested.Nesting {
		case 1, 5:
			attributes[nested.TypeName] = element
		case 2:
			attributes[nested.TypeName] = tftypes.List{ElementType: element}
		case 3:
			attributes[nested.TypeName] = tftypes.Set{ElementType: element}
		case 4:
			attributes[nested.TypeName] = tftypes.Map{ElementType: element}
		default:
			return nil, fmt.Errorf("block %q has unsupported nesting %d", nested.TypeName, nested.Nesting)
		}
	}
	return tftypes.Object{AttributeTypes: attributes}, nil
}

func nullObject(typ tftypes.Type) (tftypes.Value, error) {
	object, ok := typ.(tftypes.Object)
	if !ok {
		return tftypes.Value{}, fmt.Errorf("provider config type is %T, want object", typ)
	}
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tftypes.NewValue(object, values), nil
}
