#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import asdict, dataclass, field
from functools import lru_cache
from pathlib import Path
from typing import Iterable


FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n?", re.S)
INLINE_LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)]+)\)")
REF_LINK_RE = re.compile(r"^\[[^\]]+\]:\s*(\S+)", re.M)
SCALAR_FIELD_RE = re.compile(r"^(uuid|type|title|status|owner|version|tags|relations):\s*(.*)$")

HEX_UUID_RE = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
RFC_FILENAME_RE = re.compile(r"^\d{4}_[a-z0-9][a-z0-9_-]*_rfc\.md$")
ADR_FILENAME_RE = re.compile(r"^\d{4}-\d+_[a-z0-9][a-z0-9_-]*_adr\.md$")
PLAN_FILENAME_RE = re.compile(r"^\d{4}-\d+_[a-z0-9][a-z0-9_-]*_plan\.md$")
REFERENCE_FILENAME_RE = re.compile(r"^\d{2}_[a-z0-9][a-z0-9_]*\.md$")
GUIDE_FILENAME_RE = re.compile(r"^[a-z0-9][a-z0-9-]*-guide\.md$")
LOCALIZED_RFC_FILENAME_RE = re.compile(r"^\d{4}-[a-z0-9][a-z0-9-]*-rfc\.zh-CN\.md$")
LOCALIZED_ADR_FILENAME_RE = re.compile(r"^\d{4}-[a-z0-9][a-z0-9-]*-adr\.zh-CN\.md$")
LOCALIZED_PLAN_FILENAME_RE = re.compile(r"^\d{4}-[a-z0-9][a-z0-9-]*-plan\.zh-CN\.md$")
LOCALIZED_REFERENCE_FILENAME_RE = re.compile(r"^\d{2}-[a-z0-9][a-z0-9-]*-reference\.zh-CN\.md$")
LOCALIZED_GUIDE_FILENAME_RE = re.compile(r"^[a-z0-9][a-z0-9-]*-guide\.zh-CN\.md$")
RFC_ORIGINAL_REQUIREMENTS_RE = re.compile(r"(?m)^##\s+原始需求点总结\s*$")
RFC_CURRENT_ALIGNMENT_RE = re.compile(r"(?m)^#{2,6}\s+(?:\d+(?:\.\d+)*\.?\s+)?(?:附录：)?当前实现对齐(?:\s*\(.*\))?\s*$")
TRACEABILITY_MATRIX_RE = re.compile(r"(?:traceability\s+matrix|追踪矩阵)", re.I)
MERMAID_DIAGRAM_RE = re.compile(r"```mermaid\s+(.*?)```", re.S)
KEY_DIAGRAM_RE = re.compile(r"\b(?:flowchart|sequenceDiagram|stateDiagram(?:-v2)?|graph|erDiagram|classDiagram)\b")

