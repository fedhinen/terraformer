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

package main

import (
	"context"
	"errors"
	"net/rpc"

	proto "github.com/GoogleCloudPlatform/terraformer/terraformutils/protocolv5/internal/tfplugin5"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/grpc"
)

var handshake = plugin.HandshakeConfig{ProtocolVersion: 4, MagicCookieKey: "TF_PLUGIN_MAGIC_COOKIE", MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2"}

type providerPlugin struct{ plugin.NetRPCUnsupportedPlugin }

func (*providerPlugin) GRPCClient(context.Context, *plugin.GRPCBroker, *grpc.ClientConn) (interface{}, error) {
	return nil, errors.New("test provider is server-only")
}
func (*providerPlugin) GRPCServer(_ *plugin.GRPCBroker, server *grpc.Server) error {
	proto.RegisterProviderServer(server, &providerServer{})
	return nil
}
func (*providerPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("net/rpc unsupported")
}

type providerServer struct {
	proto.UnimplementedProviderServer
}

func (*providerServer) GetSchema(context.Context, *proto.GetProviderSchema_Request) (*proto.GetProviderSchema_Response, error) {
	provider := &proto.Schema{Block: &proto.Schema_Block{}}
	resource := &proto.Schema{Version: 3, Block: &proto.Schema_Block{Attributes: []*proto.Schema_Attribute{
		{Name: "id", Type: []byte(`"string"`), Computed: true},
		{Name: "name", Type: []byte(`"string"`), Optional: true},
	}}}
	return &proto.GetProviderSchema_Response{Provider: provider, ResourceSchemas: map[string]*proto.Schema{"test_resource": resource}}, nil
}

func (*providerServer) PrepareProviderConfig(_ context.Context, request *proto.PrepareProviderConfig_Request) (*proto.PrepareProviderConfig_Response, error) {
	return &proto.PrepareProviderConfig_Response{PreparedConfig: request.Config}, nil
}

func (*providerServer) Configure(context.Context, *proto.Configure_Request) (*proto.Configure_Response, error) {
	return &proto.Configure_Response{}, nil
}

func (*providerServer) ImportResourceState(_ context.Context, request *proto.ImportResourceState_Request) (*proto.ImportResourceState_Response, error) {
	state, err := resourceState(request.Id)
	if err != nil {
		return nil, err
	}
	return &proto.ImportResourceState_Response{ImportedResources: []*proto.ImportResourceState_ImportedResource{
		{TypeName: request.TypeName, State: state, Private: []byte("private-one")},
		{TypeName: request.TypeName, State: state, Private: []byte("private-two")},
	}}, nil
}

func (*providerServer) ReadResource(_ context.Context, request *proto.ReadResource_Request) (*proto.ReadResource_Response, error) {
	return &proto.ReadResource_Response{NewState: request.CurrentState, Private: append(request.Private, []byte("-read")...)}, nil
}

func (*providerServer) Stop(context.Context, *proto.Stop_Request) (*proto.Stop_Response, error) {
	return &proto.Stop_Response{}, nil
}

func resourceState(id string) (*proto.DynamicValue, error) {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String, "name": tftypes.String}}
	value := tftypes.NewValue(typ, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, id),
		"name": tftypes.NewValue(tftypes.String, "fixture"),
	})
	dynamic, err := tfprotov5.NewDynamicValue(typ, value)
	if err != nil {
		return nil, err
	}
	return &proto.DynamicValue{Msgpack: dynamic.MsgPack, Json: dynamic.JSON}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  handshake,
		VersionedPlugins: map[int]plugin.PluginSet{5: {"provider": &providerPlugin{}}},
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}
