"""Body-free host observation trace for the forced-cidx prompt diagnostic.

This module is deliberately separate from :mod:`assistant_session_trace`.
``passive-v1`` is the frozen replay authority for the earlier availability
run; this module defines only the later policy-diagnostic trace.  Its output
contains identities, source locations, byte counts, and command hashes, but
never source bodies, command text, shell output, assistant answers, truth, or
grades.
"""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re
from typing import Any

import assistant_session_trace as passive_v1


POLICY_TRACE_PROTOCOL = "forced-cidx-policy-v2"
TRACE_SCHEMA_VERSION = 2

_TEXT_SEARCH = {"rg", "grep", "ag", "ack", "pt"}
_FILE_SEARCH = {"find", "fd", "fdfind", "locate", "ls", "tree"}
_KNOWN_FILE_READ = {"sed", "cat", "head", "tail", "awk", "nl", "less", "more"}
_PREFIX_COMMANDS = {
    "command",
    "env",
    "exec",
    "nice",
    "nohup",
    "sudo",
    "time",
    "timeout",
}
_WRAPPER_VALUE_OPTIONS = {
    "env": {"-C", "--chdir", "-S", "--split-string", "-u", "--unset"},
    "nice": {"-n", "--adjustment"},
    "sudo": {
        "-C",
        "--close-from",
        "-g",
        "--group",
        "-h",
        "--host",
        "-p",
        "--prompt",
        "-r",
        "--role",
        "-t",
        "--type",
        "-T",
        "--command-timeout",
        "-u",
        "--user",
    },
    "time": {"-f", "--format", "-o", "--output"},
    "timeout": {"-k", "--kill-after", "-s", "--signal"},
}
_XARGS_VALUE_OPTIONS = {
    "-a",
    "--arg-file",
    "-d",
    "--delimiter",
    "-E",
    "--eof",
    "-I",
    "--replace",
    "-L",
    "--max-lines",
    "-n",
    "--max-args",
    "-P",
    "--max-procs",
    "-s",
    "--max-chars",
}
_VERIFICATION_PRODUCERS = {
    "bun",
    "cargo",
    "deno",
    "go",
    "jest",
    "just",
    "make",
    "npm",
    "pnpm",
    "pytest",
    "tsc",
    "vitest",
    "yarn",
}
_PIPELINE_FILTERS = {
    "cut",
    "head",
    "jq",
    "sed",
    "sort",
    "tail",
    "tee",
    "tr",
    "uniq",
    "wc",
}
_PATH_LIST_FAMILIES = _FILE_SEARCH | {"git-ls-files", "git-ls-tree"}
_SEQUENTIAL_COMMANDS = re.compile(r"(?:;|&&|\|\|)")
_FORBIDDEN_TRACE_KEYS = {
    "aggregated_output",
    "answer",
    "body",
    "command",
    "grade",
    "output",
    "query",
    "truth",
}


def _command_sha256(command: str) -> str:
    return hashlib.sha256(command.encode("utf-8")).hexdigest()


def _after_wrapper_options(
    arguments: list[str], value_options: set[str]
) -> list[str]:
    index = 0
    while index < len(arguments):
        token = arguments[index]
        if token == "--":
            return arguments[index + 1 :]
        if token in value_options:
            index += 2
            continue
        if any(token.startswith(option + "=") for option in value_options):
            index += 1
            continue
        if token.startswith("-"):
            index += 1
            continue
        break
    return arguments[index:]


def _stage_program(stage: list[str]) -> tuple[str | None, list[str]]:
    """Return the executable and remaining arguments for one shell pipeline stage."""
    index = 0
    while index < len(stage):
        token = stage[index]
        executable = Path(token).name
        if "=" in token and not token.startswith("=") and executable == token:
            index += 1
            continue
        if executable in _PREFIX_COMMANDS:
            remaining = _after_wrapper_options(
                stage[index + 1 :], _WRAPPER_VALUE_OPTIONS.get(executable, set())
            )
            if executable == "timeout" and remaining:
                remaining = remaining[1:]
            stage = remaining
            index = 0
            continue
        return executable, stage[index + 1 :]
    return None, []


def _xargs_program(arguments: list[str]) -> str | None:
    remaining = _after_wrapper_options(arguments, _XARGS_VALUE_OPTIONS)
    return Path(remaining[0]).name if remaining else None