RFC_ALLOWED_STATUSES = {
    "Draft",
    "InReview",
    "Accepted",
    "Rejected",
    "Superseded",
    "Implementing",
}
ADR_ALLOWED_STATUSES = {
    "Draft",
    "Accepted",
    "Rejected",
    "Superseded",
}
PLAN_ALLOWED_STATUSES = {
    "Draft",
    "Implementing",
    "Stable",
    "Superseded",
}
REFERENCE_ALLOWED_STATUSES = {
    "Draft",
    "Stable",
    "Deprecated",
    "Superseded",
}
GUIDE_ALLOWED_STATUSES = {
    "Draft",
    "Stable",
    "Deprecated",
    "Superseded",
}
CANONICAL_RELATION_TYPES = {
    "indexes",
    "is_part_of",
    "is_indexed_by",
    "has_plan",
    "is_plan_for",
    "has_adr",
    "realizes",
    "is_refined_by",
    "refines",
    "extends",
    "has_related_rfc",
    "has_reference",
    "is_specified_by",
    "specifies",
    "has_guide",
    "implemented_by",
    "implements",
    "operates",
    "uses_reference",
    "uses_guide",
    "depends_on",
    "supports",
    "validates",
    "has_repo_mapping",
    "relates_to",
    "references",
    "complements",
    "supersedes",
    "superseded_by",
}
LEGACY_RELATION_TYPE_REPLACEMENTS = {
    "is_part of": "is_part_of",
    "implemented-by": "implemented_by",
    "specified-by": "is_specified_by 或 has_reference",
    "formalized_by": "has_adr、is_refined_by 或 realizes，按方向选择",
    "is_formalized_by": "has_adr 或 is_refined_by",
    "is_implemented_by": "implemented_by",
    "is_operated_by": "has_guide 或 implemented_by",
    "is_guided_by": "has_guide 或 uses_guide",
    "decides": "realizes",
    "plans": "is_plan_for；如果目标是 ADR，则用 implements",
    "documents": "indexes 或 is_part_of",
    "contains": "indexes 或 is_part_of",
    "explains": "references、has_guide 或 supports",
    "is_explained_by": "references、has_guide 或 supports",
    "is_referenced_by": "在引用方使用 references",
    "is_supported_by": "在支撑方使用 supports",
    "is_based_on": "depends_on",
    "is_governed_by": "uses_reference 或 depends_on",
    "is_applied_in": "supports",
    "is_extended_by": "在扩展方使用 extends",
    "extends_plan": "is_refined_by、refines 或 depends_on",
}
UNSTABLE_STATUSES = {"Draft", "InReview", "Implementing"}
FORMALIZATION_RELATION_TYPES = {
    "formalized_by",
    "is_formalized_by",
    "implemented_by",
    "is_implemented_by",
    "formalizes",
    "implements",
}
FINAL_FACT_DOC_TYPES = {"Reference", "Guide"}
NON_CURRENT_FINAL_FACT_STATUSES = {"Draft", "Deprecated", "Superseded", "Rejected"}
REFERENCE_DIAGRAM_TITLE_RE = re.compile(
    r"(?:流程|调用链|时序|状态流|架构|边界|契约|合同|适配器|端点|能力|数据模型|"
    r"flow|call|sequence|state|architecture|boundary|contract|adapter|endpoint|capability|data[_ -]?model)",
    re.I,
)
TEMPLATE_UUID_PLACEHOLDERS = {
    "GENERATED_UUID",
    "[UUID of parent RFC]",
    "[UUID of related doc]",
    "[UUID of a related concept doc]",
}
SKIP_LINK_PREFIXES = (
    "#",
    "http://",
    "https://",
    "mailto:",
    "data:",
    "javascript:",
    "app://",
    "ui://",
)
SPECIAL_INDEX_FREE_POLICIES = {"latest"}
SHARED_REQUIRED_FIELDS = ("uuid", "type", "title", "status", "owner", "version", "tags", "relations")
USAGE_DOC_ROOTS = {"guides", "migration"}
REFERENCE_DIR = "reference"
DECISION_TRACEABILITY_FILENAME = "00_decision_traceability.md"
REPO_MAPPING_FILENAME = "01_repo_mapping.md"
LOCALIZED_DECISION_TRACEABILITY_FILENAME = "00-decision-traceability-reference.zh-CN.md"
LOCALIZED_REPO_MAPPING_FILENAME = "01-repo-mapping-reference.zh-CN.md"
MATRIX_PROFILE = "matrix"
LOCALIZED_ZH_CN_PROFILE = "localized-zh-cn"
README_FILENAMES = {
    MATRIX_PROFILE: ("README.md",),
    LOCALIZED_ZH_CN_PROFILE: ("README.zh-CN.md", "README.md"),
}
DECISION_TRACEABILITY_FILENAMES = {
    MATRIX_PROFILE: (DECISION_TRACEABILITY_FILENAME,),
    LOCALIZED_ZH_CN_PROFILE: (LOCALIZED_DECISION_TRACEABILITY_FILENAME, DECISION_TRACEABILITY_FILENAME),
}
REPO_MAPPING_FILENAMES = {
    MATRIX_PROFILE: (REPO_MAPPING_FILENAME,),
    LOCALIZED_ZH_CN_PROFILE: (LOCALIZED_REPO_MAPPING_FILENAME, REPO_MAPPING_FILENAME),
}
FILENAME_PATTERNS = {
    MATRIX_PROFILE: {
        "rfc": (RFC_FILENAME_RE, "NNNN_topic_rfc.md"),
        "adr": (ADR_FILENAME_RE, "NNNN-N_topic_adr.md"),
        "plan": (PLAN_FILENAME_RE, "NNNN-N_topic_plan.md"),
        "reference": (REFERENCE_FILENAME_RE, "NN_topic.md"),
        "guides": (GUIDE_FILENAME_RE, "<slug>-guide.md"),
    },
    LOCALIZED_ZH_CN_PROFILE: {
        "rfc": (LOCALIZED_RFC_FILENAME_RE, "NNNN-topic-rfc.zh-CN.md"),
        "adr": (LOCALIZED_ADR_FILENAME_RE, "NNNN-topic-adr.zh-CN.md"),
        "plan": (LOCALIZED_PLAN_FILENAME_RE, "NNNN-topic-plan.zh-CN.md"),
        "reference": (LOCALIZED_REFERENCE_FILENAME_RE, "NN-topic-reference.zh-CN.md"),
        "guides": (LOCALIZED_GUIDE_FILENAME_RE, "<slug>-guide.zh-CN.md"),
    },
}
ALL_README_FILENAMES = tuple(dict.fromkeys(name for names in README_FILENAMES.values() for name in names))
ALL_DECISION_TRACEABILITY_FILENAMES = tuple(
    dict.fromkeys(name for names in DECISION_TRACEABILITY_FILENAMES.values() for name in names)
)
ALL_REPO_MAPPING_FILENAMES = tuple(dict.fromkeys(name for names in REPO_MAPPING_FILENAMES.values() for name in names))
DESIGN_DOC_POLICIES = {
    "RFC": "rfc",
    "ADR": "adr",
    "Plan": "plan",
}
LEGACY_REPO_LOCAL_DESIGN_DIRS = {
    "rfc": "designs/rfc",
    "adr": "designs/adr",
    "plan": "designs/plan",
    "plans": "designs/plan",
}
DESIGN_DOC_DIRS = {"rfc", "adr", "plan"}
FORMAL_DOC_SET_DIRS = {"designs", "reference", "guides", "migration", "templates", "latest"}


@dataclass
class Relation:
    type: str = ""
    target_uuid: str = ""
    description: str = ""


@dataclass
class Doc:
    path: Path
    relpath: Path
    parent_relpath: Path
    has_frontmatter: bool
    uuid: str = ""
    type: str = ""
    title: str = ""
    status: str = ""
    owner: str = ""
    version: str = ""
    relations: list[Relation] = field(default_factory=list)
    has_tags_field: bool = False
    has_relations_field: bool = False
    frontmatter: str = ""
    body: str = ""

    @property
    def is_template(self) -> bool:
        return "templates" in self.relpath.parts and not self.is_readme

    @property
    def is_readme(self) -> bool:
        return is_readme_name(self.path.name)

    @property
    def is_hidden(self) -> bool:
        return any(part.startswith(".") for part in self.relpath.parts)


@dataclass
class Issue:
    level: str
    code: str
    path: str
    message: str


def unquote(value: str) -> str:
    value = value.strip()
    if not value:
        return value
    in_quotes = False
    result: list[str] = []
    for index, char in enumerate(value):
        if char == '"' and (index == 0 or value[index - 1] != "\\"):
            in_quotes = not in_quotes
        if char == "#" and not in_quotes:
            break
        result.append(char)
    cleaned = "".join(result).strip()
    if len(cleaned) >= 2 and cleaned[0] == cleaned[-1] == '"':
        return cleaned[1:-1]
    return cleaned


