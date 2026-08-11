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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
)

func TestValidateGeneratedResourcesRemovesDeprecatedConflict(t *testing.T) {
	resource := terraformutils.NewSimpleResource("i-123", "fixture", "aws_instance", "aws", nil)
	resource.Item = map[string]interface{}{
		"ami":           "ami-123",
		"instance_type": "t3.micro",
		"network_interface": []interface{}{
			map[string]interface{}{"network_interface_id": "eni-123"},
		},
		"subnet_id": "subnet-123",
	}

	provider := AWSProvider{}
	if err := provider.ValidateGeneratedResources([]terraformutils.Resource{resource}); err != nil {
		t.Fatalf("ValidateGeneratedResources() returned an error: %v", err)
	}
	if _, ok := resource.Item["network_interface"]; ok {
		t.Error("ValidateGeneratedResources() retained deprecated network_interface")
	}
}

func TestValidateGeneratedResourcesReportsAmbiguousConflict(t *testing.T) {
	resource := terraformutils.NewSimpleResource("certificate", "fixture", "aws_acm_certificate", "aws", nil)
	resource.Item = map[string]interface{}{
		"certificate_authority_arn": "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/123",
		"certificate_body":          "-----BEGIN CERTIFICATE-----",
	}

	err := (&AWSProvider{}).ValidateGeneratedResources([]terraformutils.Resource{resource})
	if err == nil {
		t.Fatal("ValidateGeneratedResources() returned nil, want conflict error")
	}
	if !strings.Contains(err.Error(), "conflicts_with violated by certificate_authority_arn, certificate_body") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateGeneratedResourcesReportsRequiredWith(t *testing.T) {
	resource := terraformutils.NewSimpleResource("certificate", "fixture", "aws_acm_certificate", "aws", nil)
	resource.Item = map[string]interface{}{
		"private_key_wo": "secret",
	}

	err := (&AWSProvider{}).ValidateGeneratedResources([]terraformutils.Resource{resource})
	if err == nil {
		t.Fatal("ValidateGeneratedResources() returned nil, want required_with error")
	}
	if !strings.Contains(err.Error(), "required_with requires private_key_wo_version") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHasConfiguredValueHandlesIndexedCatalogPath(t *testing.T) {
	item := map[string]interface{}{
		"launch_template": []interface{}{
			map[string]interface{}{"id": "lt-123"},
		},
	}
	if !hasConfiguredValue(item, "launch_template.0.id") {
		t.Error("hasConfiguredValue() = false, want true")
	}
}

func TestValidateAWSResourceOmitsOptionalComputedValue(t *testing.T) {
	resource := terraformutils.NewSimpleResource("id", "fixture", "aws_example", "aws", nil)
	resource.Item = map[string]interface{}{
		"state_derived": "provider default",
		"explicit":      "configured",
	}
	rules := map[string]conflictRule{
		"state_derived": {Optional: true, Computed: true},
		"explicit":      {Optional: true},
	}

	if diagnostics := validateAWSResource(&resource, rules); len(diagnostics) != 0 {
		t.Fatalf("validateAWSResource() diagnostics = %v, want none", diagnostics)
	}
	if _, ok := resource.Item["state_derived"]; ok {
		t.Error("validateAWSResource() retained optional+computed state value")
	}
	if _, ok := resource.Item["explicit"]; !ok {
		t.Error("validateAWSResource() removed optional configured value")
	}
}

func TestValidateAWSResourceKeepsOnlyAtLeastOneMember(t *testing.T) {
	resource := terraformutils.NewSimpleResource("id", "fixture", "aws_example", "aws", nil)
	resource.Item = map[string]interface{}{
		"first": "provider value",
	}
	rules := map[string]conflictRule{
		"first":  {Optional: true, Computed: true, AtLeastOneOf: []string{"first", "second"}},
		"second": {Optional: true, Computed: true, AtLeastOneOf: []string{"first", "second"}},
	}

	if diagnostics := validateAWSResource(&resource, rules); len(diagnostics) != 0 {
		t.Fatalf("validateAWSResource() diagnostics = %v, want none", diagnostics)
	}
	if _, ok := resource.Item["first"]; !ok {
		t.Error("validateAWSResource() removed the only at_least_one_of member")
	}
}

func TestRuleForPathNormalizesNestedIndexes(t *testing.T) {
	rules := map[string]conflictRule{
		"launch_template.id": {Deprecated: "use name instead"},
	}
	if rule := ruleForPath(rules, "launch_template.0.id"); rule.Deprecated == "" {
		t.Error("ruleForPath() did not match an indexed relationship path")
	}
}