def is_shell_cidx_attempt(command: str) -> bool:
    """Recognize cidx as an executable/probe, not merely as command data."""
    normalized = passive_v1.normalized_shell_command(command)
    segments = [
        segment.strip()
        for segment in _SEQUENTIAL_COMMANDS.split(normalized)
        if segment.strip()
    ]
    for segment in segments:
        for stage in passive_v1._pipeline_stages(segment):
            executable, arguments = _stage_program(stage)
            if executable == "cidx":
                return True
            if executable == "xargs" and _xargs_program(arguments) == "cidx":
                return True
            if executable in {"which", "type"} and any(
                Path(argument).name == "cidx" for argument in arguments
            ):
                return True
    return False


def _git_subcommand(arguments: list[str]) -> str | None:
    index = 0
    while index < len(arguments):
        token = arguments[index]
        if token == "--":
            return arguments[index + 1] if index + 1 < len(arguments) else None
        if token.startswith("-"):
            index += 2 if token in {"-C", "-c", "--git-dir", "--work-tree"} else 1
            continue
        return token
    return None


def _stage_family(stage: list[str]) -> str | None:
    executable, arguments = _stage_program(stage)
    if executable is None:
        return None
    if executable in _TEXT_SEARCH:
        return executable
    if executable in _FILE_SEARCH:
        return executable
    if executable == "git":
        subcommand = _git_subcommand(arguments)
        if subcommand == "grep":
            return "git-grep"
        if subcommand in {"ls-files", "ls-tree"}:
            return "git-" + subcommand
    if executable == "xargs":
        for token in arguments:
            nested = Path(token).name
            if nested in _TEXT_SEARCH | _FILE_SEARCH:
                return nested
    return None


def _verification_stage(stage: list[str]) -> bool:
    executable, arguments = _stage_program(stage)
    if executable in _VERIFICATION_PRODUCERS:
        return True
    if executable != "git":
        return False
    return _git_subcommand(arguments) in {
        "diff",
        "log",
        "show",
        "status",
    }


def _segment_classification(segment: str) -> dict[str, Any]:
    stages = passive_v1._pipeline_stages(segment)
    discoveries: set[str] = set()
    ambiguous: set[str] = set()
    readers: set[str] = set()
    path_list_producer = False

    for index, stage in enumerate(stages):
        family = _stage_family(stage)
        executable, _arguments = _stage_program(stage)
        if executable in _KNOWN_FILE_READ:
            readers.add(executable)
        if family is None:
            continue
        if family in _PATH_LIST_FAMILIES:
            discoveries.add(family)
            path_list_producer = True
            continue
        if family not in _TEXT_SEARCH | {"git-grep"}:
            discoveries.add(family)
            continue
        if index == 0:
            discoveries.add(family)
            continue

        upstream = stages[:index]
        upstream_families = {
            value
            for candidate in upstream
            if (value := _stage_family(candidate)) is not None
        }
        upstream_programs = {
            executable
            for candidate in upstream
            if (executable := _stage_program(candidate)[0]) is not None
        }
        upstream_readers = upstream_programs & _KNOWN_FILE_READ
        if upstream_families:
            discoveries.update(upstream_families)
            discoveries.add(family)
            path_list_producer = path_list_producer or bool(
                upstream_families & _PATH_LIST_FAMILIES
            )
        elif upstream_readers:
            readers.update(upstream_readers)
        elif any(_verification_stage(candidate) for candidate in upstream) and all(
            _verification_stage(candidate)
            or _stage_program(candidate)[0] in _PIPELINE_FILTERS
            for candidate in upstream
        ):
            # Filtering test/build output is non-search verification, not code
            # discovery. It remains an observable non-search action.
            pass
        else:
            ambiguous.add(family)

    return {
        "discoveries": discoveries,
        "ambiguous": ambiguous,
        "readers": readers,
        "path_list_producer": path_list_producer,
    }


