#!/usr/bin/env python3
"""Validate risky Matrix DSL mapping and function-signature patterns."""

from __future__ import annotations

import argparse
import json
import re
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable
from urllib.parse import parse_qs, urlparse


DEFAULT_FUNCTION_CATALOG_COMMAND = "go run ./scripts/function_catalog"
PARAM_NAME_PATTERN = re.compile(r"^[a-z]+(?:_[a-z]+)*$")


@dataclass(frozen=True)
class UriInfo:
    raw: str
    scheme: str
    host: str
    body: str
    sid: str

    @property
    def is_rulemsg_datat(self) -> bool:
        return self.scheme == "rulemsg" and self.host == "dataT"

    @property
    def is_collection_sid(self) -> bool:
        return self.sid.startswith("[]")

    @property
    def is_whole_object(self) -> bool:
        if not self.is_rulemsg_datat or not self.body:
            return False
        return "." not in self.body and "[" not in self.body

    @property
    def obj_id(self) -> str:
        if not self.body:
            return ""
        return self.body.split(".", 1)[0].split("[", 1)[0]


GENERIC_SIDS = {
    "",
    "Any",
    "Bool",
    "Float64",
    "Int64",
    "MapStringInterface",
    "MapStringString",
    "String",
    "[]Any",
}


@dataclass(frozen=True)
class Finding:
    severity: str
    code: str
    file_path: Path
    node_id: str
    packet_path: str
    message: str


@dataclass(frozen=True)
class FunctionParam:
    sid: str
    required: bool


@dataclass(frozen=True)
class FunctionSignature:
    inputs: dict[str, FunctionParam]
    outputs: dict[str, FunctionParam]


def parse_uri(raw: str) -> UriInfo | None:
    if not isinstance(raw, str) or "://" not in raw:
        return None
    parsed = urlparse(raw)
    sid = parse_qs(parsed.query).get("sid", [""])[0]
    return UriInfo(
        raw=raw,
        scheme=parsed.scheme,
        host=parsed.netloc,
        body=parsed.path.lstrip("/"),
        sid=sid,
    )


def iter_json_files(paths: Iterable[Path]) -> Iterable[Path]:
    for path in paths:
        if path.is_dir():
            yield from sorted(path.rglob("*.json"))
            continue
        if path.suffix == ".json":
            yield path


def add_finding(
    findings: list[Finding],
    severity: str,
    code: str,
    file_path: Path,
    node_id: str,
    packet_path: str,
    message: str,
) -> None:
    findings.append(
        Finding(
            severity=severity,
            code=code,
            file_path=file_path,
            node_id=node_id,
            packet_path=packet_path,
            message=message,
        )
    )


def is_strict_typed_sid(sid: str) -> bool:
    return sid not in GENERIC_SIDS and not sid.startswith("[]")


def is_patch_sid(sid: str) -> bool:
    return "Patch" in sid


def are_function_sids_compatible(actual_sid: str, expected_sid: str) -> bool:
    actual_sid = actual_sid.strip()
    expected_sid = expected_sid.strip()
    if not actual_sid or not expected_sid:
        return True
    if actual_sid == expected_sid:
        return True
    if expected_sid in GENERIC_SIDS:
        return True
    return False


def is_valid_function_param_name(name: str) -> bool:
    return bool(PARAM_NAME_PATTERN.fullmatch(name))


def parse_function_catalog(payload: dict[str, Any]) -> dict[str, FunctionSignature]:
    raw_functions = payload.get("functions")
    if not isinstance(raw_functions, dict):
        raise ValueError("function catalog payload is missing 'functions'")

    catalog: dict[str, FunctionSignature] = {}
    for function_name, raw_signature in raw_functions.items():
        if not isinstance(function_name, str) or not isinstance(raw_signature, dict):
            continue
        catalog[function_name] = FunctionSignature(
            inputs=parse_function_params(raw_signature.get("inputs")),
            outputs=parse_function_params(raw_signature.get("outputs")),
        )
    return catalog


def parse_function_params(payload: Any) -> dict[str, FunctionParam]:
    if not isinstance(payload, dict):
        return {}

    params: dict[str, FunctionParam] = {}
    for name, raw_param in payload.items():
        if not isinstance(name, str) or not isinstance(raw_param, dict):
            continue
        params[name] = FunctionParam(
            sid=str(raw_param.get("sid") or ""),
            required=bool(raw_param.get("required")),
        )
    return params


def load_function_catalog_from_json(catalog_path: Path) -> dict[str, FunctionSignature]:
    try:
        payload = json.loads(catalog_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"invalid function catalog JSON: {exc}") from exc
    return parse_function_catalog(payload)