def parse_frontmatter_fields(frontmatter: str) -> tuple[dict[str, str], bool, bool]:
    fields: dict[str, str] = {}
    has_tags_field = False
    has_relations_field = False

    for raw in frontmatter.splitlines():
        stripped = raw.strip()
        if not stripped or stripped.startswith("#"):
            continue
        match = SCALAR_FIELD_RE.match(stripped)
        if not match:
            continue
        key, raw_value = match.groups()
        if key == "tags":
            has_tags_field = True
            continue
        if key == "relations":
            has_relations_field = True
            continue
        fields[key] = unquote(raw_value)

    return fields, has_tags_field, has_relations_field


def parse_relations(frontmatter: str) -> tuple[list[Relation], bool]:
    lines = frontmatter.splitlines()
    relations: list[Relation] = []
    in_relations = False
    saw_relations_field = False
    current: Relation | None = None

    for raw in lines:
        line = raw.rstrip()
        stripped = line.strip()
        if stripped.startswith("relations:"):
            saw_relations_field = True
            in_relations = True
            if stripped == "relations: []":
                return [], True
            continue
        if not in_relations:
            continue
        if stripped and not line.startswith(" ") and not line.startswith("\t") and not stripped.startswith("-"):
            break
        if stripped.startswith("- type:"):
            if current is not None:
                relations.append(current)
            current = Relation(type=unquote(stripped.split(":", 1)[1]))
            continue
        if current is None:
            continue
        if stripped.startswith("target_uuid:"):
            current.target_uuid = unquote(stripped.split(":", 1)[1])
        elif stripped.startswith("description:"):
            current.description = unquote(stripped.split(":", 1)[1])

    if current is not None:
        relations.append(current)
    return relations, saw_relations_field


def parse_doc(root: Path, path: Path) -> Doc:
    text = path.read_text(encoding="utf-8")
    match = FRONTMATTER_RE.match(text)
    relpath = path.relative_to(root)
    parent_relpath = relpath.parent if relpath.parent != Path(".") else Path(".")
    if not match:
        return Doc(
            path=path,
            relpath=relpath,
            parent_relpath=parent_relpath,
            has_frontmatter=False,
            body=text,
        )

    frontmatter = match.group(1)
    fields, has_tags_field, scalar_relations_field = parse_frontmatter_fields(frontmatter)
    relations, relations_field_in_block = parse_relations(frontmatter)
    has_relations_field = scalar_relations_field or relations_field_in_block
    body = text[match.end():]

    return Doc(
        path=path,
        relpath=relpath,
        parent_relpath=parent_relpath,
        has_frontmatter=True,
        uuid=fields.get("uuid", "").strip(),
        type=fields.get("type", "").strip(),
        title=fields.get("title", "").strip(),
        status=fields.get("status", "").strip(),
        owner=fields.get("owner", "").strip(),
        version=fields.get("version", "").strip(),
        relations=relations,
        has_tags_field=has_tags_field,
        has_relations_field=has_relations_field,
        frontmatter=frontmatter,
        body=body,
    )


def discover_docs(root: Path) -> list[Doc]:
    return [parse_doc(root, path) for path in sorted(root.rglob("*.md")) if ".DS_Store" not in path.parts]


def resolve_scope(root: Path, scope: str | None) -> Path:
    if not scope:
        return root
    scope_path = Path(scope)
    if not scope_path.is_absolute():
        scope_path = root / scope_path
    return scope_path.resolve()


def iter_scoped_docs(all_docs: Iterable[Doc], scope_path: Path) -> list[Doc]:
    scoped: list[Doc] = []
    for doc in all_docs:
        try:
            doc.path.relative_to(scope_path)
        except ValueError:
            continue
        scoped.append(doc)
    return scoped


def iter_scoped_dirs(root: Path, scope_path: Path) -> list[Path]:
    dirs = [scope_path]
    for path in sorted(scope_path.rglob("*")):
        if not path.is_dir():
            continue
        if any(part.startswith(".") for part in path.relative_to(root).parts):
            continue
        dirs.append(path)
    return dirs


def add_issue(issues: list[Issue], level: str, code: str, path: Path, message: str) -> None:
    issues.append(Issue(level=level, code=code, path=str(path), message=message))


def strip_code_spans(text: str) -> str:
    lines: list[str] = []
    in_fence = False
    fence_delim = ""
    for raw in text.splitlines():
        stripped = raw.lstrip()
        if stripped.startswith("```") or stripped.startswith("~~~"):
            fence = stripped[:3]
            if not in_fence:
                in_fence = True
                fence_delim = fence
            elif fence == fence_delim:
                in_fence = False
                fence_delim = ""
            continue
        if in_fence:
            continue
        lines.append(re.sub(r"`[^`\n]*`", "", raw))
    return "\n".join(lines)


def iter_markdown_links(text: str) -> Iterable[str]:
    sanitized = strip_code_spans(text)
    for target in INLINE_LINK_RE.findall(sanitized):
        yield target
    for target in REF_LINK_RE.findall(sanitized):
        yield target


def is_readme_name(name: str) -> bool:
    return name in ALL_README_FILENAMES


def is_decision_traceability_name(name: str) -> bool:
    return name in ALL_DECISION_TRACEABILITY_FILENAMES


def is_repo_mapping_name(name: str) -> bool:
    return name in ALL_REPO_MAPPING_FILENAMES


def doc_set_relative_root(relpath: Path) -> Path:
    parts = relpath.parts
    for index, part in enumerate(parts):
        if part == "designs" and index + 1 < len(parts) and parts[index + 1] in DESIGN_DOC_DIRS:
            return Path(*parts[:index]) if index else Path(".")
        if part in FORMAL_DOC_SET_DIRS - {"designs"}:
            return Path(*parts[:index]) if index else Path(".")
    return Path(".")


