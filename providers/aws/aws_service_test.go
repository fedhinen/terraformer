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

package aws

import (
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
)

func TestBuildBaseConfigUsesServiceRegion(t *testing.T) {
	service := AWSService{}
	service.SetArgs(map[string]interface{}{
		"profile": "",
		"region":  "us-west-2",
	})

	config, err := service.buildBaseConfig()
	if err != nil {
		t.Fatalf("buildBaseConfig() returned an error: %v", err)
	}
	if config.Region != "us-west-2" {
		t.Errorf("config.Region = %q, want %q", config.Region, "us-west-2")
	}
}

func TestBuildBaseConfigUsesPublicRegionForGlobalServices(t *testing.T) {
	service := AWSService{}
	service.SetArgs(map[string]interface{}{
		"profile": "",
		"region":  GlobalRegion,
	})

	config, err := service.buildBaseConfig()
	if err != nil {
		t.Fatalf("buildBaseConfig() returned an error: %v", err)
	}
	if config.Region != MainRegionPublicPartition {
		t.Errorf("config.Region = %q, want %q", config.Region, MainRegionPublicPartition)
	}
}

func TestInitDoesNotOverrideRegionEnvironment(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-2")

	provider := AWSProvider{}
	if err := provider.Init([]string{"us-west-2", ""}); err != nil {
		t.Fatalf("Init() returned an error: %v", err)
	}
	if region := os.Getenv("AWS_REGION"); region != "eu-west-1" {
		t.Errorf("AWS_REGION = %q, want %q", region, "eu-west-1")
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "eu-west-2" {
		t.Errorf("AWS_DEFAULT_REGION = %q, want %q", region, "eu-west-2")
	}
}

func TestEc2PostConvertHookRemovesPrimaryNetworkInterface(t *testing.T) {
	resource := terraformutils.NewSimpleResource("i-123", "fixture", "aws_instance", "aws", nil)
	resource.Item = map[string]interface{}{
		"associate_public_ip_address": "false",
		"primary_network_interface": []interface{}{
			map[string]interface{}{"network_interface_id": "eni-123"},
		},
	}

	generator := Ec2Generator{}
	generator.Resources = []terraformutils.Resource{resource}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() returned an error: %v", err)
	}
	if _, ok := generator.Resources[0].Item["primary_network_interface"]; ok {
		t.Error("PostConvertHook() retained primary_network_interface")
	}
}
