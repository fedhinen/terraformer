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

package terraformutils

import (
	"fmt"
	"log"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils/providerwrapper"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type Resource struct {
	InstanceInfo        *ResourceAddress
	InstanceState       *ResourceState
	Outputs             map[string]*OutputState `json:",omitempty"`
	ResourceName        string
	Provider            string
	Item                map[string]interface{} `json:",omitempty"`
	IgnoreKeys          []string               `json:",omitempty"`
	AllowEmptyValues    []string               `json:",omitempty"`
	AdditionalFields    map[string]interface{} `json:",omitempty"`
	SlowQueryRequired   bool
	DataFiles           map[string][]byte
	StateJSON           []byte
	StateSchemaVersion  uint64
	MapAttributes       []string               `json:"-"`
	DiscoveryAttributes map[string]interface{} `json:"-"`
}

// ResourceAddress identifies a Terraform resource independently of the
// provider transport used to refresh it.
type ResourceAddress struct {
	Type string
	Name string
	Id   string
}

// ResourceState is Terraformer's transport-independent resource state. Value
// is authoritative; Attributes contains provider-discovery metadata still
// consumed by existing filters and provider-specific cleanup hooks.
type ResourceState struct {
	ID            string
	Attributes    map[string]string
	Value         tftypes.Value
	Private       []byte
	SchemaVersion int64
}

// OutputState is the small output model Terraformer needs while generating
// legacy state and output files.
type OutputState struct {
	Type  string
	Value interface{}
}

type ApplicableFilter interface {
	IsApplicable(resourceName string) bool
}

type ResourceFilter struct {
	ApplicableFilter
	ServiceName      string
	FieldPath        string
	AcceptableValues []string
}

func (rf *ResourceFilter) Filter(resource Resource) bool {
	if !rf.IsApplicable(strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")) {
		return true
	}
	var vals []interface{}
	switch {
	case rf.FieldPath == "id":
		vals = []interface{}{resource.InstanceState.ID}
	case rf.AcceptableValues == nil:
		var hasField = WalkAndCheckField(rf.FieldPath, resource.InstanceState.Attributes)
		if hasField {
			return true
		}
		return WalkAndCheckField(rf.FieldPath, resource.Item)
	default:
		vals = WalkAndGet(rf.FieldPath, resource.InstanceState.Attributes)
		if len(vals) == 0 {
			vals = WalkAndGet(rf.FieldPath, resource.Item)
		}
	}
	for _, val := range vals {
		for _, acceptableValue := range rf.AcceptableValues {
			if val == acceptableValue {
				return true
			}
		}
	}
	return false
}

func (rf *ResourceFilter) IsApplicable(serviceName string) bool {
	return rf.ServiceName == "" || rf.ServiceName == serviceName
}

func (rf *ResourceFilter) isInitial() bool {
	return rf.FieldPath == "id"
}

func NewResource(id, resourceName, resourceType, provider string,
	attributes map[string]string,
	allowEmptyValues []string,
	additionalFields map[string]interface{}) Resource {
	return Resource{
		ResourceName: TfSanitize(resourceName),
		Item:         nil,
		Provider:     provider,
		InstanceState: &ResourceState{
			ID:         id,
			Attributes: attributes,
		},
		InstanceInfo: &ResourceAddress{
			Type: resourceType,
			Name: TfSanitize(resourceName),
			Id:   fmt.Sprintf("%s.%s", resourceType, TfSanitize(resourceName)),
		},
		AdditionalFields: additionalFields,
		AllowEmptyValues: allowEmptyValues,
	}
}

func NewSimpleResource(id, resourceName, resourceType, provider string, allowEmptyValues []string) Resource {
	return NewResource(
		id,
		resourceName,
		resourceType,
		provider,
		map[string]string{},
		allowEmptyValues,
		map[string]interface{}{},
	)
}

func NewResourceFromDiscovery(id, resourceName, resourceType, provider string, attributes map[string]interface{}, allowEmptyValues []string, additionalFields map[string]interface{}) Resource {
	resource := NewResource(id, resourceName, resourceType, provider, map[string]string{}, allowEmptyValues, additionalFields)
	resource.DiscoveryAttributes = attributes
	return resource
}

func (r *Resource) Refresh(provider *providerwrapper.ProviderWrapper) {
	if r.SlowQueryRequired {
		time.Sleep(200 * time.Millisecond)
	}
	schema, ok := provider.GetSchema().ResourceSchemas[r.InstanceInfo.Type]
	if !ok {
		log.Printf("provider has no schema for %s", r.InstanceInfo.Type)
		return
	}
	prior := r.InstanceState.Value
	var err error
	if prior.Type() == nil {
		prior, err = discoveryPriorValue(schema.ValueType(), r.InstanceState.ID, r.InstanceState.Attributes, r.DiscoveryAttributes)
		if err != nil {
			log.Println(err)
			return
		}
	}
	value, private, err := provider.RefreshValue(r.InstanceInfo.Type, r.InstanceState.ID, prior, r.InstanceState.Private)
	if err != nil {
		log.Println(err)
		return
	}
	r.InstanceState.Value = value
	r.InstanceState.Private = private
	r.InstanceState.SchemaVersion = schema.Version
	r.MapAttributes = schemaMapAttributes(schema.Block, "")
}

func discoveryPriorValue(typ tftypes.Type, id string, attributes map[string]string, discovered map[string]interface{}) (tftypes.Value, error) {
	object, ok := typ.(tftypes.Object)
	if !ok {
		return tftypes.Value{}, fmt.Errorf("resource schema must be an object, got %T", typ)
	}
	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
		if raw, exists := discovered[name]; exists {
			value, err := discoveryValue(attributeType, raw)
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("discovery attribute %q: %w", name, err)
			}
			values[name] = value
			continue
		}
		raw, exists := attributes[name]
		if name == "id" && id != "" {
			raw, exists = id, true
		}
		if !exists {
			continue
		}
		value, err := primitiveDiscoveryValue(attributeType, raw)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("discovery attribute %q: %w", name, err)
		}
		values[name] = value
	}
	return tftypes.NewValue(typ, values), nil
}