def classify_repository_command(command: str) -> dict[str, Any]:
    """Classify a command without retaining its text or output.

    ``ordinary_discovery`` is intentionally conservative and includes the
    policy's named command families plus common equivalents.  A known-file
    reader is not counted as discovery, even if its output later exposes source
    lines.  All remaining command executions are retained as non-search work
    so compilation/tests/formatting are observable without being policy
    violations.
    """
    normalized = passive_v1.normalized_shell_command(command)
    if is_shell_cidx_attempt(normalized):
        return {
            "kind": "shell_cidx_attempt",
            "ordinary_discovery_families": [],
            "ambiguous_discovery_families": [],
            "known_file_read_families": [],
            "plain_path_output_contract": False,
        }
    families: set[str] = set()
    ambiguous: set[str] = set()
    readers: set[str] = set()
    segments = [
        segment.strip()
        for segment in _SEQUENTIAL_COMMANDS.split(normalized)
        if segment.strip()
    ]
    path_list_contracts = []
    for segment in segments:
        classified = _segment_classification(segment)
        families.update(classified["discoveries"])
        ambiguous.update(classified["ambiguous"])
        readers.update(classified["readers"])
        path_list_contracts.append(classified["path_list_producer"])

    if families:
        return {
            "kind": "ordinary_repository_discovery",
            "ordinary_discovery_families": sorted(families),
            "ambiguous_discovery_families": sorted(ambiguous),
            "known_file_read_families": sorted(readers),
            "plain_path_output_contract": bool(path_list_contracts)
            and all(path_list_contracts),
        }
    if ambiguous:
        return {
            "kind": "ambiguous_repository_discovery",
            "ordinary_discovery_families": [],
            "ambiguous_discovery_families": sorted(ambiguous),
            "known_file_read_families": sorted(readers),
            "plain_path_output_contract": False,
        }
    if readers:
        return {
            "kind": "known_file_read",
            "ordinary_discovery_families": [],
            "ambiguous_discovery_families": [],
            "known_file_read_families": sorted(readers),
            "plain_path_output_contract": False,
        }
    return {
        "kind": "non_search_repository_action",
        "ordinary_discovery_families": [],
        "ambiguous_discovery_families": [],
        "known_file_read_families": [],
        "plain_path_output_contract": False,
    }


def _safe_plain_output_paths(output: str, root: Path) -> list[str]:
    """Record only source paths that resolve under the frozen source root."""
    paths: set[str] = set()
    for rendered in output.splitlines():
        match = passive_v1.PLAIN_SOURCE_PATH.fullmatch(rendered.rstrip("\r\n"))
        if not match:
            continue
        path = passive_v1._normalized_source_path(match.group(1))
        if path is not None and passive_v1._safe_source_bytes(root, path) is not None:
            paths.add(path)
    return sorted(paths)


def _shell_observation(
    command: str,
    output: str,
    root: Path,
    ordinal: int,
    *,
    allow_plain_paths: bool,
) -> dict[str, Any]:
    """Reuse v1's verified range attribution and add safe file-search paths."""
    observation = passive_v1.shell_observation(command, output, root, ordinal)
    discovered = set(observation["discovered_source_paths"])
    if allow_plain_paths:
        discovered.update(_safe_plain_output_paths(output, root))
    observation["discovered_source_paths"] = sorted(discovered)
    return observation


def _line_interval(data: bytes, start: int, end: int) -> tuple[int, int] | None:
    if start < 1 or end < start:
        return None
    line = 1
    offset = 0
    begin: int | None = None
    while offset < len(data):
        newline = data.find(b"\n", offset)
        line_end = len(data) if newline < 0 else newline + 1
        if line == start:
            begin = offset
        if line == end:
            return (begin, line_end) if begin is not None else None
        line += 1
        offset = line_end
    return None


def _merged_byte_count(intervals: list[tuple[int, int]]) -> int:
    total = 0
    current_start: int | None = None
    current_end = 0
    for start, end in sorted(intervals):
        if current_start is None:
            current_start, current_end = start, end
        elif start <= current_end:
            current_end = max(current_end, end)
        else:
            total += current_end - current_start
            current_start, current_end = start, end
    return total + (current_end - current_start if current_start is not None else 0)


def _intervals_by_path(root: Path, ranges: list[dict[str, Any]]) -> dict[str, list[tuple[int, int]]]:
    result: dict[str, list[tuple[int, int]]] = {}
    for value in ranges:
        path = value.get("path")
        start = value.get("start_line")
        end = value.get("end_line")
        if not isinstance(path, str) or not isinstance(start, int) or not isinstance(end, int):
            continue
        data = passive_v1._safe_source_bytes(root, path)
        interval = _line_interval(data, start, end) if data is not None else None
        if interval is not None:
            result.setdefault(path, []).append(interval)
    return result


