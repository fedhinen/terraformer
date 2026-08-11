package providerwrapper //nolint

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderDiscoveryIncludesPluginCacheAndSelectsLatestVersion(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"5.100.0", "6.9.0", "6.58.0"} {
		directory := filepath.Join(root, "plugin-cache", "registry.terraform.io", "hashicorp", "aws", version, runtime.GOOS+"_"+runtime.GOARCH)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "terraform-provider-aws_v"+version+"_x5"), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	path, err := getProviderFileNameV13andV14(root, "aws")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "terraform-provider-aws_v6.58.0_x5" {
		t.Fatalf("selected %q", path)
	}
}

type fakeProviderClient struct {
	readResponse   *tfprotov5.ReadResourceResponse
	importResponse *tfprotov5.ImportResourceStateResponse
	readCalls      int
	importCalls    int
}

func (f *fakeProviderClient) GetProviderSchema(context.Context, *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	return nil, nil
}
func (f *fakeProviderClient) PrepareProviderConfig(context.Context, *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	return nil, nil
}
func (f *fakeProviderClient) ConfigureProvider(context.Context, *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	return nil, nil
}
func (f *fakeProviderClient) ReadResource(context.Context, *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	f.readCalls++
	return f.readResponse, nil
}
func (f *fakeProviderClient) ImportResourceState(context.Context, *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	f.importCalls++
	return f.importResponse, nil
}
func (f *fakeProviderClient) Close() error { return nil }

func testResourceSchema() *tfprotov5.Schema {
	return &tfprotov5.Schema{Version: 2, Block: &tfprotov5.SchemaBlock{Attributes: []*tfprotov5.SchemaAttribute{{Name: "id", Type: tftypes.String, Computed: true}}}}
}

func dynamicTestState(t *testing.T, id string) *tfprotov5.DynamicValue {
	t.Helper()
	typ := testResourceSchema().ValueType()
	dynamic, err := tfprotov5.NewDynamicValue(typ, tftypes.NewValue(typ, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, id)}))
	if err != nil {
		t.Fatal(err)
	}
	return &dynamic
}

func testWrapper(client *fakeProviderClient) *ProviderWrapper {
	return &ProviderWrapper{client: client, providerName: "test", schema: &tfprotov5.GetProviderSchemaResponse{ResourceSchemas: map[string]*tfprotov5.Schema{"test_resource": testResourceSchema()}}}
}

func TestRefreshValueReadsOnceAndPreservesPrivate(t *testing.T) {
	fake := &fakeProviderClient{readResponse: &tfprotov5.ReadResourceResponse{NewState: dynamicTestState(t, "read-id"), Private: []byte("read-private")}}
	prior, err := fake.readResponse.NewState.Unmarshal(testResourceSchema().ValueType())
	if err != nil {
		t.Fatal(err)
	}
	value, private, err := testWrapper(fake).RefreshValue("test_resource", "fixture-id", prior, []byte("prior-private"))
	if err != nil {
		t.Fatal(err)
	}
	if fake.readCalls != 1 || fake.importCalls != 0 || string(private) != "read-private" || value.IsNull() {
		t.Fatalf("read=%d import=%d private=%q value=%v", fake.readCalls, fake.importCalls, private, value)
	}
}

func TestRefreshValueFallsBackToImportWithoutRetry(t *testing.T) {
	fake := &fakeProviderClient{
		readResponse:   &tfprotov5.ReadResourceResponse{Diagnostics: []*tfprotov5.Diagnostic{{Severity: tfprotov5.DiagnosticSeverityError, Summary: "read rejected"}}},
		importResponse: &tfprotov5.ImportResourceStateResponse{ImportedResources: []*tfprotov5.ImportedResource{{TypeName: "test_resource", State: dynamicTestState(t, "imported-id"), Private: []byte("import-private")}}},
	}
	prior, err := fake.importResponse.ImportedResources[0].State.Unmarshal(testResourceSchema().ValueType())
	if err != nil {
		t.Fatal(err)
	}
	_, private, err := testWrapper(fake).RefreshValue("test_resource", "fixture-id", prior, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fake.readCalls != 1 || fake.importCalls != 1 || string(private) != "import-private" {
		t.Fatalf("read=%d import=%d private=%q", fake.readCalls, fake.importCalls, private)
	}
}

func TestRefreshValueRejectsMultiResourceImport(t *testing.T) {
	imported := &tfprotov5.ImportedResource{TypeName: "test_resource", State: dynamicTestState(t, "id")}
	fake := &fakeProviderClient{
		readResponse:   &tfprotov5.ReadResourceResponse{Diagnostics: []*tfprotov5.Diagnostic{{Severity: tfprotov5.DiagnosticSeverityError, Summary: "read rejected"}}},
		importResponse: &tfprotov5.ImportResourceStateResponse{ImportedResources: []*tfprotov5.ImportedResource{imported, imported}},
	}
	prior, err := imported.State.Unmarshal(testResourceSchema().ValueType())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = testWrapper(fake).RefreshValue("test_resource", "fixture-id", prior, nil)
	if err == nil || !regexp.MustCompile(`cannot represent a multi-resource import`).MatchString(err.Error()) {
		t.Fatalf("error = %v", err)
	}
}

func TestIgnoredAttributes(t *testing.T) {
	attributes := []*tfprotov5.SchemaAttribute{
		{
			Name:     "computed_attribute",
			Type:     tftypes.Number,
			Computed: true,
		},
		{
			Name:     "required_attribute",
			Type:     tftypes.String,
			Required: true,
		},
	}

	testCases := map[string]struct {
		block                []*tfprotov5.SchemaNestedBlock
		ignoredAttributes    []string
		notIgnoredAttributes []string
	}{
		"nesting_set": {[]*tfprotov5.SchemaNestedBlock{
			{
				TypeName: "attribute_one",
				Block: &tfprotov5.SchemaBlock{
					Attributes: attributes,
				},
				Nesting: tfprotov5.SchemaNestedBlockNestingModeSet,
			},
		}, []string{"nesting_set.attribute_one.computed_attribute"},
			[]string{"nesting_set.attribute_one.required_attribute"}},
		"nesting_list": {[]*tfprotov5.SchemaNestedBlock{
			{
				TypeName: "attribute_one",
				Block: &tfprotov5.SchemaBlock{
					BlockTypes: []*tfprotov5.SchemaNestedBlock{
						{
							TypeName: "attribute_two_nested",
							Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
							Block: &tfprotov5.SchemaBlock{
								Attributes: attributes,
							},
						},
					},
				},
				Nesting: tfprotov5.SchemaNestedBlockNestingModeList,
			},
		}, []string{"nesting_list.0.attribute_one.0.attribute_two_nested.computed_attribute"},
			[]string{"nesting_list.0.attribute_one.0.attribute_two_nested.required_attribute"}},
	}

	for key, tc := range testCases {
		t.Run(key, func(t *testing.T) {
			provider := ProviderWrapper{}
			readOnlyAttributes := provider.readObjBlocks(tc.block, []string{}, key)
			for _, attr := range tc.ignoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); !ignored {
					t.Errorf("attribute \"%s\" was not ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}

			for _, attr := range tc.notIgnoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); ignored {
					t.Errorf("attribute \"%s\" was ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}
		})
	}
}

func isAttributeIgnored(name string, patterns []string) bool {
	ignored := false
	for _, pattern := range patterns {
		if match, _ := regexp.MatchString(pattern, name); match {
			ignored = true
			break
		}
	}
	return ignored
}
