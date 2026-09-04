package catalog

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

const fixture = `version: "2"
module: sample
domain: storage
items:
  - {key: BACKEND, owner: sample, type: string, description: backend, resolution: placeholder, default: local, secret: false, schema: {enum: [local, remote]}}
  - {key: TOKEN, owner: sample, type: secret, description: token, resolution: placeholder, secret: true}
schema:
  if: {properties: {BACKEND: {const: remote}}, required: [BACKEND]}
  then: {required: [TOKEN]}
ui_schema: {type: VerticalLayout}
`

func loadFixture(t *testing.T, raw string) *Catalog {
	t.Helper()
	c, err := Load(context.Background(), FSSource{FS: fstest.MapFS{"storage_catalog.yaml": &fstest.MapFile{Data: []byte(raw)}}})
	require.NoError(t, err)
	return c
}

func TestCatalogV2ValidationAndFreeze(t *testing.T) {
	c := loadFixture(t, fixture)
	require.Empty(t, c.Validate(map[string]any{"BACKEND": "local"}))
	require.NotEmpty(t, c.Validate(map[string]any{"BACKEND": "wrong"}))
	issues := c.Validate(map[string]any{"BACKEND": "remote"})
	require.NotEmpty(t, issues)
	require.Equal(t, "TOKEN", issues[0].Key)
	require.Equal(t, "required", issues[0].Code)
	frozen := c.Freeze()
	restored, err := Restore(frozen)
	require.NoError(t, err)
	require.Equal(t, issues, restored.Validate(map[string]any{"BACKEND": "remote"}))
	frozen.Documents[0].Schema["then"] = map[string]any{}
	_, err = Restore(frozen)
	require.Error(t, err)
	require.NotEmpty(t, c.Validate(map[string]any{"BACKEND": "remote"}))
}

func TestCatalogAcceptsJSONFormsVisibilityRules(t *testing.T) {
	raw := strings.Replace(fixture, "ui_schema: {type: VerticalLayout}", `ui_schema:
  type: VerticalLayout
  elements:
    - type: Control
      scope: '#/properties/TOKEN'
      rule:
        effect: SHOW
        condition:
          scope: '#/properties/BACKEND'
          schema: {const: remote}
          failWhenUndefined: true
`, 1)
	loadFixture(t, raw)
}

func TestCatalogV1AndRejectedDefinitions(t *testing.T) {
	legacy := `version: "1"
module: sample
domain: core
items:
  - {key: PORT, owner: sample, type: int, description: port, resolution: placeholder, default: 0, secret: false}
`
	loadFixture(t, legacy)
	for name, raw := range map[string]string{
		"unknown field":       strings.Replace(legacy, "description: port", "descriptino: port", 1),
		"duplicate yaml":      legacy + "version: '1'\n",
		"secret default":      strings.Replace(legacy, "secret: false", "secret: true", 1),
		"unknown schema":      fixture + "unknown: true\n",
		"unsupported keyword": strings.Replace(fixture, "enum: [local, remote]", "active_when: true", 1),
		"duplicate default":   strings.Replace(fixture, "enum: [local, remote]", "default: remote", 1),
		"reference":           strings.Replace(fixture, "enum: [local, remote]", "$ref: 'https://invalid.example/schema'", 1),
		"unsupported ui effect": strings.Replace(fixture, "ui_schema: {type: VerticalLayout}", `ui_schema:
  type: Control
  scope: '#/properties/TOKEN'
  rule:
    effect: ENABLE
    condition: {scope: '#/properties/BACKEND', schema: {const: remote}}
`, 1),
		"runtime ui semantics": strings.Replace(fixture, "ui_schema: {type: VerticalLayout}", `ui_schema:
  type: Control
  scope: '#/properties/TOKEN'
  runtime: {when: {BACKEND: remote}}
`, 1),
		"invalid ui scope": strings.Replace(fixture, "ui_schema: {type: VerticalLayout}", `ui_schema:
  type: Control
  scope: BACKEND
`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(context.Background(), FSSource{FS: fstest.MapFS{"core_catalog.yaml": &fstest.MapFile{Data: []byte(raw)}}})
			require.Error(t, err)
		})
	}
}

func TestCatalogAggregatesDocumentsAndSafeErrors(t *testing.T) {
	other := `version: "1"
module: sample
domain: extra
items:
  - {key: ENABLED, owner: sample, type: bool, description: enabled, resolution: placeholder, secret: false}
`
	source := Documents{{Name: "storage_catalog.yaml", Content: []byte(fixture)}, {Name: "extra_catalog.yaml", Content: []byte(other)}}
	c, err := Load(context.Background(), source)
	require.NoError(t, err)
	require.Len(t, c.Items(), 3)
	_, err = Load(context.Background(), append(source, Document{Name: "duplicate_catalog.yaml", Content: []byte(other)}))
	require.Error(t, err)
	issues := c.Validate(map[string]any{"BACKEND": "private-secret-value"})
	require.NotEmpty(t, issues)
	require.NotContains(t, issues.Error(), "private-secret-value")
}

func TestCatalogRejectsConflictingTypesAndUndeclaredRuleKeys(t *testing.T) {
	for _, rule := range []string{
		"properties: {BACKEND: {type: integer}}",
		"properties: {BACKEND: {allOf: [{type: string}]}}",
		"required: [TYPO]",
		"if: {properties: {TYPO: {const: yes}}}\n  then: {required: [TOKEN]}",
	} {
		raw := strings.Split(fixture, "schema:\n")[0] + "schema:\n  " + rule + "\n"
		_, err := Load(context.Background(), Documents{{Name: "core_catalog.yaml", Content: []byte(raw)}})
		require.Error(t, err, rule)
	}
}