def layout_relative_path(relpath: Path) -> Path:
    rel = relpath.as_posix()
    if rel == ".":
        return Path(".")
    parts = relpath.parts
    for index, part in enumerate(parts):
        if part == "designs" or part in FORMAL_DOC_SET_DIRS - {"designs"}:
            return Path(*parts[index:])
    return relpath


def get_directory_policy(relpath: Path) -> str:
    layout_relpath = layout_relative_path(relpath)
    rel = layout_relpath.as_posix()
    if rel == ".":
        return "docs_root"
    parts = layout_relpath.parts
    if len(parts) >= 2 and parts[0] == "reference":
        return "reference"
    if rel == "designs":
        return "designs_root"
    if rel == "designs/rfc":
        return "rfc"
    if rel == "designs/adr":
        return "adr"
    if rel == "designs/plan":
        return "plan"
    if rel == "reference":
        return "reference"
    if rel == "guides":
        return "guides"
    if rel == "guides/components":
        return "guides_components"
    if rel == "migration":
        return "migration"
    if rel == "templates":
        return "templates"
    if rel == "latest":
        return "latest"
    return "generic"


def get_root_policy(root: Path) -> str | None:
    parts = root.parts
    if len(parts) >= 2 and parts[-2] == "designs":
        if parts[-1] == "rfc":
            return "rfc"
        if parts[-1] == "adr":
            return "adr"
        if parts[-1] == "plan":
            return "plan"
    if root.name == "reference":
        return "reference"
    if root.name == "guides":
        return "guides"
    if len(parts) >= 2 and parts[-2] == "guides" and parts[-1] == "components":
        return "guides_components"
    if root.name == "migration":
        return "migration"
    if root.name == "templates":
        return "templates"
    if root.name == "latest":
        return "latest"
    return None


def get_doc_policy(root: Path, doc: Doc) -> str:
    policy = get_directory_policy(doc.parent_relpath)
    if policy == "docs_root":
        root_policy = get_root_policy(root)
        if root_policy:
            return root_policy
    return policy


def is_cross_repo_context(root: Path, doc: Doc | None = None) -> bool:
    if "cross-repo" in root.parts:
        return True
    if doc and doc.relpath.parts and doc.relpath.parts[0] == "cross-repo":
        return True
    return False


def is_placeholder_uuid(value: str) -> bool:
    return value in TEMPLATE_UUID_PLACEHOLDERS


def is_valid_uuid(value: str) -> bool:
    return bool(HEX_UUID_RE.match(value))


def is_formal_doc(doc: Doc) -> bool:
    if doc.is_readme or doc.is_template or doc.is_hidden:
        return False
    if doc.type in {"RFC", "ADR", "Plan", "Reference"}:
        return True
    rel = doc.relpath.as_posix()
    return rel.startswith(("designs/rfc/", "designs/adr/", "designs/plan/", "reference/"))


def root_doc_set_base(root: Path) -> Path:
    parts = root.parts
    if len(parts) >= 2 and parts[-2] == "designs" and parts[-1] in {"rfc", "adr", "plan"}:
        return root.parent.parent
    if len(parts) >= 2 and parts[-2] == "guides" and parts[-1] == "components":
        return root.parent.parent
    if root.name in {"reference", "guides", "migration", "templates", "latest"}:
        return root.parent
    return root


def doc_set_root(root: Path, doc: Doc) -> Path:
    base = root_doc_set_base(root)
    relroot = doc_set_relative_root(doc.relpath)
    if relroot == Path("."):
        return base
    return base / relroot


@lru_cache(maxsize=None)
def infer_doc_set_profile(set_root: Path) -> str:
    if (set_root / "README.zh-CN.md").exists():
        return LOCALIZED_ZH_CN_PROFILE
    if (set_root / REFERENCE_DIR / LOCALIZED_DECISION_TRACEABILITY_FILENAME).exists():
        return LOCALIZED_ZH_CN_PROFILE
    return MATRIX_PROFILE


def profile_names(profile: str, registry: dict[str, tuple[str, ...]]) -> tuple[str, ...]:
    preferred = registry.get(profile, ())
    fallback = tuple(name for names in registry.values() for name in names if name not in preferred)
    return tuple(dict.fromkeys((*preferred, *fallback)))


def find_named_file(directory: Path, names: Iterable[str]) -> Path | None:
    for name in names:
        candidate = directory / name
        if candidate.exists():
            return candidate
    return None


def find_directory_readme(directory: Path) -> Path | None:
    return find_named_file(directory, ALL_README_FILENAMES)


def find_decision_traceability_file(set_root: Path) -> Path | None:
    profile = infer_doc_set_profile(set_root)
    return find_named_file(set_root / REFERENCE_DIR, profile_names(profile, DECISION_TRACEABILITY_FILENAMES))


def expected_decision_traceability_path(set_root: Path) -> Path:
    profile = infer_doc_set_profile(set_root)
    return set_root / REFERENCE_DIR / DECISION_TRACEABILITY_FILENAMES[profile][0]


def filename_pattern(root: Path, doc: Doc, policy: str) -> tuple[re.Pattern[str], str, str]:
    profile = infer_doc_set_profile(doc_set_root(root, doc))
    pattern, label = FILENAME_PATTERNS[profile][policy]
    return pattern, label, profile


def doc_set_relative_path(root: Path, path: Path) -> Path | None:
    base = root_doc_set_base(root)
    try:
        relpath = path.relative_to(base)
    except ValueError:
        return None
    relroot = doc_set_relative_root(relpath)
    if relroot == Path("."):
        return relpath
    try:
        return relpath.relative_to(relroot)
    except ValueError:
        return relpath


def final_fact_doc(doc: Doc) -> bool:
    if doc.is_readme or doc.is_template:
        return False
    if doc.type in FINAL_FACT_DOC_TYPES:
        return doc.status not in NON_CURRENT_FINAL_FACT_STATUSES
    return bool(doc.relpath.parts) and doc.relpath.parts[0] in {"reference", "guides"}