def load_function_catalog(
    repo_root: Path,
    command_text: str = DEFAULT_FUNCTION_CATALOG_COMMAND,
) -> dict[str, FunctionSignature]:
    command = shlex.split(command_text)
    if not command:
        raise RuntimeError("function catalog command is empty")
    completed = subprocess.run(
        command,
        cwd=repo_root,
        capture_output=True,
        text=True,
        check=False,
    )
    if completed.returncode != 0:
        stderr = completed.stderr.strip()
        raise RuntimeError(stderr or f"{command_text} failed")

    try:
        payload = json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"invalid function catalog JSON: {exc}") from exc
    return parse_function_catalog(payload)


def validate_packet(
    *,
    file_path: Path,
    node_id: str,
    node_type: str,
    packet: dict[str, Any] | None,
    packet_path: str,
    findings: list[Finding],
) -> None:
    if not isinstance(packet, dict):
        return

    fields = packet.get("fields")
    if not isinstance(fields, list):
        fields = []

    for index, field in enumerate(fields):
        if not isinstance(field, dict):
            continue
        field_type = str(field.get("type") or "").strip()
        src = parse_uri(field.get("name", ""))
        dst = parse_uri(field.get("bindPath", ""))
        if field_type == "object":
            if (src and src.is_collection_sid) or (dst and dst.is_collection_sid):
                add_finding(
                    findings,
                    "error",
                    "collection-sid-object-conversion",
                    file_path,
                    node_id,
                    f"{packet_path}.fields[{index}]",
                    "field type 'object' cannot safely map a collection SID; leave type empty or use a list-aware mapping",
                )

            if (
                src
                and dst
                and src.is_rulemsg_datat
                and dst.is_rulemsg_datat
                and src.is_whole_object
                and dst.is_whole_object
                and is_strict_typed_sid(src.sid)
                and is_strict_typed_sid(dst.sid)
                and src.sid != dst.sid
            ):
                message = (
                    "field type 'object' cannot whole-object convert between different typed SIDs; "
                    "map fields explicitly instead of decoding one typed object into another"
                )
                if is_patch_sid(dst.sid):
                    message = (
                        "field type 'object' cannot whole-object convert a typed business object into a Patch SID; "
                        "build the patch field-by-field"
                    )
                add_finding(
                    findings,
                    "error",
                    "typed-whole-object-cross-sid-conversion",
                    file_path,
                    node_id,
                    f"{packet_path}.fields[{index}]",
                    message,
                )

    if node_type != "transform/object_mapper":
        return

    map_all = packet.get("mapAll")
    if map_all:
        return
    if len(fields) != 1 or not isinstance(fields[0], dict):
        return

    field = fields[0]
    if str(field.get("type") or "").strip():
        return

    src = parse_uri(field.get("name", ""))
    dst = parse_uri(field.get("bindPath", ""))
    if not src or not dst:
        return
    if not src.is_whole_object or not dst.is_whole_object:
        return
    if not src.sid or src.sid != dst.sid:
        return
    if src.obj_id == "" or dst.obj_id == "":
        return

    add_finding(
        findings,
        "warning",
        "object-mapper-alias-copy",
        file_path,
        node_id,
        f"{packet_path}.fields[0]",
        "object_mapper only aliases one whole object to another object with the same SID; prefer reading the original objId directly unless a true copy is required",
    )