func discoveryValue(typ tftypes.Type, raw interface{}) (tftypes.Value, error) {
	if raw == nil {
		return tftypes.NewValue(typ, nil), nil
	}
	if typ.Is(tftypes.String) || typ.Is(tftypes.Bool) || typ.Is(tftypes.Number) {
		return primitiveDiscoveryValue(typ, fmt.Sprint(raw))
	}
	switch typed := typ.(type) {
	case tftypes.List:
		return discoveryCollectionValue(typ, typed.ElementType, raw)
	case tftypes.Set:
		return discoveryCollectionValue(typ, typed.ElementType, raw)
	case tftypes.Map:
		input, ok := raw.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected map, got %T", raw)
		}
		values := make(map[string]tftypes.Value, len(input))
		for name, item := range input {
			value, err := discoveryValue(typed.ElementType, item)
			if err != nil {
				return tftypes.Value{}, err
			}
			values[name] = value
		}
		return tftypes.NewValue(typ, values), nil
	case tftypes.Object:
		input, ok := raw.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected object, got %T", raw)
		}
		values := make(map[string]tftypes.Value, len(typed.AttributeTypes))
		for name, attributeType := range typed.AttributeTypes {
			item, exists := input[name]
			if !exists {
				values[name] = tftypes.NewValue(attributeType, nil)
				continue
			}
			value, err := discoveryValue(attributeType, item)
			if err != nil {
				return tftypes.Value{}, err
			}
			values[name] = value
		}
		return tftypes.NewValue(typ, values), nil
	default:
		return tftypes.NewValue(typ, nil), nil
	}
}

func discoveryCollectionValue(collectionType, elementType tftypes.Type, raw interface{}) (tftypes.Value, error) {
	items := reflect.ValueOf(raw)
	if items.Kind() != reflect.Slice && items.Kind() != reflect.Array {
		return tftypes.Value{}, fmt.Errorf("expected collection, got %T", raw)
	}
	values := make([]tftypes.Value, items.Len())
	for i := 0; i < items.Len(); i++ {
		value, err := discoveryValue(elementType, items.Index(i).Interface())
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("element %d: %w", i, err)
		}
		values[i] = value
	}
	return tftypes.NewValue(collectionType, values), nil
}

func primitiveDiscoveryValue(typ tftypes.Type, raw string) (tftypes.Value, error) {
	switch {
	case typ.Is(tftypes.String):
		return tftypes.NewValue(typ, raw), nil
	case typ.Is(tftypes.Bool):
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(typ, value), nil
	case typ.Is(tftypes.Number):
		value, _, err := big.ParseFloat(raw, 10, 256, big.ToNearestEven)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(typ, value), nil
	default:
		return tftypes.NewValue(typ, nil), nil
	}
}

func schemaMapAttributes(block *tfprotov5.SchemaBlock, parent string) []string {
	if block == nil {
		return nil
	}
	result := make([]string, 0)
	for _, attribute := range block.Attributes {
		if attribute.Type != nil && attribute.Type.Is(tftypes.Map{}) {
			result = append(result, joinProjectionPath(parent, attribute.Name))
		}
	}
	for _, nested := range block.BlockTypes {
		result = append(result, schemaMapAttributes(nested.Block, joinProjectionPath(parent, nested.TypeName))...)
	}
	return result
}

func (r Resource) GetIDKey() string {
	if _, exist := r.InstanceState.Attributes["self_link"]; exist {
		return "self_link"
	}
	return "id"
}

func (r Resource) StateAttribute(key string) interface{} {
	if key == "id" {
		return r.InstanceState.ID
	}
	if value, ok := r.InstanceState.Attributes[key]; ok {
		return value
	}
	if r.Item != nil {
		return r.Item[key]
	}
	return nil
}

func (r *Resource) ConvertTFstate(provider *providerwrapper.ProviderWrapper) error {
	ignoreKeys := []*regexp.Regexp{}
	for _, pattern := range r.IgnoreKeys {
		ignoreKeys = append(ignoreKeys, regexp.MustCompile(pattern))
	}
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range r.AllowEmptyValues {
		if pattern != "" {
			allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
		}
	}
	schema, ok := provider.GetSchema().ResourceSchemas[r.InstanceInfo.Type]
	if !ok {
		return fmt.Errorf("provider has no schema for %s", r.InstanceInfo.Type)
	}
	if r.InstanceState.Value.Type() != nil {
		projected, err := ProjectTerraformValue(r.InstanceState.Value)
		if err != nil {
			return err
		}
		attributes, ok := projected.(map[string]interface{})
		if !ok {
			return fmt.Errorf("resource state must be an object, got %T", projected)
		}
		cleaned, _ := cleanupProjection(attributes, "", ignoreKeys, allowEmptyValues)
		attributes = cleaned.(map[string]interface{})
		for key, value := range r.AdditionalFields {
			attributes[key] = value
		}
		r.Item = attributes
		r.StateJSON, err = typedValueJSON(r.InstanceState.Value)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("resource %s has no typed protocol state", r.InstanceInfo.Id)
	}
	r.StateSchemaVersion = uint64(schema.Version)
	r.InstanceState.SchemaVersion = int64(r.StateSchemaVersion)
	return nil
}

func (r *Resource) ServiceName() string {
	return strings.TrimPrefix(r.InstanceInfo.Type, r.Provider+"_")
}