def target_is_final_fact(target: Doc | None) -> bool:
    return target is not None and final_fact_doc(target)


def markdown_link_points_to_final_fact(root: Path, doc: Doc) -> bool:
    for raw_target in iter_markdown_links(doc.body):
        target = raw_target.strip()
        if not target or target.startswith(SKIP_LINK_PREFIXES):
            continue
        target_no_anchor = target.split("#", 1)[0]
        if not target_no_anchor:
            continue
        target_path = (doc.path.parent / target_no_anchor).resolve()
        rel = doc_set_relative_path(root, target_path)
        if rel is None:
            continue
        if rel.parts and rel.parts[0] in {"reference", "guides"} and target_path.exists():
            return True
    return False


def decision_traceability_mentions_final_fact(doc: Doc, root: Path) -> bool:
    traceability = find_decision_traceability_file(doc_set_root(root, doc))
    if traceability is None:
        return False
    text = traceability.read_text(encoding="utf-8")
    identifiers = [value for value in (doc.uuid, doc.path.name, doc.relpath.as_posix()) if value]
    if not identifiers or not any(identifier in text for identifier in identifiers):
        return False
    for raw_target in iter_markdown_links(text):
        target = raw_target.strip()
        if not target or target.startswith(SKIP_LINK_PREFIXES):
            continue
        target_no_anchor = target.split("#", 1)[0]
        if not target_no_anchor:
            continue
        target_path = (traceability.parent / target_no_anchor).resolve()
        rel = doc_set_relative_path(root, target_path)
        if rel is None:
            continue
        if rel.parts and rel.parts[0] in {"reference", "guides"} and target_path.exists():
            return True
    return False


def has_key_mermaid_diagram(doc: Doc) -> bool:
    for block in MERMAID_DIAGRAM_RE.findall(doc.body):
        if KEY_DIAGRAM_RE.search(block):
            return True
    return False


def reference_needs_key_diagram(doc: Doc) -> bool:
    if doc.type != "Reference" or doc.is_readme or doc.status != "Stable":
        return False
    if is_decision_traceability_name(doc.path.name) or is_repo_mapping_name(doc.path.name):
        return False
    if "flows" in doc.relpath.parts:
        return True
    haystack = " ".join([doc.path.stem, doc.title])
    return bool(REFERENCE_DIAGRAM_TITLE_RE.search(haystack))


def index_expected_entries(directory: Path) -> set[str]:
    expected: set[str] = set()
    for child in sorted(directory.iterdir()):
        if child.name.startswith("."):
            continue
        if child.is_dir():
            expected.add(child.name)
        elif child.is_file() and child.suffix == ".md" and not is_readme_name(child.name):
            expected.add(child.name)
    return expected


def extract_readme_index_entries(readme_path: Path) -> set[str]:
    text = readme_path.read_text(encoding="utf-8")
    entries: set[str] = set()
    for raw_target in iter_markdown_links(text):
        target = raw_target.strip()
        if not target or target.startswith(SKIP_LINK_PREFIXES):
            continue
        target_no_anchor = target.split("#", 1)[0]
        if not target_no_anchor:
            continue
        resolved = (readme_path.parent / target_no_anchor).resolve()
        try:
            rel = resolved.relative_to(readme_path.parent.resolve())
        except ValueError:
            continue
        parts = rel.parts
        if not parts:
            continue
        if len(parts) == 1 and resolved.is_dir():
            entries.add(parts[0])
        elif len(parts) == 1 and resolved.is_file() and not is_readme_name(resolved.name):
            entries.add(parts[0])
        elif len(parts) == 2 and is_readme_name(parts[1]):
            entries.add(parts[0])
    return entries


def audit_directory_structure(root: Path, scope_path: Path, issues: list[Issue]) -> None:
    for directory in iter_scoped_dirs(root, scope_path):
        readme = find_directory_readme(directory)
        rel_dir = directory.relative_to(root)
        policy = get_directory_policy(rel_dir if rel_dir != Path(".") else Path("."))
        rel_dir_text = rel_dir.as_posix()

        if not is_cross_repo_context(root) and rel_dir_text in LEGACY_REPO_LOCAL_DESIGN_DIRS:
            add_issue(
                issues,
                "error",
                "legacy_design_directory",
                directory,
                f"repo-local design docs must live under {LEGACY_REPO_LOCAL_DESIGN_DIRS[rel_dir_text]}, not {rel_dir_text}",
            )

        if readme is None:
            add_issue(issues, "error", "missing_directory_readme", directory, "目录缺少 README.md 或 README.zh-CN.md")
            continue

        if policy in SPECIAL_INDEX_FREE_POLICIES:
            continue

        expected_entries = index_expected_entries(directory)
        indexed_entries = extract_readme_index_entries(readme)

        for missing in sorted(expected_entries - indexed_entries):
            add_issue(issues, "warning", "directory_index_missing_entry", readme, f"README 索引缺少子项: {missing}")
        for stale in sorted(indexed_entries - expected_entries):
            add_issue(issues, "error", "directory_index_stale_entry", readme, f"README 索引引用了不存在的子项: {stale}")


def audit_doc_fields(doc: Doc, issues: list[Issue]) -> None:
    if not doc.has_frontmatter:
        add_issue(issues, "error", "missing_frontmatter", doc.path, "缺少 YAML frontmatter")
        return

    field_values = {
        "uuid": doc.uuid,
        "type": doc.type,
        "title": doc.title,
        "status": doc.status,
        "owner": doc.owner,
        "version": doc.version,
        "tags": "present" if doc.has_tags_field else "",
        "relations": "present" if doc.has_relations_field else "",
    }
    for field_name in SHARED_REQUIRED_FIELDS:
        if not field_values[field_name]:
            add_issue(issues, "error", "missing_field", doc.path, f"frontmatter 缺少字段: {field_name}")

    if doc.is_template:
        if doc.uuid and doc.uuid != "GENERATED_UUID" and not is_valid_uuid(doc.uuid):
            add_issue(issues, "error", "invalid_uuid_format", doc.path, f"uuid 不是标准小写 UUID: {doc.uuid}")
        return

    if not doc.uuid:
        return
    if is_placeholder_uuid(doc.uuid):
        add_issue(issues, "error", "template_placeholder_outside_templates", doc.path, f"模板占位 uuid 只能出现在 templates 目录: {doc.uuid}")
    elif not is_valid_uuid(doc.uuid):
        add_issue(issues, "error", "invalid_uuid_format", doc.path, f"uuid 不是标准小写 UUID: {doc.uuid}")