def validate_node(file_path: Path, node: dict[str, Any], findings: list[Finding]) -> None:
    node_id = str(node.get("id") or "<unknown>")
    node_type = str(node.get("type") or "")
    config = node.get("configuration")
    if not isinstance(config, dict):
        return

    if node_type == "transform/object_mapper":
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=config.get("mappingDefinition"),
            packet_path="configuration.mappingDefinition",
            findings=findings,
        )
        return

    if node_type == "external/httpClient":
        request = config.get("request")
        if isinstance(request, dict):
            validate_packet(
                file_path=file_path,
                node_id=node_id,
                node_type=node_type,
                packet=request.get("headers"),
                packet_path="configuration.request.headers",
                findings=findings,
            )
            validate_packet(
                file_path=file_path,
                node_id=node_id,
                node_type=node_type,
                packet=request.get("queryParams"),
                packet_path="configuration.request.queryParams",
                findings=findings,
            )
            validate_packet(
                file_path=file_path,
                node_id=node_id,
                node_type=node_type,
                packet=request.get("body"),
                packet_path="configuration.request.body",
                findings=findings,
            )
        response = config.get("response")
        if isinstance(response, dict):
            validate_packet(
                file_path=file_path,
                node_id=node_id,
                node_type=node_type,
                packet=response.get("headers"),
                packet_path="configuration.response.headers",
                findings=findings,
            )
            validate_packet(
                file_path=file_path,
                node_id=node_id,
                node_type=node_type,
                packet=response.get("body"),
                packet_path="configuration.response.body",
                findings=findings,
            )
        return

    if node_type != "endpoint/http":
        return

    endpoint_def = config.get("endpointDefinition")
    if not isinstance(endpoint_def, dict):
        return

    request = endpoint_def.get("request")
    if isinstance(request, dict):
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet={"fields": request.get("pathParams", [])},
            packet_path="configuration.endpointDefinition.request.pathParams",
            findings=findings,
        )
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=request.get("queryParams"),
            packet_path="configuration.endpointDefinition.request.queryParams",
            findings=findings,
        )
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=request.get("headers"),
            packet_path="configuration.endpointDefinition.request.headers",
            findings=findings,
        )
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=request.get("body"),
            packet_path="configuration.endpointDefinition.request.body",
            findings=findings,
        )

    response = endpoint_def.get("response")
    if isinstance(response, dict):
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=response.get("headers"),
            packet_path="configuration.endpointDefinition.response.headers",
            findings=findings,
        )
        validate_packet(
            file_path=file_path,
            node_id=node_id,
            node_type=node_type,
            packet=response.get("body"),
            packet_path="configuration.endpointDefinition.response.body",
            findings=findings,
        )


def validate_function_node(
    file_path: Path,
    node: dict[str, Any],
    findings: list[Finding],
    function_catalog: dict[str, FunctionSignature] | None,
) -> None:
    if function_catalog is None:
        return

    node_id = str(node.get("id") or "<unknown>")
    node_type = str(node.get("type") or "")
    if node_type != "functions":
        return

    config = node.get("configuration")
    if not isinstance(config, dict):
        return

    function_name = str(config.get("functionName") or "").strip()
    if not function_name:
        return

    signature = function_catalog.get(function_name)
    if signature is None:
        add_finding(
            findings,
            "error",
            "function-not-found",
            file_path,
            node_id,
            "configuration.functionName",
            f"function '{function_name}' is not registered in the local function catalog",
        )
        return

    validate_function_bindings(
        file_path=file_path,
        node_id=node_id,
        function_name=function_name,
        direction="input",
        node_bindings=node.get("inputs"),
        expected=signature.inputs,
        findings=findings,
        require_all_required=True,
    )
    validate_function_bindings(
        file_path=file_path,
        node_id=node_id,
        function_name=function_name,
        direction="output",
        node_bindings=node.get("outputs"),
        expected=signature.outputs,
        findings=findings,
        require_all_required=False,
    )


def validate_function_bindings(
    *,
    file_path: Path,
    node_id: str,
    function_name: str,
    direction: str,
    node_bindings: Any,
    expected: dict[str, FunctionParam],
    findings: list[Finding],
    require_all_required: bool,
) -> None:
    if not isinstance(node_bindings, dict):
        node_bindings = {}

    issue_prefix = f"function-{direction}"
    for binding_name, binding in node_bindings.items():
        if not is_valid_function_param_name(binding_name):
            add_finding(
                findings,
                "error",
                "function-param-name-invalid",
                file_path,
                node_id,
                f"{direction}s.{binding_name}",
                (
                    f"{direction} parameter '{binding_name}' for function '{function_name}' must use semantic lower_snake_case "
                    "without digits or objId-style suffixes"
                ),
            )

        if binding_name not in expected:
            add_finding(
                findings,
                "error",
                f"{issue_prefix}-not-defined",
                file_path,
                node_id,
                f"{direction}s.{binding_name}",
                f"{direction} '{binding_name}' is not defined in function '{function_name}'",
            )
            continue

        if not isinstance(binding, dict):
            continue

        define_sid = str(binding.get("defineSid") or "").strip()
        expected_sid = expected[binding_name].sid.strip()
        if not are_function_sids_compatible(define_sid, expected_sid):
            add_finding(
                findings,
                "error",
                f"{issue_prefix}-sid-mismatch",
                file_path,
                node_id,
                f"{direction}s.{binding_name}.defineSid",
                (
                    f"{direction} '{binding_name}' SID mismatch for function '{function_name}': "
                    f"DSL uses '{define_sid}', function expects '{expected_sid}'"
                ),
            )

    if not require_all_required:
        return

    for binding_name, param in expected.items():
        if param.required and binding_name not in node_bindings:
            add_finding(
                findings,
                "error",
                "function-required-input-missing",
                file_path,
                node_id,
                f"inputs.{binding_name}",
                f"required input '{binding_name}' is missing for function '{function_name}'",
            )