def _unique_bytes(intervals: dict[str, list[tuple[int, int]]]) -> int:
    return sum(_merged_byte_count(values) for values in intervals.values())


def _intersection_bytes(
    left: dict[str, list[tuple[int, int]]], right: dict[str, list[tuple[int, int]]]
) -> int:
    total = 0
    for path in set(left) & set(right):
        intersections = []
        for left_start, left_end in left[path]:
            for right_start, right_end in right[path]:
                start, end = max(left_start, right_start), min(left_end, right_end)
                if end > start:
                    intersections.append((start, end))
        total += _merged_byte_count(intersections)
    return total


def _temporal_intersection_bytes(
    root: Path,
    earlier_ranges: list[dict[str, Any]],
    later_ranges: list[dict[str, Any]],
) -> int:
    """Count unique source bytes reacquired strictly after an earlier read."""
    intersections: dict[str, list[tuple[int, int]]] = {}
    for earlier in earlier_ranges:
        earlier_ordinal = earlier.get("ordinal")
        path = earlier.get("path")
        if not isinstance(earlier_ordinal, int) or not isinstance(path, str):
            continue
        data = passive_v1._safe_source_bytes(root, path)
        earlier_interval = (
            _line_interval(data, earlier.get("start_line"), earlier.get("end_line"))
            if data is not None
            and isinstance(earlier.get("start_line"), int)
            and isinstance(earlier.get("end_line"), int)
            else None
        )
        if earlier_interval is None:
            continue
        for later in later_ranges:
            if later.get("path") != path:
                continue
            later_ordinal = later.get("ordinal")
            if not isinstance(later_ordinal, int) or earlier_ordinal >= later_ordinal:
                continue
            later_interval = (
                _line_interval(data, later.get("start_line"), later.get("end_line"))
                if isinstance(later.get("start_line"), int)
                and isinstance(later.get("end_line"), int)
                else None
            )
            if later_interval is None:
                continue
            start = max(earlier_interval[0], later_interval[0])
            end = min(earlier_interval[1], later_interval[1])
            if end > start:
                intersections.setdefault(path, []).append((start, end))
    return _unique_bytes(intersections)


def _assert_body_free(value: Any, path: str = "$") -> None:
    """Fail closed if a future edit adds model-facing or source-bearing fields."""
    if isinstance(value, dict):
        forbidden = _FORBIDDEN_TRACE_KEYS & set(value)
        if forbidden:
            rendered = ", ".join(sorted(forbidden))
            raise ValueError(f"body-free trace boundary violated at {path}: {rendered}")
        for key, nested in value.items():
            _assert_body_free(nested, f"{path}.{key}")
    elif isinstance(value, list):
        for index, nested in enumerate(value):
            _assert_body_free(nested, f"{path}[{index}]")


def _sanitized_search(search: dict[str, Any], ordinal: int) -> dict[str, Any]:
    query = search.get("query")
    query_bytes = len(query.encode("utf-8")) if isinstance(query, str) else None
    locators = []
    for locator in search.get("locators", []):
        if not isinstance(locator, dict):
            continue
        locators.append(
            {
                key: locator.get(key)
                for key in (
                    "key",
                    "chunk_id",
                    "path",
                    "qualified_symbol",
                    "start_line",
                    "end_line",
                    "indexed_sha256",
                    "match_sources",
                )
            }
        )
    return {
        "ordinal": ordinal,
        "query_sha256": hashlib.sha256(query.encode("utf-8")).hexdigest()
        if isinstance(query, str)
        else None,
        "query_bytes": query_bytes,
        "requested_mode": search.get("requested_mode"),
        "effective_mode": search.get("effective_mode"),
        "effective_mode_authority": search.get("effective_mode_authority"),
        "fallback_reason": search.get("fallback_reason"),
        "k": search.get("k"),
        "max_inline_bytes": search.get("max_inline_bytes"),
        "payload_bytes": search.get("payload_bytes"),
        "locators": locators,
    }


