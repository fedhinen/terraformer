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
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
)

// awsConflictsRuntime was generated from hashicorp/aws v6.58.0.
//
//go:embed assets/aws_conflicts_runtime.json
var awsConflictsRuntime []byte

type conflictRule struct {
	AtLeastOneOf  []string    `json:"at_least_one_of"`
	Computed      bool        `json:"computed"`
	ConflictsWith []string    `json:"conflicts_with"`
	Deprecated    string      `json:"deprecated"`
	Optional      bool        `json:"optional"`
	ExactlyOneOf  []string    `json:"exactly_one_of"`
	ForceNew      bool        `json:"force_new"`
	Required      bool        `json:"required"`
	RequiredWith  []string    `json:"required_with"`
	Default       interface{} `json:"default"`
}

type conflictCatalog map[string]map[string]conflictRule

var (
	awsConflictCatalog     conflictCatalog
	awsConflictCatalogErr  error
	awsConflictCatalogOnce sync.Once
)

func loadAWSConflictCatalog() (conflictCatalog, error) {
	awsConflictCatalogOnce.Do(func() {
		awsConflictCatalog = conflictCatalog{}
		awsConflictCatalogErr = json.Unmarshal(awsConflictsRuntime, &awsConflictCatalog)
	})
	return awsConflictCatalog, awsConflictCatalogErr
}

// ValidateGeneratedResources validates the final AWS configuration before it
// is rendered. Optional+computed attributes are state-derived values and are
// omitted unless needed to satisfy an at_least_one_of constraint. It only
// removes other attributes when deprecation makes the choice unambiguous.
func (p *AWSProvider) ValidateGeneratedResources(resources []terraformutils.Resource) error {
	catalog, err := loadAWSConflictCatalog()
	if err != nil {
		return fmt.Errorf("load AWS v6.58 conflict catalog: %w", err)
	}

	var diagnostics []string
	for i := range resources {
		resource := &resources[i]
		rules, ok := catalog[resource.InstanceInfo.Type]
		if !ok || resource.Item == nil {
			continue
		}
		diagnostics = append(diagnostics, validateAWSResource(resource, rules)...)
	}
	if len(diagnostics) == 0 {
		return nil
	}
	sort.Strings(diagnostics)
	return fmt.Errorf("AWS generated configuration violates hashicorp/aws v6.58 constraints:\n%s", strings.Join(diagnostics, "\n"))
}

func validateAWSResource(resource *terraformutils.Resource, rules map[string]conflictRule) []string {
	var diagnostics []string
	processedGroups := map[string]bool{}
	pruneOptionalComputed(resource.Item, rules)

	for path, rule := range rules {
		if len(rule.ConflictsWith) > 0 {
			for _, conflictingPath := range rule.ConflictsWith {
				pair := canonicalPathGroup([]string{path, conflictingPath})
				if processedGroups[pair] || !hasConfiguredValue(resource.Item, path) || !hasConfiguredValue(resource.Item, conflictingPath) {
					continue
				}
				processedGroups[pair] = true
				if removeDeprecatedConflict(resource.Item, path, conflictingPath, rules) {
					continue
				}
				diagnostics = append(diagnostics, conflictDiagnostic(resource, "conflicts_with", []string{path, conflictingPath}))
			}
		}

		if len(rule.ExactlyOneOf) > 0 {
			group := canonicalPathGroup(rule.ExactlyOneOf)
			if !processedGroups[group] {
				processedGroups[group] = true
				present := configuredPaths(resource.Item, rule.ExactlyOneOf)
				if len(present) > 1 {
					if !removeDeprecatedFromGroup(resource.Item, present, rules) {
						diagnostics = append(diagnostics, conflictDiagnostic(resource, "exactly_one_of", present))
					}
				}
			}
		}

		if len(rule.AtLeastOneOf) > 0 {
			group := canonicalPathGroup(rule.AtLeastOneOf)
			if !processedGroups[group] && len(configuredPaths(resource.Item, rule.AtLeastOneOf)) == 0 {
				processedGroups[group] = true
				diagnostics = append(diagnostics, conflictDiagnostic(resource, "at_least_one_of", rule.AtLeastOneOf))
			}
		}

		if len(rule.RequiredWith) > 0 && hasConfiguredValue(resource.Item, path) {
			missing := []string{}
			for _, requiredPath := range rule.RequiredWith {
				if !hasConfiguredValue(resource.Item, requiredPath) {
					missing = append(missing, requiredPath)
				}
			}
			if len(missing) > 0 {
				diagnostics = append(diagnostics, fmt.Sprintf("%s.%s: required_with requires %s", resource.InstanceInfo.Type, resource.ResourceName, strings.Join(missing, ", ")))
			}
		}
	}

	return diagnostics
}

// pruneOptionalComputed omits values which exist only because the provider
// returned them in state. Required attributes and catalog entries without both
// flags are intentionally left untouched. An optional+computed value remains
// when it is the only configured member of one of its required OR groups.
func pruneOptionalComputed(item map[string]interface{}, rules map[string]conflictRule) {
	paths := make([]string, 0, len(rules))
	for path := range rules {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		rule := rules[path]
		if rule.Required || !rule.Optional || !rule.Computed || !hasConfiguredValue(item, path) {
			continue
		}
		if isOnlyConfiguredAtLeastOneMember(item, path, rule.AtLeastOneOf) {
			continue
		}
		removeConfiguredPath(item, path)
	}
}