def validate_json_file(
    file_path: Path,
    function_catalog: dict[str, FunctionSignature] | None = None,
) -> list[Finding]:
    findings: list[Finding] = []
    try:
        payload = json.loads(file_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        add_finding(
            findings,
            "error",
            "invalid-json",
            file_path,
            "<file>",
            "<json>",
            f"invalid JSON: {exc}",
        )
        return findings

    nodes = None
    if isinstance(payload, dict):
        if isinstance(payload.get("metadata"), dict):
            nodes = payload["metadata"].get("nodes")
        if nodes is None and isinstance(payload.get("ruleChain"), dict):
            metadata = payload["ruleChain"].get("metadata")
            if isinstance(metadata, dict):
                nodes = metadata.get("nodes")

    if not isinstance(nodes, list):
        return findings

    for node in nodes:
        if isinstance(node, dict):
            validate_node(file_path, node, findings)
            validate_function_node(file_path, node, findings, function_catalog)
    return findings


def validate_paths(
    paths: Iterable[Path],
    function_catalog: dict[str, FunctionSignature] | None = None,
) -> list[Finding]:
    findings: list[Finding] = []
    for file_path in iter_json_files(paths):
        findings.extend(validate_json_file(file_path, function_catalog=function_catalog))
    return sorted(
        findings,
        key=lambda item: (
            str(item.file_path),
            0 if item.severity == "error" else 1,
            item.node_id,
            item.packet_path,
        ),
    )


def format_finding(repo_root: Path, finding: Finding) -> str:
    try:
        display_path = finding.file_path.relative_to(repo_root)
    except ValueError:
        display_path = finding.file_path
    return (
        f"{finding.severity.upper()} {finding.code} "
        f"{display_path} node={finding.node_id} path={finding.packet_path}: {finding.message}"
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Validate Matrix DSL mappings and function node signatures for risky configuration patterns."
    )
    parser.add_argument(
        "--repo-root",
        type=Path,
        default=Path.cwd(),
        help="Repository root containing code/dsl. Defaults to the current working directory.",
    )
    parser.add_argument(
        "--dsl-root",
        type=Path,
        default=None,
        help="DSL directory to scan. Relative paths are resolved from --repo-root. Defaults to code/dsl.",
    )
    parser.add_argument(
        "--function-catalog-command",
        default=DEFAULT_FUNCTION_CATALOG_COMMAND,
        help=(
            "Command run from --repo-root to emit the function catalog JSON. "
            "Defaults to 'go run ./scripts/function_catalog'."
        ),
    )
    parser.add_argument(
        "--function-catalog-json",
        type=Path,
        default=None,
        help="Read function catalog JSON from this file instead of running --function-catalog-command.",
    )
    parser.add_argument(
        "paths",
        nargs="*",
        type=Path,
        help="JSON file or directory to scan. Relative paths are resolved from --repo-root. Defaults to --dsl-root.",
    )
    parser.add_argument(
        "--strict-warnings",
        action="store_true",
        help="Return non-zero when warnings are found.",
    )
    parser.add_argument(
        "--skip-function-signatures",
        action="store_true",
        help="Skip checking function node inputs/outputs against the local registered function catalog.",
    )
    return parser


def resolve_repo_path(repo_root: Path, path: Path) -> Path:
    if path.is_absolute():
        return path
    return repo_root / path


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    repo_root = args.repo_root.resolve()
    dsl_root = resolve_repo_path(repo_root, args.dsl_root or Path("code/dsl"))
    paths = [resolve_repo_path(repo_root, path) for path in args.paths] or [dsl_root]

    function_catalog = None
    if not args.skip_function_signatures:
        try:
            if args.function_catalog_json is not None:
                function_catalog = load_function_catalog_from_json(
                    resolve_repo_path(repo_root, args.function_catalog_json)
                )
            else:
                function_catalog = load_function_catalog(
                    repo_root,
                    command_text=args.function_catalog_command,
                )
        except RuntimeError as exc:
            print(f"ERROR function-catalog-load-failed: {exc}", file=sys.stderr)
            return 2

    findings = validate_paths(paths, function_catalog=function_catalog)

    error_count = sum(1 for finding in findings if finding.severity == "error")
    warning_count = sum(1 for finding in findings if finding.severity == "warning")

    for finding in findings:
        print(format_finding(repo_root, finding))

    if not findings:
        print("No risky rulechain mapping patterns found.")
        return 0

    print(f"Summary: {error_count} error(s), {warning_count} warning(s)")
    if error_count > 0:
        return 1
    if args.strict_warnings and warning_count > 0:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