def audit_uuid_uniqueness(all_docs: Iterable[Doc], issues: list[Issue]) -> None:
    uuid_seen: dict[str, Path] = {}
    for doc in all_docs:
        if not doc.has_frontmatter or doc.is_template:
            continue
        if not doc.uuid or not is_valid_uuid(doc.uuid):
            continue
        if doc.uuid in uuid_seen:
            add_issue(issues, "error", "duplicate_uuid", doc.path, f"uuid 与 {uuid_seen[doc.uuid]} 重复: {doc.uuid}")
        else:
            uuid_seen[doc.uuid] = doc.path


def audit_relations(doc: Doc, docs_by_uuid: dict[str, Doc], strict_targets: bool, issues: list[Issue]) -> None:
    if not doc.has_frontmatter:
        return
    for relation in doc.relations:
        if not relation.type:
            add_issue(issues, "error", "blank_relation_type", doc.path, "relations 中存在空的 type")
        elif relation.type in LEGACY_RELATION_TYPE_REPLACEMENTS:
            add_issue(
                issues,
                "warning",
                "legacy_relation_type",
                doc.path,
                f"relation type 使用历史写法: {relation.type}；建议改为 {LEGACY_RELATION_TYPE_REPLACEMENTS[relation.type]}",
            )
        elif relation.type not in CANONICAL_RELATION_TYPES:
            add_issue(issues, "error", "unknown_relation_type", doc.path, f"relation type 不在规范词表中: {relation.type}")

        if not relation.target_uuid:
            add_issue(issues, "error", "blank_target_uuid", doc.path, "relations 中存在空的 target_uuid")
            continue
        if is_placeholder_uuid(relation.target_uuid):
            if doc.is_template:
                continue
            add_issue(
                issues,
                "error",
                "template_placeholder_outside_templates",
                doc.path,
                f"模板占位 target_uuid 只能出现在 templates 目录: {relation.target_uuid}",
            )
            continue
        if not is_valid_uuid(relation.target_uuid):
            add_issue(issues, "error", "invalid_target_uuid_format", doc.path, f"target_uuid 不是标准小写 UUID: {relation.target_uuid}")
            continue
        if relation.target_uuid not in docs_by_uuid:
            description = relation.description.lower()
            is_external = "external" in description
            level = "warning"
            if strict_targets and not is_external:
                level = "error"
            code = "unresolved_external_target_uuid" if is_external else "unresolved_target_uuid"
            add_issue(issues, level, code, doc.path, f"target_uuid 未解析到任何文档节点: {relation.target_uuid}")


def audit_relative_links(doc: Doc, issues: list[Issue]) -> None:
    target_text = doc.body if doc.has_frontmatter else doc.body
    for raw_target in iter_markdown_links(target_text):
        target = raw_target.strip()
        if not target or target.startswith(SKIP_LINK_PREFIXES):
            continue
        target_no_anchor = target.split("#", 1)[0]
        if not target_no_anchor:
            continue
        target_path = (doc.path.parent / target_no_anchor).resolve()
        if not target_path.exists():
            add_issue(issues, "error", "broken_relative_link", doc.path, f"相对链接不存在: {target}")


def has_parent_rfc_relation(doc: Doc, docs_by_uuid: dict[str, Doc], relation_type: str) -> bool:
    for relation in doc.relations:
        if relation.type != relation_type:
            continue
        target = docs_by_uuid.get(relation.target_uuid)
        if target and target.type == "RFC":
            return True
    return False


def has_usage_doc_relation(doc: Doc, docs_by_uuid: dict[str, Doc]) -> bool:
    for relation in doc.relations:
        target = docs_by_uuid.get(relation.target_uuid)
        if target is None or target.is_readme:
            continue
        if not target.relpath.parts:
            continue
        if target.relpath.parts[0] in USAGE_DOC_ROOTS:
            return True
    return False


def has_final_fact_relation(doc: Doc, docs_by_uuid: dict[str, Doc]) -> bool:
    for relation in doc.relations:
        if target_is_final_fact(docs_by_uuid.get(relation.target_uuid)):
            return True
    return False


def has_original_requirements_summary(doc: Doc) -> bool:
    return bool(RFC_ORIGINAL_REQUIREMENTS_RE.search(doc.body))


def has_current_implementation_alignment(doc: Doc) -> bool:
    return bool(RFC_CURRENT_ALIGNMENT_RE.search(doc.body))


def audit_doc_text_policy(doc: Doc, issues: list[Issue]) -> None:
    if not doc.has_frontmatter or doc.is_template:
        return
    searchable = "\n".join([doc.path.name, doc.title, strip_code_spans(doc.body)])
    if TRACEABILITY_MATRIX_RE.search(searchable):
        add_issue(
            issues,
            "error",
            "forbidden_traceability_matrix_naming",
            doc.path,
            "避免使用 Traceability Matrix / 追踪矩阵 命名；请统一使用 decision traceability / 决策追踪索引。",
        )