func isOnlyConfiguredAtLeastOneMember(item map[string]interface{}, path string, group []string) bool {
	if len(group) == 0 {
		return false
	}
	for _, groupPath := range group {
		if !sameSchemaPath(groupPath, path) && hasConfiguredValue(item, groupPath) {
			return false
		}
	}
	return hasConfiguredValue(item, path)
}

func removeDeprecatedConflict(item map[string]interface{}, first, second string, rules map[string]conflictRule) bool {
	firstDeprecated := ruleForPath(rules, first).Deprecated != ""
	secondDeprecated := ruleForPath(rules, second).Deprecated != ""
	switch {
	case firstDeprecated && !secondDeprecated:
		removeConfiguredPath(item, first)
		return true
	case secondDeprecated && !firstDeprecated:
		removeConfiguredPath(item, second)
		return true
	default:
		return false
	}
}

func removeDeprecatedFromGroup(item map[string]interface{}, present []string, rules map[string]conflictRule) bool {
	deprecated := []string{}
	for _, path := range present {
		if ruleForPath(rules, path).Deprecated != "" {
			deprecated = append(deprecated, path)
		}
	}
	if len(deprecated) == 0 || len(deprecated) == len(present) {
		return false
	}
	for _, path := range deprecated {
		removeConfiguredPath(item, path)
	}
	return true
}

func conflictDiagnostic(resource *terraformutils.Resource, kind string, paths []string) string {
	orderedPaths := append([]string(nil), paths...)
	sort.Strings(orderedPaths)
	return fmt.Sprintf("%s.%s: %s violated by %s", resource.InstanceInfo.Type, resource.ResourceName, kind, strings.Join(orderedPaths, ", "))
}

func ruleForPath(rules map[string]conflictRule, path string) conflictRule {
	if rule, ok := rules[path]; ok {
		return rule
	}
	return rules[normalizeSchemaPath(path)]
}

func sameSchemaPath(first, second string) bool {
	return normalizeSchemaPath(first) == normalizeSchemaPath(second)
}

// Nested attributes are keyed without indexes by the dumper, while the SDK's
// relationship rules commonly use Terraform's .0. list-index notation.
func normalizeSchemaPath(path string) string {
	segments := strings.Split(path, ".")
	normalized := make([]string, 0, len(segments))
	for _, segment := range segments {
		if _, err := strconv.Atoi(segment); err == nil {
			continue
		}
		normalized = append(normalized, segment)
	}
	return strings.Join(normalized, ".")
}

func canonicalPathGroup(paths []string) string {
	copyPaths := append([]string(nil), paths...)
	sort.Strings(copyPaths)
	return strings.Join(copyPaths, "\x00")
}

func configuredPaths(item map[string]interface{}, paths []string) []string {
	present := []string{}
	for _, path := range paths {
		if hasConfiguredValue(item, path) {
			present = append(present, path)
		}
	}
	return present
}

func hasConfiguredValue(item map[string]interface{}, path string) bool {
	for _, value := range valuesAtPath(item, strings.Split(path, ".")) {
		if isConfiguredValue(value) {
			return true
		}
	}
	return false
}

func valuesAtPath(value interface{}, path []string) []interface{} {
	if len(path) == 0 {
		return []interface{}{value}
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		next, ok := typed[path[0]]
		if !ok {
			return nil
		}
		return valuesAtPath(next, path[1:])
	case []interface{}:
		if index, err := strconv.Atoi(path[0]); err == nil {
			if index < 0 || index >= len(typed) {
				return nil
			}
			return valuesAtPath(typed[index], path[1:])
		}
		var values []interface{}
		for _, nestedValue := range typed {
			values = append(values, valuesAtPath(nestedValue, path)...)
		}
		return values
	default:
		return nil
	}
}

func isConfiguredValue(value interface{}) bool {
	if value == nil {
		return false
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.String, reflect.Array:
		return reflected.Len() > 0
	case reflect.Slice:
		for i := 0; i < reflected.Len(); i++ {
			if isConfiguredValue(reflected.Index(i).Interface()) {
				return true
			}
		}
		return false
	case reflect.Map:
		for _, key := range reflected.MapKeys() {
			if isConfiguredValue(reflected.MapIndex(key).Interface()) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func removeConfiguredPath(value interface{}, path string) {
	removePath(value, strings.Split(path, "."))
}

func removePath(value interface{}, path []string) {
	if len(path) == 0 {
		return
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		if len(path) == 1 {
			delete(typed, path[0])
			return
		}
		next, ok := typed[path[0]]
		if ok {
			removePath(next, path[1:])
		}
	case []interface{}:
		if index, err := strconv.Atoi(path[0]); err == nil {
			if index >= 0 && index < len(typed) {
				removePath(typed[index], path[1:])
			}
			return
		}
		for _, nestedValue := range typed {
			removePath(nestedValue, path)
		}
	}
}
