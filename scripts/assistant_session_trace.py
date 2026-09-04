"""Passive, host-side assistant observation trace.

The trace is an ignored evaluation artifact. It never becomes product, MCP,
server, or SQLite state, and it contains neither source bodies nor truth,
usefulness, grading, or claim judgments.
"""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re
import shlex
from typing import Any


TRACE_SCHEMA_VERSION = 1
PASSIVE_TRACE_PROTOCOL = "passive-v1"
LEGACY_TRACE_PROTOCOLS = {
    "assistant-ab-chi-rhf-v3": "legacy-v3",
    "assistant-ab-chi-rhf-v4": "legacy-v4",
    "assistant-ab-chi-rhf-v5": "legacy-v5",
    "assistant-ab-chi-rhf-v6": "legacy-v6",
}

SHELL_NAMES = {"sh", "bash", "zsh", "dash", "ksh"}
INSPECTION = re.compile(
    r"(?i)(?:^|[;&|()\s])"
    r"(?:rg|grep|find|fd|ls|tree|sed|cat|head|tail|awk|nl)(?:\s|$)"
    r"|git\s+grep"
)
SOURCE_SUFFIX_PATTERN = r"(?:go|ts|tsx)"
RG_LINE = re.compile(
    rf"^(?:\./)?([A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN}):(\d+):(.*)$"
)
PLAIN_SOURCE_PATH = re.compile(
    rf"^(?:\./)?([A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})$"
)
NL_LINE = re.compile(r"^\s*(\d+)\t(.*)$")
NUMBERED_SOURCE_LINE = re.compile(r"^(\d+):(.*)$")
NL_PATH = re.compile(
    rf"nl\s+-ba\s+((?:\./)?[A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})"
)
SIMPLE_SED = re.compile(
    rf"sed\s+-n\s+['\"](\d+),(\d+)p['\"]\s+"
    rf"((?:\./)?[A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})"
)
COMMAND_SOURCE_PATH = re.compile(
    rf"(?<![A-Za-z0-9_@+./-])"
    rf"((?:\./)?[A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})"
    rf"(?![A-Za-z0-9_@+./-])"
)
SIMPLE_CAT = re.compile(
    rf"cat\s+(?:--\s+)?"
    rf"((?:\./)?[A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})"
)
SIMPLE_HEAD_TAIL = re.compile(
    rf"(head|tail)(?:\s+(?:-n\s+)?-(?P<negative>\d+)"
    rf"|\s+-n\s+(?P<count>\d+))?\s+"
    rf"(?P<path>(?:\./)?[A-Za-z0-9_@+./-]+\.{SOURCE_SUFFIX_PATTERN})"
)
COMMAND_SEQUENCE = re.compile(r"(?:;|&&|\|\|)")
RG_GREP_COMMAND = re.compile(
    r"(?i)(?:^|[|()\s])(?:rg|grep)(?:\s|$)"
    r"|(?:^|[|()\s])git\s+grep(?:\s|$)"
)
NL_COMMAND = re.compile(r"(?i)(?:^|[|()\s])nl(?:\s|$)")
PATH_LIST_COMMAND = re.compile(
    r"(?i)(?:^|[|()\s])(?:find|fd|ls|tree)(?:\s|$)"
)


def resolve_trace_protocol(manifest: dict[str, Any]) -> str:
    """Resolve the manifest's trace protocol or fail closed."""
    controls = manifest.get("controls")
    configured = controls.get("session_trace_protocol") if isinstance(controls, dict) else None
    if configured is not None:
        if configured != PASSIVE_TRACE_PROTOCOL:
            raise ValueError(f"unknown session_trace_protocol: {configured}")
        return configured
    protocol = LEGACY_TRACE_PROTOCOLS.get(manifest.get("manifest_id"))
    if protocol is None:
        raise ValueError("manifest must declare session_trace_protocol")
    return protocol


def normalized_shell_command(command: str) -> str:
    """Unwrap a recorded shell ``-c`` invocation."""
    current = command
    for _ in range(4):
        try:
            parts = shlex.split(current, posix=True)
        except ValueError:
            return current
        if not parts or Path(parts[0]).name not in SHELL_NAMES:
            return current
        script = next(
            (
                parts[index + 1]
                for index, token in enumerate(parts[1:], 1)
                if token.startswith("-")
                and "c" in token[1:]
                and index + 1 < len(parts)
            ),
            None,
        )
        if script is None or script == current:
            return current
        current = script
    return current