def _sanitized_read(read: dict[str, Any], ordinal: int) -> dict[str, Any]:
    result = {
        "ordinal": ordinal,
        "key": read.get("key"),
        "requested": read.get("requested"),
        "delivered": read.get("delivered"),
        "success": read.get("success"),
        "conformance_error": read.get("conformance_error"),
        "result_code": read.get("result_code"),
        "delivered_source_bytes": read.get("delivered_source_bytes"),
        "payload_bytes": read.get("payload_bytes"),
        "overlaps_prior_successful_read_keys": read.get(
            "overlaps_prior_successful_read_keys"
        ),
    }
    if "invocation_id" in read:
        result.update(
            {
                "invocation_id": read.get("invocation_id"),
                "input_version": read.get("input_version"),
                "evidence_unit_index": read.get("evidence_unit_index"),
                "is_first_evidence_unit": read.get("is_first_evidence_unit"),
            }
        )
    return result


def _locator_identity(locator: dict[str, Any]) -> tuple[str, int, int, str] | None:
    path = locator.get("path")
    start = locator.get("start_line")
    end = locator.get("end_line")
    digest = locator.get("indexed_sha256")
    if (
        not isinstance(path, str)
        or not isinstance(start, int)
        or not isinstance(end, int)
        or not isinstance(digest, str)
    ):
        return None
    return path, start, end, digest


def _requested_read_identity(read: dict[str, Any]) -> tuple[str, int, int, str] | None:
    requested = read.get("requested")
    if not isinstance(requested, dict):
        return None
    path = requested.get("path")
    start = requested.get("start_line")
    end = requested.get("end_line")
    digest = requested.get("expected_sha256")
    if (
        not isinstance(path, str)
        or not isinstance(start, int)
        or not isinstance(end, int)
        or not isinstance(digest, str)
    ):
        return None
    return path, start, end, digest


def _read_invocation_key(read: dict[str, Any], fallback_index: int) -> str:
    """Keep one MCP call distinct from each flattened batch evidence unit."""
    value = read.get("invocation_id")
    if isinstance(value, str) and value:
        return value
    return f"ordinal:{read.get('ordinal')}:{fallback_index}"


def _batch_eligibility_before_first_source_acquisition(
    searches: list[dict[str, Any]], reads: list[dict[str, Any]]
) -> tuple[bool, int]:
    """Return whether two distinct valid locators preceded the first delivered source."""
    first_source_ordinal = min(
        (
            read["ordinal"]
            for read in reads
            if read.get("success") and _requested_read_identity(read) is not None
        ),
        default=None,
    )
    if first_source_ordinal is None:
        return False, 0
    exposed = {
        identity
        for search in searches
        if search["ordinal"] < first_source_ordinal
        for locator in search.get("locators", [])
        if isinstance(locator, dict)
        and (identity := _locator_identity(locator)) is not None
    }
    return len(exposed) >= 2, len(exposed)