def audit_doc_set_policy(root: Path, all_docs: Iterable[Doc], scoped_docs: Iterable[Doc], issues: list[Issue]) -> None:
    all_docs_list = list(all_docs)
    scoped_set_roots = {doc_set_root(root, doc) for doc in scoped_docs if is_formal_doc(doc)}
    if not scoped_set_roots:
        return

    docs_by_set: dict[Path, list[Doc]] = {}
    for doc in all_docs_list:
        if is_formal_doc(doc):
            docs_by_set.setdefault(doc_set_root(root, doc), []).append(doc)

    for set_root, formal_docs in sorted(docs_by_set.items(), key=lambda item: str(item[0])):
        if set_root not in scoped_set_roots or not formal_docs:
            continue

        reference_dir = set_root / REFERENCE_DIR
        decision_traceability = find_decision_traceability_file(set_root)
        if decision_traceability is None:
            expected_traceability = expected_decision_traceability_path(set_root)
            add_issue(
                issues,
                "error",
                "missing_decision_traceability",
                expected_traceability,
                f"正式 RFC / ADR / Plan / Reference 文档集必须维护 {expected_traceability.relative_to(set_root).as_posix()}。",
            )

        if reference_dir.exists():
            for candidate in sorted(reference_dir.glob("00*.md")):
                if not is_decision_traceability_name(candidate.name):
                    add_issue(
                        issues,
                        "error",
                        "reserved_reference_00_filename",
                        candidate,
                        "reference/00* 是固定保留序号，必须使用当前文档集 profile 的 decision traceability 文件名。",
                    )
            repo_mapping_candidates = set(reference_dir.glob("*repo_mapping*.md")) | set(reference_dir.glob("*repo-mapping*.md"))
            for candidate in sorted(repo_mapping_candidates):
                if not is_repo_mapping_name(candidate.name):
                    add_issue(
                        issues,
                        "error",
                        "invalid_repo_mapping_filename",
                        candidate,
                        "repo mapping 是条件固定序号；需要 repo mapping 时必须使用当前文档集 profile 的 repo mapping 文件名。",
                    )


def audit_policy(root: Path, doc: Doc, docs_by_uuid: dict[str, Doc], issues: list[Issue]) -> None:
    if not doc.has_frontmatter or doc.is_readme or doc.is_template:
        return

    policy = get_doc_policy(root, doc)
    expected_policy = DESIGN_DOC_POLICIES.get(doc.type)
    if expected_policy and policy != expected_policy and not is_cross_repo_context(root, doc):
        add_issue(
            issues,
            "error",
            "misplaced_design_doc",
            doc.path,
            f"{doc.type} documents must live under docs/designs/{expected_policy}, not {doc.parent_relpath.as_posix()}",
        )
        return

    if policy == "rfc":
        if doc.type != "RFC":
            add_issue(issues, "error", "invalid_rfc_type", doc.path, f"RFC 目录文档的 type 应为 RFC，当前为 {doc.type!r}")
        pattern, label, profile = filename_pattern(root, doc, policy)
        if not pattern.match(doc.path.name):
            add_issue(issues, "warning", "rfc_filename_convention", doc.path, f"文件名不符合 {profile} RFC 命名规范，应遵循 {label}: {doc.path.name}")
        if doc.status and doc.status not in RFC_ALLOWED_STATUSES:
            add_issue(issues, "error", "invalid_rfc_status", doc.path, f"RFC status 不在允许集合中: {doc.status}")
        if not has_original_requirements_summary(doc):
            add_issue(
                issues,
                "error",
                "missing_original_requirements_summary",
                doc.path,
                "RFC 必须包含一个二级标题“## 原始需求点总结”，用于保留最初需求。",
            )
        if doc.status in {"Accepted", "Implementing"} and not has_current_implementation_alignment(doc):
            add_issue(
                issues,
                "error",
                "missing_current_implementation_alignment",
                doc.path,
                "Accepted/Implementing RFC 必须包含“当前实现对齐”章节，用于回写当前实现范围与偏差。",
            )
        if doc.status in {"Accepted", "Implementing"} and not (
            has_final_fact_relation(doc, docs_by_uuid)
            or markdown_link_points_to_final_fact(root, doc)
            or decision_traceability_mentions_final_fact(doc, root)
        ):
            add_issue(
                issues,
                "error",
                "missing_final_fact_relation",
                doc.path,
                "Accepted/Implementing RFC 必须能追踪到一个或多个当前 Reference 或 Guide；仅关联 Plan/ADR 不算完成闭环。",
            )
        if doc.status == "Accepted":
            targets = [
                docs_by_uuid[relation.target_uuid]
                for relation in doc.relations
                if relation.target_uuid in docs_by_uuid
                and (relation.type in FORMALIZATION_RELATION_TYPES or "formal" in relation.type or "implement" in relation.type)
            ]
            unstable = [target for target in targets if target.status in UNSTABLE_STATUSES]
            if unstable:
                statuses = ", ".join(f"{target.relpath}:{target.status}" for target in unstable)
                add_issue(issues, "warning", "accepted_rfc_target_not_stable", doc.path, f"Accepted RFC 关联的 formalized/implemented 目标仍未稳定: {statuses}")
        return

    if policy == "adr":
        if doc.type != "ADR":
            add_issue(issues, "error", "invalid_adr_type", doc.path, f"ADR 目录文档的 type 应为 ADR，当前为 {doc.type!r}")
        pattern, label, profile = filename_pattern(root, doc, policy)
        if not pattern.match(doc.path.name):
            add_issue(issues, "warning", "adr_filename_convention", doc.path, f"文件名不符合 {profile} ADR 命名规范，应遵循 {label}: {doc.path.name}")
        if doc.status and doc.status not in ADR_ALLOWED_STATUSES:
            add_issue(issues, "error", "invalid_adr_status", doc.path, f"ADR status 不在允许集合中: {doc.status}")
        if not has_parent_rfc_relation(doc, docs_by_uuid, "realizes"):
            add_issue(issues, "error", "missing_parent_rfc_relation", doc.path, "ADR 必须至少有一条 type=realizes 且指向 RFC 的关系")
        return

    if policy == "plan":
        if doc.type != "Plan":
            add_issue(issues, "error", "invalid_plan_type", doc.path, f"Plan 目录文档的 type 应为 Plan，当前为 {doc.type!r}")
        pattern, label, profile = filename_pattern(root, doc, policy)
        if not pattern.match(doc.path.name):
            add_issue(issues, "warning", "plan_filename_convention", doc.path, f"文件名不符合 {profile} Plan 命名规范，应遵循 {label}: {doc.path.name}")
        if doc.status and doc.status not in PLAN_ALLOWED_STATUSES:
            add_issue(issues, "error", "invalid_plan_status", doc.path, f"Plan status 不在允许集合中: {doc.status}")
        if not has_parent_rfc_relation(doc, docs_by_uuid, "is_plan_for"):
            add_issue(issues, "error", "missing_parent_rfc_relation", doc.path, "Plan 必须至少有一条 type=is_plan_for 且指向 RFC 的关系")
        return

    if policy == "reference":
        if doc.type != "Reference":
            add_issue(issues, "error", "invalid_reference_type", doc.path, f"Reference 目录文档的 type 应为 Reference，当前为 {doc.type!r}")
        if doc.status and doc.status not in REFERENCE_ALLOWED_STATUSES:
            add_issue(issues, "error", "invalid_reference_status", doc.path, f"Reference status 不在允许集合中: {doc.status}")
        pattern, label, profile = filename_pattern(root, doc, policy)
        if not pattern.match(doc.path.name):
            add_issue(issues, "warning", "reference_filename_convention", doc.path, f"Reference 文件名不符合 {profile} 命名规范，应遵循 {label}: {doc.path.name}")
        if reference_needs_key_diagram(doc) and not has_key_mermaid_diagram(doc):
            add_issue(
                issues,
                "error",
                "missing_reference_key_diagram",
                doc.path,
                "Stable Reference 涉及流程、调用链、架构、边界、contract 或数据模型时，必须包含关键流程图、时序图或状态图。",
            )
        return

    if policy == "guides":
        if doc.type != "Guide":
            add_issue(issues, "error", "invalid_guide_type", doc.path, f"Guide 目录文档的 type 应为 Guide，当前为 {doc.type!r}")
        if doc.status and doc.status not in GUIDE_ALLOWED_STATUSES:
            add_issue(issues, "error", "invalid_guide_status", doc.path, f"Guide status 不在允许集合中: {doc.status}")
        pattern, label, profile = filename_pattern(root, doc, policy)
        if not pattern.match(doc.path.name):
            add_issue(issues, "warning", "guide_filename_convention", doc.path, f"Guide 文件名不符合 {profile} 命名规范，应遵循 {label}: {doc.path.name}")


