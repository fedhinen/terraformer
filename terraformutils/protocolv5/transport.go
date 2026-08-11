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
	"fmt"

	proto "github.com/GoogleCloudPlatform/terraformer/terraformutils/protocolv5/internal/tfplugin5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (c *SpikeClient) GetProviderSchema(ctx context.Context, _ *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	response, err := c.provider.GetSchema(ctx, &proto.GetProviderSchema_Request{})
	if err != nil {
		return nil, err
	}
	resources, err := schemasFromProto(response.ResourceSchemas)
	if err != nil {
		return nil, err
	}
	provider, err := schemaFromProto(response.Provider)
	if err != nil {
		return nil, err
	}
	return &tfprotov5.GetProviderSchemaResponse{
		Provider:        provider,
		ResourceSchemas: resources,
		Diagnostics:     diagnosticsFromProto(response.Diagnostics),
	}, nil
}

func (c *SpikeClient) PrepareProviderConfig(ctx context.Context, request *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	response, err := c.provider.PrepareProviderConfig(ctx, &proto.PrepareProviderConfig_Request{Config: dynamicToProto(request.Config)})
	if err != nil {
		return nil, err
	}
	return &tfprotov5.PrepareProviderConfigResponse{PreparedConfig: dynamicFromProto(response.PreparedConfig), Diagnostics: diagnosticsFromProto(response.Diagnostics)}, nil
}

func (c *SpikeClient) ConfigureProvider(ctx context.Context, request *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	response, err := c.provider.Configure(ctx, &proto.Configure_Request{TerraformVersion: request.TerraformVersion, Config: dynamicToProto(request.Config)})
	if err != nil {
		return nil, err
	}
	return &tfprotov5.ConfigureProviderResponse{Diagnostics: diagnosticsFromProto(response.Diagnostics)}, nil
}

func (c *SpikeClient) ReadResource(ctx context.Context, request *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	response, err := c.provider.ReadResource(ctx, &proto.ReadResource_Request{TypeName: request.TypeName, CurrentState: dynamicToProto(request.CurrentState), Private: request.Private})
	if err != nil {
		return nil, err
	}
	return &tfprotov5.ReadResourceResponse{NewState: dynamicFromProto(response.NewState), Private: response.Private, Diagnostics: diagnosticsFromProto(response.Diagnostics)}, nil
}

func (c *SpikeClient) ImportResourceState(ctx context.Context, request *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	response, err := c.provider.ImportResourceState(ctx, &proto.ImportResourceState_Request{TypeName: request.TypeName, Id: request.ID})
	if err != nil {
		return nil, err
	}
	imported := make([]*tfprotov5.ImportedResource, 0, len(response.ImportedResources))
	for _, resource := range response.ImportedResources {
		imported = append(imported, &tfprotov5.ImportedResource{TypeName: resource.TypeName, State: dynamicFromProto(resource.State), Private: resource.Private})
	}
	return &tfprotov5.ImportResourceStateResponse{ImportedResources: imported, Diagnostics: diagnosticsFromProto(response.Diagnostics)}, nil
}

func dynamicToProto(value *tfprotov5.DynamicValue) *proto.DynamicValue {
	if value == nil {
		return nil
	}
	return &proto.DynamicValue{Msgpack: value.MsgPack, Json: value.JSON}
}

func dynamicFromProto(value *proto.DynamicValue) *tfprotov5.DynamicValue {
	if value == nil {
		return nil
	}
	return &tfprotov5.DynamicValue{MsgPack: value.Msgpack, JSON: value.Json}
}

func schemasFromProto(input map[string]*proto.Schema) (map[string]*tfprotov5.Schema, error) {
	output := make(map[string]*tfprotov5.Schema, len(input))
	for name, schema := range input {
		converted, err := schemaFromProto(schema)
		if err != nil {
			return nil, fmt.Errorf("schema %q: %w", name, err)
		}
		output[name] = converted
	}
	return output, nil
}

func schemaFromProto(input *proto.Schema) (*tfprotov5.Schema, error) {
	if input == nil {
		return nil, nil
	}
	block, err := schemaBlockFromProto(input.Block)
	if err != nil {
		return nil, err
	}
	return &tfprotov5.Schema{Version: input.Version, Block: block}, nil
}

func schemaBlockFromProto(input *proto.Schema_Block) (*tfprotov5.SchemaBlock, error) {
	if input == nil {
		return nil, nil
	}
	output := &tfprotov5.SchemaBlock{Version: input.Version, Description: input.Description, Deprecated: input.Deprecated, DeprecationMessage: input.DeprecationMessage}
	for _, attribute := range input.Attributes {
		typ, err := tftypes.ParseJSONType(attribute.Type)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", attribute.Name, err)
		}
		output.Attributes = append(output.Attributes, &tfprotov5.SchemaAttribute{
			Name: attribute.Name, Type: typ, Description: attribute.Description,
			Required: attribute.Required, Optional: attribute.Optional, Computed: attribute.Computed,
			Sensitive: attribute.Sensitive, Deprecated: attribute.Deprecated, WriteOnly: attribute.WriteOnly,
		})
	}
	for _, nested := range input.BlockTypes {
		block, err := schemaBlockFromProto(nested.Block)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", nested.TypeName, err)
		}
		output.BlockTypes = append(output.BlockTypes, &tfprotov5.SchemaNestedBlock{
			TypeName: nested.TypeName, Block: block, Nesting: tfprotov5.SchemaNestedBlockNestingMode(nested.Nesting),
			MinItems: nested.MinItems, MaxItems: nested.MaxItems,
		})
	}
	return output, nil
}

func diagnosticsFromProto(input []*proto.Diagnostic) []*tfprotov5.Diagnostic {
	output := make([]*tfprotov5.Diagnostic, 0, len(input))
	for _, diagnostic := range input {
		output = append(output, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverity(diagnostic.Severity), Summary: diagnostic.Summary,
			Detail: diagnostic.Detail, Attribute: attributePathFromProto(diagnostic.Attribute),
		})
	}
	return output
}

func attributePathFromProto(input *proto.AttributePath) *tftypes.AttributePath {
	if input == nil {
		return nil
	}
	path := tftypes.NewAttributePath()
	for _, step := range input.Steps {
		switch selector := step.Selector.(type) {
		case *proto.AttributePath_Step_AttributeName:
			path = path.WithAttributeName(selector.AttributeName)
		case *proto.AttributePath_Step_ElementKeyString:
			path = path.WithElementKeyString(selector.ElementKeyString)
		case *proto.AttributePath_Step_ElementKeyInt:
			path = path.WithElementKeyInt(int(selector.ElementKeyInt))
		}
	}
	return path
}

var _ ProviderClient = (*SpikeClient)(nil)