def build_policy_trace(
    events_path: Path,
    final_path: Path,
    source_root: Path,
    *,
    frozen_fts_default: bool,
    include_batch_diagnostics: bool = False,
) -> dict[str, Any]:
    """Build the independently-versioned, body-free forced-policy trace."""
    base = passive_v1.build_session_trace(
        events_path, final_path, source_root, frozen_fts_default=frozen_fts_default
    )
    actions: list[dict[str, Any]] = []
    shell_by_ordinal: dict[int, dict[str, Any]] = {}
    ordinal = 0

    for event in passive_v1._read_events(events_path):
        if event.get("type") != "item.started":
            continue
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        item_type = item.get("type")
        item_id = item.get("id")
        record: dict[str, Any] | None = None
        if item_type == "mcp_tool_call" and item.get("server") == "cidx":
            tool = item.get("tool")
            if tool in {"search", "read_span"}:
                record = {
                    "ordinal": ordinal,
                    "kind": "cidx_" + str(tool),
                    "id": item_id,
                    "completed": False,
                }
            else:
                record = {
                    "ordinal": ordinal,
                    "kind": "cidx_other",
                    "id": item_id,
                    "completed": False,
                }
        elif item_type == "command_execution":
            command = str(item.get("command", ""))
            classification = classify_repository_command(command)
            record = {
                "ordinal": ordinal,
                "kind": classification["kind"],
                "id": item_id,
                "completed": False,
                "command_sha256": _command_sha256(command),
                "ordinary_discovery_families": classification[
                    "ordinary_discovery_families"
                ],
                "ambiguous_discovery_families": classification[
                    "ambiguous_discovery_families"
                ],
                "known_file_read_families": classification["known_file_read_families"],
                "plain_path_output_contract": classification[
                    "plain_path_output_contract"
                ],
            }
        elif item_type == "mcp_tool_call":
            record = {
                "ordinal": ordinal,
                "kind": "other_tool_action",
                "id": item_id,
                "completed": False,
            }
        if record is not None:
            actions.append(record)
            ordinal += 1

    action_by_id: dict[str, dict[str, Any]] = {
        str(action["id"]): action
        for action in actions
        if isinstance(action.get("id"), str)
    }
    for event in passive_v1._read_events(events_path):
        if event.get("type") != "item.completed":
            continue
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        action = action_by_id.get(str(item.get("id")))
        if action is None:
            continue
        action["completed"] = True
        if item.get("type") != "command_execution" or action["kind"] not in {
            "ordinary_repository_discovery",
            "ambiguous_repository_discovery",
            "known_file_read",
        }:
            continue
        action_ordinal = action["ordinal"]
        output = item.get("aggregated_output")
        shell_by_ordinal[action_ordinal] = _shell_observation(
            str(item.get("command", "")),
            output if isinstance(output, str) else "",
            source_root,
            action_ordinal,
            allow_plain_paths=bool(action["plain_path_output_contract"]),
        )

    base_action_ids = {
        (action.get("kind"), action.get("ordinal")): action.get("id")
        for action in base.get("actions", [])
        if isinstance(action, dict)
    }

    def mapped_ordinal(observation: dict[str, Any], kind: str) -> int:
        item_id = base_action_ids.get((kind, observation.get("ordinal")))
        action = action_by_id.get(str(item_id)) if isinstance(item_id, str) else None
        if action is None:
            raise ValueError(
                "completed cidx observation has no matching started event identity"
            )
        return int(action["ordinal"])

    searches = [
        _sanitized_search(search, mapped_ordinal(search, "cidx_search"))
        for search in base["search_observations"]
    ]
    reads = [
        _sanitized_read(read, mapped_ordinal(read, "cidx_read_span"))
        for read in base["read_observations"]
    ]
    shells = [shell_by_ordinal[key] for key in sorted(shell_by_ordinal)]

    for search in searches:
        for locator in search["locators"]:
            identity = _locator_identity(locator)
            locator["read_status"] = (
                "selected_for_read"
                if identity is not None
                and any(
                    read.get("success")
                    and search["ordinal"] < read["ordinal"]
                    and _requested_read_identity(read) == identity
                    for read in reads
                )
                else "unread"
            )

    ordinary_actions = [
        action for action in actions if action["kind"] == "ordinary_repository_discovery"
    ]
    ambiguous_actions = [
        action for action in actions if action["kind"] == "ambiguous_repository_discovery"
    ]
    shell_cidx_actions = [
        action for action in actions if action["kind"] == "shell_cidx_attempt"
    ]
    discovery_violations = ordinary_actions + ambiguous_actions + shell_cidx_actions
    cidx_search_actions = [
        action for action in actions if action["kind"] == "cidx_search"
    ]
    cidx_read_actions = [
        action for action in actions if action["kind"] == "cidx_read_span"
    ]
    first_cidx_search = next((search["ordinal"] for search in searches), None)
    first_cidx_search_attempt = next(
        (action["ordinal"] for action in cidx_search_actions), None
    )
    first_ordinary_discovery = next(
        (action["ordinal"] for action in ordinary_actions), None
    )
    first_ambiguous_discovery = next(
        (action["ordinal"] for action in ambiguous_actions), None
    )
    discovery_actions = [
        {"ordinal": action["ordinal"], "kind": "cidx_search"}
        for action in cidx_search_actions
    ] + [
        {"ordinal": action["ordinal"], "kind": "ordinary_repository_discovery"}
        for action in ordinary_actions
    ] + [
        {"ordinal": action["ordinal"], "kind": "ambiguous_repository_discovery"}
        for action in ambiguous_actions
    ] + [
        {"ordinal": action["ordinal"], "kind": "shell_cidx_attempt"}
        for action in shell_cidx_actions
    ]
    discovery_actions.sort(key=lambda value: value["ordinal"])
    first_discovery = discovery_actions[0] if discovery_actions else None

    ordinary_after = (
        sum(action["ordinal"] > first_cidx_search for action in ordinary_actions)
        if first_cidx_search is not None
        else None
    )
    ordinary_before = (
        sum(action["ordinal"] < first_cidx_search for action in ordinary_actions)
        if first_cidx_search is not None
        else None
    )
    disallowed_before = (
        sum(action["ordinal"] < first_cidx_search for action in discovery_violations)
        if first_cidx_search is not None
        else None
    )
    disallowed_after = (
        sum(action["ordinal"] > first_cidx_search for action in discovery_violations)
        if first_cidx_search is not None
        else None
    )
    selected_locator_reads = 0
    for read in reads:
        requested_identity = _requested_read_identity(read)
        if not read.get("success") or requested_identity is None:
            continue
        if any(
            search["ordinal"] < read["ordinal"]
            and any(
                _locator_identity(locator) == requested_identity
                for locator in search["locators"]
            )
            for search in searches
        ):
            selected_locator_reads += 1

    cidx_ranges = [
        {
            "ordinal": read["ordinal"],
            "path": read["delivered"].get("path"),
            "start_line": read["delivered"].get("start_line"),
            "end_line": read["delivered"].get("end_line"),
        }
        for read in reads
        if read.get("success") and isinstance(read.get("delivered"), dict)
    ]
    ordinary_ranges = [
        {**value, "ordinal": shell["ordinal"]}
        for shell in shells
        for value in shell.get("delivered_ranges", [])
        if isinstance(value, dict)
    ]
    cidx_intervals = _intervals_by_path(source_root, cidx_ranges)
    ordinary_intervals = _intervals_by_path(source_root, ordinary_ranges)
    combined_intervals = {
        path: cidx_intervals.get(path, []) + ordinary_intervals.get(path, [])
        for path in set(cidx_intervals) | set(ordinary_intervals)
    }
    read_invocation_count = len(
        {_read_invocation_key(read, index) for index, read in enumerate(reads)}
    )
    batch_read_span_invocation_count = len(
        {
            _read_invocation_key(read, index)
            for index, read in enumerate(reads)
            if read.get("input_version") == 2
        }
    )
    (
        batch_eligible_before_first_source_acquisition,
        batch_eligible_locator_count,
    ) = _batch_eligibility_before_first_source_acquisition(searches, reads)
    cidx_overlap_count = sum(
        bool(read.get("overlaps_prior_successful_read_keys"))
        for read in reads
        if read.get("success")
    )
    compliance_reasons = []
    if not searches:
        compliance_reasons.append("missing_cidx_search")
    if ordinary_actions:
        compliance_reasons.append("ordinary_repository_discovery")
    if ambiguous_actions:
        compliance_reasons.append("ambiguous_repository_discovery")
    if shell_cidx_actions:
        compliance_reasons.append("shell_cidx_attempt")
    incomplete_search_count = sum(not action["completed"] for action in cidx_search_actions)
    incomplete_read_count = sum(not action["completed"] for action in cidx_read_actions)
    if incomplete_search_count:
        compliance_reasons.append("incomplete_cidx_search_attempt")
    if incomplete_read_count:
        compliance_reasons.append("incomplete_cidx_read_span_attempt")
    if first_discovery is None or first_discovery["kind"] != "cidx_search":
        compliance_reasons.append("first_repository_discovery_not_cidx_search")
    if selected_locator_reads == 0:
        compliance_reasons.append("no_selected_locator_read_span")

    trace = {
        "schema_version": TRACE_SCHEMA_VERSION,
        "protocol": POLICY_TRACE_PROTOCOL,
        "scope": {
            "artifact": "ignored_evaluation_artifact",
            "product_server_state": False,
            "sqlite_session_state": False,
            "model_facing": False,
            "source_bodies_present": False,
            "shell_output_present": False,
            "command_text_present": False,
            "body_free_key_audit": "passed",
        },
        "actions": actions,
        "search_observations": sorted(searches, key=lambda value: value["ordinal"]),
        "read_observations": sorted(reads, key=lambda value: value["ordinal"]),
        "shell_observations": shells,
        "final_citations": base["final_citations"],
        "policy_stage": {
            "ordinary_discovery_action_count": len(ordinary_actions),
            "ordinary_discovery_by_family": {
                family: sum(family in action["ordinary_discovery_families"] for action in ordinary_actions)
                for family in sorted(
                    {family for action in ordinary_actions for family in action["ordinary_discovery_families"]}
                )
            },
            "ambiguous_discovery_action_count": len(ambiguous_actions),
            "ambiguous_discovery_by_family": {
                family: sum(
                    family in action["ambiguous_discovery_families"]
                    for action in ambiguous_actions
                )
                for family in sorted(
                    {
                        family
                        for action in ambiguous_actions
                        for family in action["ambiguous_discovery_families"]
                    }
                )
            },
            "shell_cidx_attempt_count": len(shell_cidx_actions),
            "known_file_read_action_count": sum(
                action["kind"] == "known_file_read" for action in actions
            ),
            "non_search_repository_action_count": sum(
                action["kind"] == "non_search_repository_action" for action in actions
            ),
            "first_cidx_search_ordinal": first_cidx_search,
            "first_cidx_search_attempt_ordinal": first_cidx_search_attempt,
            "first_ordinary_discovery_ordinal": first_ordinary_discovery,
            "first_ambiguous_discovery_ordinal": first_ambiguous_discovery,
            "first_repository_discovery": first_discovery,
            "ordinary_discovery_before_first_cidx_search_count": ordinary_before,
            "ordinary_discovery_after_first_cidx_search_count": ordinary_after,
            "disallowed_discovery_before_first_cidx_search_count": disallowed_before,
            "disallowed_discovery_after_first_cidx_search_count": disallowed_after,
            "ordinary_discovery_without_cidx_search_count": len(ordinary_actions)
            if first_cidx_search is None
            else 0,
            "ambiguous_discovery_without_cidx_search_count": len(ambiguous_actions)
            if first_cidx_search is None
            else 0,
            "cidx_search_count": len(searches),
            "cidx_search_attempt_count": len(cidx_search_actions),
            "incomplete_cidx_search_attempt_count": incomplete_search_count,
            "cidx_read_span_count": read_invocation_count,
            "cidx_evidence_unit_count": len(reads),
            "cidx_read_span_attempt_count": len(cidx_read_actions),
            "incomplete_cidx_read_span_attempt_count": incomplete_read_count,
            "selected_locator_read_span_count": selected_locator_reads,
            "exact_cidx_read_span_count": sum(bool(read.get("success")) for read in reads),
            "overlapping_successful_cidx_read_count": cidx_overlap_count,
            "ordinary_verified_range_count": len(ordinary_ranges),
            "ordinary_fully_attributed_command_count": sum(
                shell.get("range_attribution") == "FULL" for shell in shells
            ),
            "cidx_unique_source_bytes": _unique_bytes(cidx_intervals),
            "ordinary_unique_source_bytes": _unique_bytes(ordinary_intervals),
            "combined_unique_source_bytes": _unique_bytes(combined_intervals),
            "cidx_ordinary_overlap_source_bytes": _intersection_bytes(
                cidx_intervals, ordinary_intervals
            ),
            "cidx_ordinary_reacquired_source_bytes": (
                _temporal_intersection_bytes(source_root, cidx_ranges, ordinary_ranges)
            ),
            "directed_policy": {
                "has_cidx_search": bool(searches),
                "has_no_ordinary_repository_discovery": not discovery_violations,
                "first_repository_discovery_is_cidx_search": bool(first_discovery)
                and first_discovery["kind"] == "cidx_search",
                "has_selected_locator_read_span": selected_locator_reads > 0,
                "would_mechanically_comply_if_directed": not compliance_reasons,
                "failure_reasons": compliance_reasons,
            },
        },
    }
    if not include_batch_diagnostics and not any("invocation_id" in read for read in reads):
        # Preserve the frozen scalar-v1 trace shape for historical replays.
        trace["policy_stage"].pop("cidx_evidence_unit_count")
    else:
        trace["policy_stage"].update(
            {
                "batch_read_span_invocation_count": batch_read_span_invocation_count,
                "batch_eligible_before_first_source_acquisition": (
                    batch_eligible_before_first_source_acquisition
                ),
                "batch_eligible_locator_count": batch_eligible_locator_count,
            }
        )
    _assert_body_free(trace)
    return trace
