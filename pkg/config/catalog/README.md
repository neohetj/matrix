# Module configuration catalogs

This package owns loading, conversion, immutable definitions and JSON Schema validation.
It does not discover a workspace, resolve resource credentials, render UI, or write process environment variables.

- `CatalogSource` supplies explicit documents; `FSSource` reads an explicit filesystem.
- v1 retains its existing fields. v2 adds item/root `schema` and root `ui_schema`.
- `schema` uses JSON Schema draft 2020-12 validation semantics, not a custom condition language.
  The accepted keyword subset is enforced by `checkSchema`. References (including local `$ref`),
  remote loading, schema `default`, format/content vocabularies and unknown keywords are rejected.
  Catalog item `default` remains the only default source. UI metadata never affects resolution.
- `Resolve` creates one typed, read-only effective view, then validates the complete view.
  Named sources preserve `env → business → default`; node-explicit and Secret policies retain Matrix semantics.
  Hosts map an explicit resource binding into their env source before calling this package.
- `ResolveString` is node-local and does not run whole-module completeness rules.
- `ValidateProvided` validates supplied fields only for incomplete drafts; it is not an execution gate.
- `Freeze` / `Restore` retain pinned definitions and verify format plus digest. They do not upgrade old snapshots.
- `int64`, zero and false remain values. JSON restoration uses `json.Number`; lossy floating-point integers
  and integer overflow are rejected. Errors expose keys/codes/paths, never rejected input values.

Example v2 field and cross-field rule:

```yaml
version: "2"
module: sample
domain: storage
items:
  - {key: BACKEND, owner: sample, type: string, description: backend, resolution: placeholder, default: local, secret: false, schema: {enum: [local, remote]}}
  - {key: URI, owner: sample, type: url, description: address, resolution: placeholder, secret: false}
schema:
  if: {properties: {BACKEND: {const: remote}}, required: [BACKEND]}
  then: {required: [URI], properties: {URI: {minLength: 1}}}
ui_schema:
  type: VerticalLayout
  elements:
    - type: Control
      scope: "#/properties/URI"
      rule:
        effect: SHOW
        condition:
          scope: "#/properties/BACKEND"
          schema: {const: remote}
          failWhenUndefined: true
```

The UI may hide URI when BACKEND is local, but must retain its draft value. A supplied URI still
participates in validation and injection. There is no `runtime.when`, `active_when` or `required_when`.

Run focused verification with `go test ./pkg/config ./pkg/config/catalog -race` from the repository root.

## Instance readers

`NewReader(definition, resolver)` freezes only declared source keys and aliases, once per source key.
It does not run whole-module rules at construction. `Read[T]` and `ReadNode[T]` validate the selected
field; node overrides carry explicit presence so false and zero are not missing. Secret fields reject
node overrides and never read business/default values. Reader formatting and JSON never expose values.

`ReadDuration` requires an explicit bare-number unit. `NewDecoder` collects typed assembly errors;
callers must check `Err()` before performing side effects. Optional-field fallbacks do not suppress
unknown keys, required fields, or conversion failures. `AuditBusinessSecrets` reports keys and aliases
only. Legacy `ref` is a string and `float` is a finite double; neither changes exact int64 semantics.

`RenderString` reuses Asset placeholder syntax for initialization-time config expressions. In this
explicit Reader entry point, old env/engine scope spelling does not override the Reader's fixed
source order. Optional expression defaults are checked through Reader and cannot supply a Secret.
Other asset schemes and node-local scopes require their own runtime context and are not accepted here.

Hosts register readers using Matrix Engine options before constructing nodes. `ConfigReaderAware`
and `NodePoolAware` receive their Engine-owned dependencies before `Init`. `FromAssetContext` selects
the reader using an explicit module ID, without falling back to a process-global resolver.