def is_repository_inspection(command: str) -> bool:
    return bool(INSPECTION.search(normalized_shell_command(command)))


def _all_output_segments_match(command: str, pattern: re.Pattern[str]) -> bool:
    """Require every sequential output stream to contain the expected producer.

    Pipelines stay in one segment because filters such as ``head`` or ``sed``
    preserve the upstream format. Sequential commands can concatenate unrelated
    output, so a mixed segment is deliberately left unattributed.
    """
    segments = [
        segment.strip()
        for segment in COMMAND_SEQUENCE.split(normalized_shell_command(command))
        if segment.strip()
    ]
    return bool(segments) and all(pattern.search(segment) for segment in segments)


def _pipeline_stages(segment: str) -> list[list[str]]:
    try:
        lexer = shlex.shlex(segment, posix=True, punctuation_chars="|")
        lexer.whitespace_split = True
        lexer.commenters = ""
        tokens = list(lexer)
    except ValueError:
        return []
    stages: list[list[str]] = [[]]
    for token in tokens:
        if token == "|":
            stages.append([])
        elif token != "||":
            stages[-1].append(token)
    return [stage for stage in stages if stage]


def _search_stage(segment: str) -> tuple[str, list[str], bool] | None:
    stages = _pipeline_stages(segment)
    for stage_index in range(len(stages) - 1, -1, -1):
        stage = stages[stage_index]
        for index, token in enumerate(stage):
            executable = Path(token).name
            if executable in {"rg", "grep"}:
                return executable, stage[index + 1 :], stage_index > 0
            if executable == "git":
                try:
                    grep_index = stage.index("grep", index + 1)
                except ValueError:
                    continue
                return "git-grep", stage[grep_index + 1 :], stage_index > 0
    return None


def _option_enabled(arguments: list[str], short: str, long: str) -> bool:
    for token in arguments:
        if token == "--":
            break
        if token == long or token.startswith(long + "="):
            return True
        if (
            token.startswith("-")
            and not token.startswith("--")
            and token != "-"
            and short in token[1:]
        ):
            return True
    return False


