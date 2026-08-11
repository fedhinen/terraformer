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

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// ProviderClient is Terraformer's protocol-facing boundary. Implementations
// own transport and process lifecycle; provider and output packages consume
// only the public protocol model.
type ProviderClient interface {
	GetProviderSchema(context.Context, *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error)
	PrepareProviderConfig(context.Context, *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error)
	ConfigureProvider(context.Context, *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error)
	ReadResource(context.Context, *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error)
	ImportResourceState(context.Context, *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error)
	Close() error
}