def audit_docs(root: Path, scope_path: Path, strict_targets: bool) -> tuple[list[Issue], list[Doc]]:
    all_docs = discover_docs(root)
    docs_by_uuid = {
        doc.uuid: doc
        for doc in all_docs
        if doc.has_frontmatter and not doc.is_template and doc.uuid and is_valid_uuid(doc.uuid)
    }
    scoped_docs = iter_scoped_docs(all_docs, scope_path)
    issues: list[Issue] = []

    audit_uuid_uniqueness(all_docs, issues)
    audit_directory_structure(root, scope_path, issues)
    audit_doc_set_policy(root, all_docs, scoped_docs, issues)

    for doc in scoped_docs:
        audit_doc_fields(doc, issues)
        audit_doc_text_policy(doc, issues)
        audit_relations(doc, docs_by_uuid, strict_targets, issues)
        audit_relative_links(doc, issues)
        audit_policy(root, doc, docs_by_uuid, issues)

    return issues, scoped_docs


def summarize(issues: list[Issue], scoped_docs: list[Doc]) -> dict[str, object]:
    counts = {"error": 0, "warning": 0, "info": 0}
    for issue in issues:
        counts[issue.level] = counts.get(issue.level, 0) + 1
    return {
        "scanned_docs": len(scoped_docs),
        "counts": counts,
        "issues": [asdict(issue) for issue in issues],
    }


def print_text_report(summary: dict[str, object]) -> None:
    counts = summary["counts"]
    print(f"Scanned docs: {summary['scanned_docs']}")
    print(f"Errors: {counts['error']}  Warnings: {counts['warning']}  Info: {counts['info']}")
    print()
    for issue in summary["issues"]:
        print(f"[{issue['level'].upper()}] {issue['code']} :: {issue['path']}")
        print(f"  {issue['message']}")


def should_fail(fail_on: str, summary: dict[str, object]) -> bool:
    counts = summary["counts"]
    if fail_on == "none":
        return False
    if fail_on == "warning":
        return bool(counts["error"] or counts["warning"])
    return bool(counts["error"])


def main() -> int:
    parser = argparse.ArgumentParser(description="Audit Matrix markdown docs, frontmatter relations, and README indices.")
    parser.add_argument("--root", required=True, help="Matrix docs root, usually /path/to/Matrix/docs")
    parser.add_argument("--scope", help="Subdirectory or absolute directory to audit, e.g. designs/rfc")
    parser.add_argument("--format", choices=("text", "json"), default="text")
    parser.add_argument("--fail-on", choices=("none", "error", "warning"), default="error")
    parser.add_argument("--strict-targets", action="store_true", help="Treat unresolved non-external target_uuid as errors")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    if not root.exists():
        print(f"root does not exist: {root}", file=sys.stderr)
        return 2

    scope_path = resolve_scope(root, args.scope)
    if not scope_path.exists():
        print(f"scope does not exist: {scope_path}", file=sys.stderr)
        return 2

    issues, scoped_docs = audit_docs(root, scope_path, args.strict_targets)
    summary = summarize(issues, scoped_docs)

    if args.format == "json":
        print(json.dumps(summary, ensure_ascii=False, indent=2))
    else:
        print_text_report(summary)

    return 1 if should_fail(args.fail_on, summary) else 0


if __name__ == "__main__":
    raise SystemExit(main())