def encoded_json_bytes(value: Any) -> int:
    return len(
        json.dumps(
            value,
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
    )


def line_range(value: dict[str, Any]) -> tuple[int, int] | None:
    start = value.get("start_line")
    end = value.get("end_line")
    if isinstance(start, int) and isinstance(end, int) and start > 0 and end >= start:
        return start, end
    parent = value.get("parent_range")
    return line_range(parent) if isinstance(parent, dict) else None


def locator_key(digest: str, path: str, symbol: str, start: int, end: int) -> str:
    return "\0".join((digest, path, symbol, str(start), str(end)))


def locator_from_hit(hit: dict[str, Any]) -> dict[str, Any] | None:
    lines = line_range(hit)
    path = hit.get("path")
    digest = hit.get("indexed_sha256")
    symbol = hit.get("qualified_symbol")
    if lines is None or not all(isinstance(value, str) for value in (path, digest, symbol)):
        return None
    sources = hit.get("match_sources", hit.get("lexical_sources", []))
    return {
        "key": locator_key(digest, path, symbol, *lines),
        "chunk_id": hit.get("chunk_id"),
        "path": path,
        "qualified_symbol": symbol,
        "start_line": lines[0],
        "end_line": lines[1],
        "indexed_sha256": digest,
        "match_sources": (
            sorted({value for value in sources if isinstance(value, str)})
            if isinstance(sources, list)
            else []
        ),
    }


def tool_result_parts(item: dict[str, Any]) -> tuple[Any, int, int, int]:
    result = item.get("result")
    if not isinstance(result, dict):
        return None, 0, 0, 0
    structured = result.get("structuredContent", result.get("structured_content"))
    text_bytes = 0
    decoded = None
    content = result.get("content")
    for block in content if isinstance(content, list) else []:
        if not isinstance(block, dict) or not isinstance(block.get("text"), str):
            continue
        text_bytes += len(block["text"].encode("utf-8"))
        if decoded is None:
            try:
                decoded = json.loads(block["text"])
            except json.JSONDecodeError:
                pass
    return (
        structured if structured is not None else decoded,
        encoded_json_bytes(structured) if structured is not None else 0,
        text_bytes,
        encoded_json_bytes(result),
    )


def _safe_source_bytes(root: Path, path: str) -> bytes | None:
    normalized_root = root.resolve()
    target = (root / path).resolve()
    try:
        target.relative_to(normalized_root)
    except ValueError:
        return None
    try:
        return target.read_bytes()
    except OSError:
        return None


def _normalized_source_path(value: str) -> str | None:
    current = value
    while current.startswith("./"):
        current = current[2:]
    if not current or current.startswith("/"):
        return None
    if any(part in {"", ".", ".."} for part in current.split("/")):
        return None
    return current


def _safe_source_lines(root: Path, path: str) -> list[str] | None:
    data = _safe_source_bytes(root, path)
    if data is None:
        return None
    try:
        return data.decode("utf-8").splitlines()
    except UnicodeDecodeError:
        return None


def _go_line_span(data: bytes, start: int, end: int) -> bytes | None:
    """Mirror internal/app.lines, including retained line-ending bytes."""
    if start <= 0 or end < start:
        return None
    line = 1
    begin = 0
    offset = 0
    while offset < len(data):
        newline = data.find(b"\n", offset)
        line_end = len(data) if newline < 0 else newline + 1
        if line == start:
            begin = offset
        if line == end:
            return data[begin:line_end]
        line += 1
        offset = line_end
    return None


def _source_line_count(data: bytes) -> int:
    if not data:
        return 0
    return data.count(b"\n") + (0 if data.endswith(b"\n") else 1)


def _merge_ranges(values: list[dict[str, Any]]) -> list[dict[str, Any]]:
    merged: list[dict[str, Any]] = []
    for item in sorted(
        values,
        key=lambda value: (value["path"], value["start_line"], value["end_line"]),
    ):
        if (
            merged
            and item["path"] == merged[-1]["path"]
            and item["start_line"] <= merged[-1]["end_line"] + 1
        ):
            merged[-1]["end_line"] = max(merged[-1]["end_line"], item["end_line"])
        else:
            merged.append(dict(item))
    return merged


def _verified_line_match(
    root: Path,
    candidates: list[str],
    number: int,
    content: str,
) -> str | None:
    matches = []
    for path in candidates:
        lines = _safe_source_lines(root, path)
        if lines is not None and number <= len(lines) and lines[number - 1] == content:
            matches.append(path)
    return matches[0] if len(matches) == 1 else None


def _command_source_paths(command: str, root: Path) -> list[str]:
    paths: list[str] = []
    for matched in COMMAND_SOURCE_PATH.finditer(normalized_shell_command(command)):
        path = _normalized_source_path(matched.group(1))
        if (
            path is not None
            and path not in paths
            and _safe_source_bytes(root, path) is not None
        ):
            paths.append(path)
    return paths


def _existing_file_arguments(arguments: list[str], root: Path) -> set[Path]:
    files: set[Path] = set()
    for token in arguments:
        if token.startswith("-"):
            continue
        candidate = (root / token).resolve()
        try:
            candidate.relative_to(root.resolve())
        except ValueError:
            continue
        if candidate.is_file():
            files.add(candidate)
    return files


def _search_output_contract(
    command: str,
    root: Path,
) -> tuple[bool, bool, bool]:
    """Return line-number, filename+line, and path-only guarantees.

    Every sequential stream must have the same guarantee. A pipeline may feed
    stdin into a search command, in which case filename output is not inferred.
    """
    segments = [
        segment.strip()
        for segment in COMMAND_SEQUENCE.split(normalized_shell_command(command))
        if segment.strip()
    ]
    contracts: list[tuple[bool, bool, bool]] = []
    for segment in segments:
        stage = _search_stage(segment)
        if stage is None:
            return False, False, False
        executable, arguments, piped_input = stage
        line_number = _option_enabled(arguments, "n", "--line-number")
        path_only = (
            _option_enabled(arguments, "l", "--files-with-matches")
            or "--files" in arguments
            or "--files-without-match" in arguments
        )
        explicit_filename = _option_enabled(arguments, "H", "--with-filename")
        no_filename = (
            _option_enabled(arguments, "I", "--no-filename")
            if executable == "rg"
            else _option_enabled(arguments, "h", "--no-filename")
        )
        input_files = _existing_file_arguments(arguments, root)
        if executable == "git-grep":
            filename = not no_filename
        elif explicit_filename:
            filename = not no_filename
        elif piped_input or "-" in arguments or len(input_files) == 1:
            filename = False
        elif executable == "rg":
            filename = not no_filename
        else:
            recursive = _option_enabled(arguments, "r", "--recursive")
            filename = not no_filename and (recursive or len(input_files) > 1)
        contracts.append((line_number, line_number and filename, path_only))
    if not contracts:
        return False, False, False
    return tuple(all(contract[index] for contract in contracts) for index in range(3))


def _discovered_paths(
    output: str,
    root: Path,
    *,
    allow_plain_paths: bool,
    allow_rg_lines: bool,
) -> set[str]:
    discovered: set[str] = set()
    for rendered in output.splitlines():
        plain = PLAIN_SOURCE_PATH.fullmatch(rendered) if allow_plain_paths else None
        if plain:
            path = _normalized_source_path(plain.group(1))
            if path is not None and _safe_source_bytes(root, path) is not None:
                discovered.add(path)
            continue
        matched = RG_LINE.fullmatch(rendered) if allow_rg_lines else None
        if matched:
            path = _normalized_source_path(matched.group(1))
            if path is not None and _safe_source_bytes(root, path) is not None:
                discovered.add(path)
    return discovered


def _range_source_bytes(root: Path, ranges: list[dict[str, Any]]) -> int:
    total = 0
    for value in ranges:
        data = _safe_source_bytes(root, value["path"])
        span = (
            _go_line_span(data, value["start_line"], value["end_line"])
            if data is not None
            else None
        )
        if span is not None:
            total += len(span)
    return total


def shell_observation(
    command: str,
    output: str,
    root: Path,
    ordinal: int,
) -> dict[str, Any]:
    """Attribute only source-verified output; leave all other bytes unmatched."""
    ranges: list[dict[str, Any]] = []
    attributed_output_lines: set[int] = set()
    rendered_lines = output.splitlines(keepends=True)
    compact = normalized_shell_command(command).strip()
    rg_output = _all_output_segments_match(compact, RG_GREP_COMMAND)
    numbered_search_output, filename_search_output, path_search_output = (
        _search_output_contract(compact, root) if rg_output else (False, False, False)
    )
    nl_output = _all_output_segments_match(compact, NL_COMMAND)
    path_list_output = _all_output_segments_match(compact, PATH_LIST_COMMAND)

    # ``rg -n`` output identifies its own path and line and is verified against
    # the source checkout before it contributes a range.
    if filename_search_output:
        for index, rendered_with_ending in enumerate(rendered_lines):
            rendered = rendered_with_ending.rstrip("\r\n")
            match = RG_LINE.fullmatch(rendered)
            if not match:
                continue
            path = _normalized_source_path(match.group(1))
            if path is None:
                continue
            number = int(match.group(2))
            content = match.group(3)
            lines = _safe_source_lines(root, path)
            if lines is not None and number <= len(lines) and lines[number - 1] == content:
                ranges.append({"path": path, "start_line": number, "end_line": number})
                attributed_output_lines.add(index)

    # A compound command can emit several indistinguishable ``nl`` streams.
    # Attribute a numbered output line only when exactly one mentioned source
    # file has that exact source line. Ambiguous lines remain unattributed.
    nl_paths = list(
        dict.fromkeys(
            normalized
            for path in NL_PATH.findall(normalized_shell_command(command))
            if (normalized := _normalized_source_path(path)) is not None
        )
    )
    if nl_output:
        for index, rendered_with_ending in enumerate(rendered_lines):
            rendered = rendered_with_ending.rstrip("\r\n")
            match = NL_LINE.fullmatch(rendered)
            if not match or not nl_paths:
                continue
            number = int(match.group(1))
            path = _verified_line_match(root, nl_paths, number, match.group(2))
            if path is not None:
                ranges.append({"path": path, "start_line": number, "end_line": number})
                attributed_output_lines.add(index)

    # ``rg -n`` or ``grep -n`` can omit a filename when a single source path
    # is searched. The command supplies candidates, but attribution still
    # requires an exact, unique source-line match in the emitted output.
    if numbered_search_output:
        command_paths = _command_source_paths(command, root)
        for index, rendered_with_ending in enumerate(rendered_lines):
            rendered = rendered_with_ending.rstrip("\r\n")
            match = NUMBERED_SOURCE_LINE.fullmatch(rendered)
            if not match or not command_paths:
                continue
            number = int(match.group(1))
            path = _verified_line_match(root, command_paths, number, match.group(2))
            if path is not None:
                ranges.append({"path": path, "start_line": number, "end_line": number})
                attributed_output_lines.add(index)

    # A single plain ``sed -n a,bp path`` can be verified as one exact body.
    simple = SIMPLE_SED.fullmatch(compact)
    if simple:
        start = int(simple.group(1))
        end = int(simple.group(2))
        path = _normalized_source_path(simple.group(3))
        if path is not None:
            data = _safe_source_bytes(root, path)
            expected = _go_line_span(data, start, end) if data is not None else None
            if expected is not None and output.encode("utf-8") == expected:
                ranges.append({"path": path, "start_line": start, "end_line": end})
                attributed_output_lines.update(range(len(rendered_lines)))

    # Exact single-file cat/head/tail commands are also mechanically
    # attributable. Compound or option-rich variants remain unattributed.
    cat = SIMPLE_CAT.fullmatch(compact)
    if cat:
        path = _normalized_source_path(cat.group(1))
        data = _safe_source_bytes(root, path) if path is not None else None
        if data is not None and output.encode("utf-8") == data and data:
            ranges.append(
                {
                    "path": path,
                    "start_line": 1,
                    "end_line": _source_line_count(data),
                }
            )
            attributed_output_lines.update(range(len(rendered_lines)))

    head_tail = SIMPLE_HEAD_TAIL.fullmatch(compact)
    if head_tail:
        path = _normalized_source_path(head_tail.group("path"))
        data = _safe_source_bytes(root, path) if path is not None else None
        count_text = head_tail.group("negative") or head_tail.group("count")
        count = int(count_text) if count_text is not None else 10
        line_count = _source_line_count(data) if data is not None else 0
        if head_tail.group(1) == "head":
            start = 1
            end = min(count, line_count)
        else:
            start = max(1, line_count - count + 1)
            end = line_count
        expected = (
            _go_line_span(data, start, end)
            if data is not None and line_count > 0 and count > 0
            else b""
        )
        if expected is not None and output.encode("utf-8") == expected:
            if end >= start:
                ranges.append({"path": path, "start_line": start, "end_line": end})
            attributed_output_lines.update(range(len(rendered_lines)))

    ranges = _merge_ranges(ranges)
    discovered = _discovered_paths(
        output,
        root,
        allow_plain_paths=path_search_output or path_list_output,
        allow_rg_lines=filename_search_output,
    )
    discovered.update(value["path"] for value in ranges)
    output_bytes = len(output.encode("utf-8"))
    attributed_output_bytes = sum(
        len(rendered_lines[index].encode("utf-8"))
        for index in attributed_output_lines
    )
    return {
        "ordinal": ordinal,
        "command_sha256": hashlib.sha256(command.encode("utf-8")).hexdigest(),
        "output_bytes": output_bytes,
        "attributed_output_bytes": attributed_output_bytes,
        "unattributed_output_bytes": output_bytes - attributed_output_bytes,
        "discovered_source_paths": sorted(discovered),
        "delivered_ranges": ranges,
        "attributed_source_bytes": _range_source_bytes(root, ranges),
        "range_attribution": (
            "FULL"
            if output_bytes > 0 and attributed_output_bytes == output_bytes
            else "PARTIAL"
            if attributed_output_bytes > 0
            else "NOT_OBSERVED"
        ),
        "named_parent_attribution": "NOT_OBSERVED",
        "named_parent_reason": "ordinary_tool_output_has_no_canonical_parent_identity",
    }


def final_citations(path: Path) -> list[dict[str, Any]]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return []
    result = []
    evidence_values = value.get("evidence", []) if isinstance(value, dict) else []
    for index, evidence in enumerate(evidence_values):
        if not isinstance(evidence, dict) or not isinstance(evidence.get("path"), str):
            continue
        lines = line_range(evidence)
        normalized_path = _normalized_source_path(evidence["path"])
        if lines and normalized_path is not None:
            result.append(
                {
                    "evidence_index": index,
                    "path": normalized_path,
                    "start_line": lines[0],
                    "end_line": lines[1],
                }
            )
    return result


def _read_events(path: Path) -> list[dict[str, Any]]:
    events = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            value = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(value, dict):
            events.append(value)
    return events


def _read_conformance(
    root: Path,
    *,
    requested_path: Any,
    start: Any,
    end: Any,
    expected_digest: Any,
    delivered_path: Any,
    delivered_start: Any,
    delivered_end: Any,
    delivered_digest: Any,
    body: Any,
    no_error: bool,
) -> tuple[bool, str | None, int]:
    identity = (
        delivered_path == requested_path
        and delivered_start == start
        and delivered_end == end
        and delivered_digest == expected_digest
    )
    typed = (
        isinstance(body, str)
        and isinstance(requested_path, str)
        and isinstance(start, int)
        and isinstance(end, int)
        and isinstance(expected_digest, str)
    )
    if not no_error or not typed:
        return False, "READ_FAILURE", 0
    if not identity:
        return False, "IDENTITY_MISMATCH", 0
    data = _safe_source_bytes(root, requested_path)
    if data is None:
        return False, "SOURCE_NOT_AVAILABLE", 0
    if hashlib.sha256(data).hexdigest() != expected_digest:
        return False, "SOURCE_DIGEST_MISMATCH", 0
    expected_body = _go_line_span(data, start, end)
    if expected_body is None or body.encode("utf-8") != expected_body:
        return False, "SOURCE_BODY_MISMATCH", 0
    return True, None, len(expected_body)


def build_session_trace(
    events_path: Path,
    final_path: Path,
    source_root: Path,
    *,
    frozen_fts_default: bool,
) -> dict[str, Any]:
    actions: list[dict[str, Any]] = []
    searches: list[dict[str, Any]] = []
    reads: list[dict[str, Any]] = []
    shells: list[dict[str, Any]] = []
    started: dict[str, int] = {}
    ordinal = 0

    for event in _read_events(events_path):
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        kind = item.get("type")
        item_id = item.get("id")
        command = str(item.get("command", ""))

        if event.get("type") == "item.started":
            action = None
            if (
                kind == "mcp_tool_call"
                and item.get("server") == "cidx"
                and item.get("tool") in ("search", "read_span")
            ):
                action = "cidx_" + item["tool"]
            elif kind == "command_execution" and is_repository_inspection(command):
                action = "shell_repository_inspection"
            if action is not None:
                if isinstance(item_id, str):
                    started[item_id] = ordinal
                actions.append({"ordinal": ordinal, "kind": action, "id": item_id})
                ordinal += 1
            continue

        if event.get("type") != "item.completed":
            continue
        action_ordinal = started.get(str(item_id), ordinal)
        if kind == "command_execution" and is_repository_inspection(command):
            output = item.get("aggregated_output")
            shells.append(
                shell_observation(
                    command,
                    output if isinstance(output, str) else "",
                    source_root,
                    action_ordinal,
                )
            )
            continue
        if (
            kind != "mcp_tool_call"
            or item.get("server") != "cidx"
            or item.get("tool") not in ("search", "read_span")
        ):
            continue

        tool = item["tool"]
        payload, structured_bytes, text_bytes, result_bytes = tool_result_parts(item)
        arguments = item.get("arguments")
        arguments = arguments if isinstance(arguments, dict) else {}
        payload_bytes = {
            "structured": structured_bytes,
            "text": text_bytes,
            "event_result": result_bytes,
        }
        if tool == "search":
            requested_mode = arguments.get("mode")
            if requested_mode == "fts":
                effective_mode = "fts"
                authority = "explicit_request"
                fallback_reason = None
            elif requested_mode is None and frozen_fts_default:
                effective_mode = "fts"
                authority = "frozen_fts_default_config"
                fallback_reason = None
            elif requested_mode == "hybrid":
                effective_mode = None
                authority = "not_authoritative"
                fallback_reason = "hybrid_request"
            else:
                effective_mode = None
                authority = "not_observed"
                fallback_reason = None
            hits = payload.get("results", []) if isinstance(payload, dict) else []
            locators = [
                locator
                for hit in hits
                if isinstance(hit, dict) and (locator := locator_from_hit(hit))
            ]
            searches.append(
                {
                    "ordinal": action_ordinal,
                    "query": arguments.get("query"),
                    "requested_mode": requested_mode,
                    "effective_mode": effective_mode,
                    "effective_mode_authority": authority,
                    "fallback_reason": fallback_reason,
                    "k": arguments.get("k"),
                    "max_inline_bytes": arguments.get("max_inline_bytes"),
                    "payload_bytes": payload_bytes,
                    "locators": locators,
                }
            )
            continue

        requested_path = arguments.get("path")
        start = arguments.get("start_line")
        end = arguments.get("end_line")
        expected_digest = arguments.get("expected_sha256")
        delivered_path = payload.get("path") if isinstance(payload, dict) else None
        delivered_start = payload.get("start_line") if isinstance(payload, dict) else None
        delivered_end = payload.get("end_line") if isinstance(payload, dict) else None
        delivered_digest = (
            payload.get("indexed_sha256") if isinstance(payload, dict) else None
        )
        body = payload.get("body") if isinstance(payload, dict) else None
        result = item.get("result")
        no_error = isinstance(result, dict) and not result.get(
            "isError", result.get("is_error", False)
        )
        success, conformance_error, delivered_source_bytes = _read_conformance(
            source_root,
            requested_path=requested_path,
            start=start,
            end=end,
            expected_digest=expected_digest,
            delivered_path=delivered_path,
            delivered_start=delivered_start,
            delivered_end=delivered_end,
            delivered_digest=delivered_digest,
            body=body,
            no_error=no_error,
        )
        reads.append(
            {
                "ordinal": action_ordinal,
                "requested": {
                    "path": requested_path,
                    "start_line": start,
                    "end_line": end,
                    "expected_sha256": expected_digest,
                },
                "delivered": {
                    "path": delivered_path,
                    "start_line": delivered_start,
                    "end_line": delivered_end,
                    "indexed_sha256": delivered_digest,
                },
                "success": success,
                "conformance_error": conformance_error,
                "result_code": payload.get("code") if isinstance(payload, dict) else None,
                "delivered_source_bytes": delivered_source_bytes,
                "payload_bytes": payload_bytes,
                "overlaps_prior_successful_read_keys": [],
            }
        )

    prior: list[dict[str, Any]] = []
    for read in reads:
        requested = read["requested"]
        key = "\0".join(
            str(requested.get(value, ""))
            for value in ("expected_sha256", "path", "start_line", "end_line")
        )
        read["key"] = key
        if read["success"]:
            read["overlaps_prior_successful_read_keys"] = [
                old["key"]
                for old in prior
                if old["requested"]["path"] == requested["path"]
                and old["requested"]["start_line"] <= requested["end_line"]
                and requested["start_line"] <= old["requested"]["end_line"]
            ]
            prior.append(read)

    for search in searches:
        for locator in search["locators"]:
            expected_read = {
                "path": locator["path"],
                "start_line": locator["start_line"],
                "end_line": locator["end_line"],
                "expected_sha256": locator["indexed_sha256"],
            }
            locator["read_status"] = (
                "selected_for_read"
                if any(read["success"] and read["requested"] == expected_read for read in reads)
                else "unread"
            )

    return {
        "schema_version": TRACE_SCHEMA_VERSION,
        "scope": {
            "artifact": "ignored_evaluation_artifact",
            "product_server_state": False,
            "sqlite_session_state": False,
            "model_facing": False,
        },
        "actions": sorted(actions, key=lambda value: value["ordinal"]),
        "search_observations": sorted(searches, key=lambda value: value["ordinal"]),
        "read_observations": sorted(reads, key=lambda value: value["ordinal"]),
        "shell_observations": sorted(shells, key=lambda value: value["ordinal"]),
        "final_citations": final_citations(final_path),
    }
