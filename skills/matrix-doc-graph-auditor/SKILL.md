---
name: matrix-doc-graph-auditor
description: Audit Matrix markdown docs for frontmatter completeness, README index drift, relative link breakage, and unresolved target_uuid relations. Use when reviewing the full Matrix docs tree, checking whether each docs directory still matches its README rules, or validating that target_uuid points to real document nodes.
---

# Matrix Doc Graph Auditor

## Overview

Use this skill when you need to audit Matrix documentation as a graph instead of reading files one by one. The canonical source now lives in the Matrix repo at `/Users/hejiajun/Documents/开发设计/Evolution/platform/Matrix/skills/matrix-doc-graph-auditor`, and `~/.codex/skills/matrix-doc-graph-auditor` should remain only a symlink to that repo directory.

It is especially useful for:

- checking the full `docs/` tree, including `designs`, `guides`, `reference`, `migration`, `templates`, and `latest`
- finding missing frontmatter, duplicate `uuid`, blank `target_uuid`, and broken relative links
- enforcing lowercase hyphenated UUID format for both `uuid` and `target_uuid`
- comparing each directory `README.md` index against the actual files and child directories on disk
- validating whether an RFC/ADR/Plan status chain is consistent with the document graph it points to
- enforcing that `Accepted` / `Implementing` RFCs point to at least one current usage document under `docs/guides/` or `docs/migration/`
- enforcing that every RFC contains a dedicated `原始需求点总结` section near the front of the document
- enforcing that `Accepted` / `Implementing` RFCs contain a dedicated `当前实现对齐` section
- allowing template-only placeholder UUIDs inside `docs/templates`, while flagging those placeholders elsewhere

This skill is intentionally structural. It can prove whether references resolve and whether README rules are followed, but code-implementation alignment for an RFC still needs a short manual review after the structural scan.

## Quick Start

1. Pick the Matrix repo root or docs root.
2. Run the bundled auditor script.
3. Fix the reported errors first, then the warnings.
4. If you are auditing RFCs, do a short manual pass on any non-`Draft` RFC to see whether code and formal spec status still match.

Common commands:

```bash
python3 ~/.codex/skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py \
  --root /path/to/Matrix/docs \
  --scope .
```

```bash
python3 ~/.codex/skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py \
  --root /path/to/Matrix/docs \
  --scope designs/rfc \
  --strict-targets \
  --fail-on warning
```

## Workflow

### 1. Run the structural audit

The bundled script checks:

- frontmatter existence and required fields
- `uuid` / `target_uuid` format (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
- duplicate `uuid`
- blank or unresolved `target_uuid`
- broken relative markdown links
- missing `README.md` in any docs subdirectory
- `README.md` index drift for files and child directories
- RFC / ADR / Plan filename convention and status values
- `Accepted` / `Implementing` RFCs include at least one current usage-doc relation
- RFCs include a dedicated `原始需求点总结` section
- `Accepted` / `Implementing` RFCs include a dedicated `当前实现对齐` section
- required parent-RFC relations for ADR / Plan docs
- placeholder UUID leakage outside `docs/templates`

Use `--strict-targets` when unresolved `target_uuid` should be treated as errors instead of warnings.

### 2. Interpret findings

Treat findings in this order:

- `error`: must be fixed before trusting the docs graph
- `warning`: probably stale, ambiguous, or intentionally external; confirm manually
- `info`: useful context, not a blocker

Expected warning examples:

- an RFC relation that intentionally points to an external task or external SOP
- an accepted RFC whose formalized spec still says `Draft`

### 3. Do the manual status pass for RFCs

For RFC directories, after the script finishes:

1. Focus on RFCs whose status is not `Draft`.
2. Open their target reference/ADR/plan docs.
3. Check whether those target docs have a compatible status.
4. Check the codebase for the claimed implementation or formalization.

Use `rg` for the code evidence pass. Prefer searching for the concrete APIs, node types, config fields, or relation names introduced by the RFC.

## Output Rules

When using this skill in a user task:

- summarize the highest-severity findings first
- separate structural issues from implementation-status issues
- if you fix docs, rerun the script and report what remains

## Resources

### scripts/

- `audit_matrix_docs.py`: scans Markdown docs, frontmatter relations, README indices, and relative links
