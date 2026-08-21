#!/usr/bin/env python3
"""Prepare blind grading packets and aggregate a frozen assistant A/B run."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import random
import re
import shlex
import statistics
from typing import Any


ARMS = ("baseline", "cidx_fts")
OUTCOMES = {"complete", "partial", "incorrect", "ungradable"}
SOURCE_SUFFIXES = (".go", ".ts", ".tsx")
REPOSITORY_INSPECTION_RE = re.compile(
    r"(?i)(?:^|[;&|()\s])(?:rg|grep|find|fd|ls|tree|sed|cat|head|tail|awk|nl)(?:\s|$)|git\s+grep"
)
SHELL_NAMES = {"sh", "bash", "zsh", "dash", "ksh"}


class ScoreError(RuntimeError):
    pass


def normalized_shell_command(command: str) -> str:
    """Unwrap a recorded shell -c invocation before command classification."""
    current = command
    for _ in range(4):
        try:
            parts = shlex.split(current, posix=True)
        except ValueError:
            return current
        if not parts or Path(parts[0]).name not in SHELL_NAMES:
            return current
        script: str | None = None
        for index, token in enumerate(parts[1:], start=1):
            if token.startswith("-") and "c" in token[1:] and index + 1 < len(parts):
                script = parts[index + 1]
                break
        if script is None or script == current:
            return current
        current = script
    return current


def is_repository_inspection(command: str) -> bool:
    return bool(REPOSITORY_INSPECTION_RE.search(normalized_shell_command(command)))


def encoded_json_bytes(value: Any) -> int:
    return len(
        json.dumps(
            value,
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
    )


def read_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ScoreError(f"cannot read JSON {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise ScoreError(f"expected JSON object: {path}")
    return value


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(
        json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    temporary.replace(path)


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def resolve(project_root: Path, value: str) -> Path:
    path = Path(value)
    if not path.is_absolute():
        path = project_root / path
    return path.resolve()


def load_context(args: argparse.Namespace) -> dict[str, Any]:
    project_root = Path(__file__).resolve().parent.parent
    manifest_path = resolve(project_root, args.manifest)
    manifest = read_json(manifest_path)
    run_root = resolve(project_root, args.run_root)
    run_manifest = read_json(run_root / "run-manifest.json")
    if run_manifest.get("experiment_manifest_sha256") != sha256_file(manifest_path):
        raise ScoreError("run and experiment manifest digests differ")
    bindings_raw = read_json(resolve(project_root, args.bindings))
    bindings = {
        key: resolve(project_root, value) for key, value in bindings_raw.items()
    }
    sources: list[dict[str, dict[str, Any]]] = []
    for source in manifest["question_sources"]:
        path = resolve(project_root, source["path"])
        if sha256_file(path) != source["sha256"]:
            raise ScoreError(f"question source digest mismatch: {path}")
        payload = read_json(path)
        sources.append({case["id"]: case for case in payload["cases"]})
    return {
        "project_root": project_root,
        "manifest": manifest,
        "manifest_path": manifest_path,
        "run_root": run_root,
        "run_manifest": run_manifest,
        "bindings": bindings,
        "sources": sources,
    }


def safe_source_path(root: Path, relative: str) -> Path:
    candidate = (root / relative).resolve()
    try:
        candidate.relative_to(root.resolve())
    except ValueError as exc:
        raise ScoreError(f"source path escapes corpus root: {relative}") from exc
    return candidate


def line_excerpt(root: Path, evidence: dict[str, Any]) -> dict[str, Any]:
    path_value = evidence.get("path")
    start = evidence.get("start_line")
    end = evidence.get("end_line")
    result = {
        "path": path_value,
        "symbol": evidence.get("symbol"),
        "start_line": start,
        "end_line": end,
    }
    if not isinstance(path_value, str):
        return {**result, "error": "invalid_path"}
    path = safe_source_path(root, path_value)
    if not path.is_file():
        return {**result, "error": "missing_path"}
    try:
        source = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return {**result, "error": "non_utf8_source"}
    lines = source.splitlines()
    if not isinstance(start, int) or not isinstance(end, int):
        return {**result, "error": "missing_line_range"}
    if start < 1 or end < start or end > len(lines):
        return {**result, "error": "invalid_line_range", "file_lines": len(lines)}
    if end - start + 1 > 500:
        return {**result, "error": "citation_range_over_500_lines"}
    excerpt = "\n".join(
        f"{number}: {lines[number - 1]}" for number in range(start, end + 1)
    )
    return {**result, "source_excerpt": excerpt}


def byte_span(root: Path, span: dict[str, Any]) -> dict[str, Any]:
    path = safe_source_path(root, span["path"])
    data = path.read_bytes()
    start = span["start_byte"]
    end = span["end_byte"]
    if start < 0 or end < start or end > len(data):
        raise ScoreError(f"invalid frozen byte range: {span}")
    return {
        "path": span["path"],
        "qualified_symbol": span.get("qualified_symbol"),
        "start_byte": start,
        "end_byte": end,
        "source_excerpt": data[start:end].decode("utf-8"),
    }


def byte_range_to_lines(data: bytes, start: int, end: int) -> tuple[int, int]:
    if start < 0 or end <= start or end > len(data):
        raise ScoreError(f"invalid source byte range: {start}:{end}/{len(data)}")
    return data.count(b"\n", 0, start) + 1, data.count(b"\n", 0, end - 1) + 1


def frozen_line_groups(root: Path, case: dict[str, Any]) -> list[dict[str, Any]]:
    files: dict[str, bytes] = {}
    groups: list[dict[str, Any]] = []
    for group in case["required_groups"]:
        alternatives: list[list[dict[str, Any]]] = []
        for alternative in group["alternatives"]:
            spans: list[dict[str, Any]] = []
            for span in alternative["spans"]:
                path = span["path"]
                if path not in files:
                    files[path] = safe_source_path(root, path).read_bytes()
                start_line, end_line = byte_range_to_lines(
                    files[path], span["start_byte"], span["end_byte"]
                )
                spans.append(
                    {
                        "path": path,
                        "qualified_symbol": span.get("qualified_symbol"),
                        "start_line": start_line,
                        "end_line": end_line,
                    }
                )
            alternatives.append(spans)
        groups.append({"group_id": group["id"], "alternatives": alternatives})
    return groups


def line_range(value: dict[str, Any]) -> tuple[int, int] | None:
    start = value.get("start_line")
    end = value.get("end_line")
    if isinstance(start, int) and isinstance(end, int) and start > 0 and end >= start:
        return start, end
    parent = value.get("parent_range")
    if isinstance(parent, dict):
        start = parent.get("start_line")
        end = parent.get("end_line")
        if isinstance(start, int) and isinstance(end, int) and start > 0 and end >= start:
            return start, end
    return None


def locator_from_hit(hit: dict[str, Any]) -> dict[str, Any] | None:
    lines = line_range(hit)
    path = hit.get("path")
    digest = hit.get("indexed_sha256")
    symbol = hit.get("qualified_symbol")
    if (
        lines is None
        or not isinstance(path, str)
        or not isinstance(digest, str)
        or not isinstance(symbol, str)
    ):
        return None
    sources = hit.get("match_sources", hit.get("lexical_sources", []))
    if not isinstance(sources, list):
        sources = []
    return {
        "chunk_id": hit.get("chunk_id"),
        "path": path,
        "qualified_symbol": symbol,
        "start_line": lines[0],
        "end_line": lines[1],
        "indexed_sha256": digest,
        "match_sources": sorted({item for item in sources if isinstance(item, str)}),
    }


def locator_key(value: dict[str, Any]) -> tuple[Any, ...]:
    return (
        value.get("indexed_sha256"),
        value.get("path"),
        value.get("start_line"),
        value.get("end_line"),
        value.get("qualified_symbol"),
    )


def range_key(value: dict[str, Any]) -> tuple[Any, ...]:
    return (value.get("path"), value.get("start_line"), value.get("end_line"))


def canonical_arguments(value: Any) -> str | None:
    if not isinstance(value, dict):
        return None
    return json.dumps(
        value,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    )


def search_query_key(value: Any) -> tuple[Any, ...] | None:
    if not isinstance(value, dict) or not isinstance(value.get("query"), str):
        return None
    return (value["query"].strip(), value.get("mode", "fts"))


def read_matches_locator(read: dict[str, Any], locator: dict[str, Any]) -> bool:
    if range_key(read) != range_key(locator):
        return False
    expected = read.get("expected_sha256")
    indexed = locator.get("indexed_sha256")
    return not isinstance(expected, str) or expected == indexed


def ranges_overlap(left: dict[str, Any], right: dict[str, Any]) -> bool:
    if left.get("path") != right.get("path"):
        return False
    left_range = line_range(left)
    right_range = line_range(right)
    return bool(
        left_range
        and right_range
        and left_range[0] <= right_range[1]
        and right_range[0] <= left_range[1]
    )


def item_matches_span(item: dict[str, Any], span: dict[str, Any]) -> bool:
    if not ranges_overlap(item, span):
        return False
    wanted_symbol = span.get("qualified_symbol")
    item_symbol = item.get("qualified_symbol")
    return not wanted_symbol or not item_symbol or wanted_symbol == item_symbol


def group_is_covered(items: list[dict[str, Any]], group: dict[str, Any]) -> bool:
    return any(
        all(any(item_matches_span(item, span) for item in items) for span in alternative)
        for alternative in group["alternatives"]
    )


def group_coverage(items: list[dict[str, Any]], groups: list[dict[str, Any]]) -> float:
    return (
        sum(group_is_covered(items, group) for group in groups) / len(groups)
        if groups
        else 0.0
    )


def accepted_item(item: dict[str, Any], groups: list[dict[str, Any]]) -> bool:
    return any(
        item_matches_span(item, span)
        for group in groups
        for alternative in group["alternatives"]
        for span in alternative
    )


def tool_result_parts(item: dict[str, Any]) -> tuple[Any, int, int, int]:
    result = item.get("result")
    if not isinstance(result, dict):
        return None, 0, 0, 0
    structured = result.get("structuredContent", result.get("structured_content"))
    structured_bytes = encoded_json_bytes(structured) if structured is not None else 0
    text_bytes = 0
    decoded_text: Any = None
    content = result.get("content")
    if isinstance(content, list):
        for block in content:
            if not isinstance(block, dict) or not isinstance(block.get("text"), str):
                continue
            text = block["text"]
            text_bytes += len(text.encode("utf-8"))
            if decoded_text is None:
                try:
                    decoded_text = json.loads(text)
                except json.JSONDecodeError:
                    pass
    return (
        structured if structured is not None else decoded_text,
        structured_bytes,
        text_bytes,
        encoded_json_bytes(result),
    )


def source_line_interval(data: bytes, start: int, end: int) -> tuple[int, int] | None:
    if start < 1 or end < start:
        return None
    line = 1
    begin: int | None = None
    offset = 0
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


def union_interval_bytes(intervals: list[tuple[str, int, int]]) -> int:
    total = 0
    by_path: dict[str, list[tuple[int, int]]] = {}
    for path, start, end in intervals:
        by_path.setdefault(path, []).append((start, end))
    for values in by_path.values():
        current_start = current_end = -1
        for start, end in sorted(values):
            if current_start < 0:
                current_start, current_end = start, end
            elif start <= current_end:
                current_end = max(current_end, end)
            else:
                total += current_end - current_start
                current_start, current_end = start, end
        if current_start >= 0:
            total += current_end - current_start
    return total


def ratio_or_none(numerator: int | float, denominator: int | float) -> float | None:
    return numerator / denominator if denominator else None


def blind_id(seed: str, task_id: str, arm: str) -> str:
    framed = f"cidx/assistant-blind/v1\0{seed}\0{task_id}\0{arm}".encode()
    return "blind-" + hashlib.sha256(framed).hexdigest()[:12]


def prepare(context: dict[str, Any]) -> None:
    manifest = context["manifest"]
    run_root = context["run_root"]
    run_manifest = context["run_manifest"]
    sources = context["sources"]
    bindings = context["bindings"]
    grading_root = run_root / "grading"
    if grading_root.exists():
        raise ScoreError(f"grading directory already exists: {grading_root}")
    seed = run_manifest["experiment_manifest_sha256"] + run_manifest["run_id"]
    packets: dict[str, list[dict[str, Any]]] = {}
    key: dict[str, Any] = {
        "schema_version": 1,
        "run_id": run_manifest["run_id"],
        "entries": {},
    }
    journey_records: list[dict[str, Any]] = []
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        corpus_id = task["corpus_id"]
        root = bindings[corpus_id]
        case = sources[task["question_source_index"]][task_id]
        for arm in ARMS:
            answer = read_json(run_root / task_id / arm / "final.json")
            observation = read_json(run_root / task_id / arm / "observation.json")
            identifier = blind_id(seed, task_id, arm)
            journey_records.append(
                {
                    "blind_id": identifier,
                    **deterministic_journey(
                        run_root / task_id / arm / "events.jsonl",
                        run_root / task_id / arm / "final.json",
                        root,
                        case,
                    ),
                }
            )
            key["entries"][identifier] = {
                "task_id": task_id,
                "arm": arm,
                "corpus_id": corpus_id,
            }
            required_groups = []
            for group in case["required_groups"]:
                alternatives = []
                for alternative in group["alternatives"]:
                    alternatives.append(
                        {
                            "spans": [
                                byte_span(root, span)
                                for span in alternative["spans"]
                            ]
                        }
                    )
                required_groups.append(
                    {"group_id": group["id"], "alternatives": alternatives}
                )
            hard_negatives = [
                {
                    "reason": item["reason"],
                    "span": byte_span(root, item["span"]),
                }
                for item in case.get("hard_negatives", [])
            ]
            citations = [
                {"evidence_index": index, **line_excerpt(root, evidence)}
                for index, evidence in enumerate(answer.get("evidence", []))
            ]
            packets.setdefault(corpus_id, []).append(
                {
                    "blind_id": identifier,
                    "task_id": task_id,
                    "question": case["text"],
                    "language": case["language"],
                    "cohorts": case["cohorts"],
                    "assistant_output": answer,
                    "frozen_truth": {
                        "required_groups": required_groups,
                        "hard_negatives": hard_negatives,
                    },
                    "cited_source_excerpts": citations,
                    "operationally_gradable": bool(
                        observation.get("valid_execution")
                        and observation.get("final_error") is None
                    ),
                }
            )
    grading_root.mkdir(parents=True)
    journey_records.sort(key=lambda item: item["blind_id"])
    journey_path = grading_root / "journey-frozen.jsonl"
    with journey_path.open("w", encoding="utf-8") as handle:
        for record in journey_records:
            handle.write(
                json.dumps(record, sort_keys=True, ensure_ascii=False) + "\n"
            )
    write_json(
        grading_root / "journey-freeze.json",
        {
            "schema_version": 2,
            "record_count": len(journey_records),
            "journey_sha256": sha256_file(journey_path),
            "reducer_sha256": sha256_file(Path(__file__).resolve()),
            "arm_identity_present": False,
            "manual_fields_present": False,
        },
    )
    write_json(grading_root / "blind-key.json", key)
    instructions = (
        "Blindly grade every entry in the packet. Do not call tools. You are not "
        "shown arm identity, token use, execution order, or search journey. Use "
        "only the question, frozen required groups, cited excerpts, hard negatives, "
        "and assistant output. A complete answer covers every required group, "
        "answers the question, and has no material unsupported or contradicted "
        "claim. Partial is directionally correct with a material missing group or "
        "unsupported claim. Incorrect has a wrong main mechanism or materially "
        "relies on a hard negative. Ungradable is only for an operationally "
        "ungradable entry. Return exactly one grade for every blind_id and exactly "
        "one required_groups record for every frozen group. evidence_indices are "
        "zero-based indices from assistant_output.evidence. Do not infer missing "
        "evidence. Return only JSON matching the supplied schema."
    )
    for corpus_id, entries in packets.items():
        entries.sort(key=lambda item: hashlib.sha256(item["blind_id"].encode()).hexdigest())
        packet = {
            "schema_version": 1,
            "corpus_id": corpus_id,
            "grading_instructions": instructions,
            "entries": entries,
        }
        write_json(grading_root / f"packet-{corpus_id}.json", packet)
        (grading_root / f"prompt-{corpus_id}.txt").write_text(
            instructions
            + "\n\nGRADING PACKET:\n"
            + json.dumps(packet, ensure_ascii=False, sort_keys=True),
            encoding="utf-8",
        )
    print(f"prepared blind grading packets in {grading_root}")


def events(path: Path) -> list[dict[str, Any]]:
    result = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(event, dict):
            result.append(event)
    return result


def source_paths_from_text(text: str) -> set[str]:
    found: set[str] = set()
    suffix = r"(?:go|ts|tsx)"
    line_pattern = re.compile(rf"(?m)^(?:\./)?([A-Za-z0-9_@+./-]+\.{suffix})(?::\d+)?")
    token_pattern = re.compile(rf"(?<![A-Za-z0-9_])(?:\./)?([A-Za-z0-9_@+./-]+\.{suffix})(?![A-Za-z0-9_])")
    for pattern in (line_pattern, token_pattern):
        for match in pattern.finditer(text):
            value = match.group(1).lstrip("./")
            if not value.startswith(".cidx/"):
                found.add(value)
    return found


def deterministic_journey(
    events_path: Path,
    final_path: Path,
    root: Path,
    case: dict[str, Any],
) -> dict[str, Any]:
    groups = frozen_line_groups(root, case)
    visible_paths: set[str] = set()
    cidx_paths: set[str] = set()
    shell_event_output_bytes = 0
    shell_repository_output_bytes = 0
    cidx_event_result_bytes = 0
    cidx_structured_bytes = 0
    cidx_text_bytes = 0
    search_structured_bytes = 0
    search_text_bytes = 0
    search_event_result_bytes = 0
    read_structured_bytes = 0
    read_text_bytes = 0
    read_event_result_bytes = 0
    cidx_both_representation_calls = 0
    search_both_representation_calls = 0
    read_both_representation_calls = 0
    search_source_bytes = 0
    shell_actions = 0
    search_actions = 0
    read_span_actions = 0
    first_discovery: str | None = None
    action_ordinal = 0
    started_ordinals: dict[str, int] = {}
    search_calls: list[dict[str, Any]] = []
    read_attempts: list[dict[str, Any]] = []
    successful_reads: list[dict[str, Any]] = []
    for event in events(events_path):
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        kind = item.get("type")
        if event.get("type") == "item.started":
            if kind == "mcp_tool_call" and item.get("server") == "cidx":
                tool = item.get("tool")
                if tool in ("search", "read_span"):
                    if isinstance(item.get("id"), str):
                        started_ordinals[item["id"]] = action_ordinal
                    action_ordinal += 1
                    if first_discovery is None:
                        first_discovery = f"cidx_{tool}"
            elif kind == "command_execution":
                command = str(item.get("command", ""))
                if is_repository_inspection(command):
                    if isinstance(item.get("id"), str):
                        started_ordinals[item["id"]] = action_ordinal
                    action_ordinal += 1
                    if first_discovery is None:
                        first_discovery = "shell_repository_inspection"
            continue
        if event.get("type") != "item.completed":
            continue
        if kind == "command_execution":
            command = str(item.get("command", ""))
            output = item.get("aggregated_output")
            if isinstance(output, str):
                shell_event_output_bytes += len(output.encode("utf-8"))
            if is_repository_inspection(command):
                shell_actions += 1
                if isinstance(output, str):
                    shell_repository_output_bytes += len(output.encode("utf-8"))
                    visible_paths.update(source_paths_from_text(output))
                visible_paths.update(source_paths_from_text(command))
        elif kind == "mcp_tool_call" and item.get("server") == "cidx":
            tool = item.get("tool")
            if tool not in ("search", "read_span"):
                continue
            payload, structured_bytes, text_bytes, event_result_bytes = tool_result_parts(item)
            cidx_structured_bytes += structured_bytes
            cidx_text_bytes += text_bytes
            cidx_event_result_bytes += event_result_bytes
            if structured_bytes and text_bytes:
                cidx_both_representation_calls += 1
            if tool == "search":
                search_actions += 1
                search_structured_bytes += structured_bytes
                search_text_bytes += text_bytes
                search_event_result_bytes += event_result_bytes
                if structured_bytes and text_bytes:
                    search_both_representation_calls += 1
                locators: list[dict[str, Any]] = []
                if isinstance(payload, dict) and isinstance(payload.get("results"), list):
                    for hit in payload["results"]:
                        if not isinstance(hit, dict):
                            continue
                        locator = locator_from_hit(hit)
                        if locator is not None:
                            locators.append(locator)
                        body = hit.get("body")
                        if isinstance(body, str):
                            search_source_bytes += len(body.encode("utf-8"))
                search_calls.append(
                    {
                        "ordinal": started_ordinals.get(str(item.get("id")), action_ordinal),
                        "arguments": item.get("arguments"),
                        "locators": locators,
                    }
                )
            else:
                read_span_actions += 1
                read_structured_bytes += structured_bytes
                read_text_bytes += text_bytes
                read_event_result_bytes += event_result_bytes
                if structured_bytes and text_bytes:
                    read_both_representation_calls += 1
                result = item.get("result")
                arguments = item.get("arguments")
                read_attempt = {
                    "ordinal": started_ordinals.get(
                        str(item.get("id")), action_ordinal
                    ),
                    "arguments": arguments,
                    "status": item.get("status"),
                    "result_code": (
                        payload.get("code") if isinstance(payload, dict) else None
                    ),
                }
                if isinstance(arguments, dict):
                    read_attempt.update(arguments)
                read_attempts.append(read_attempt)
                if (
                    isinstance(payload, dict)
                    and isinstance(result, dict)
                    and not result.get("isError", result.get("is_error", False))
                    and isinstance(arguments, dict)
                ):
                    lines = line_range(payload)
                    path = payload.get("path")
                    body = payload.get("body")
                    if lines and isinstance(path, str) and isinstance(body, str):
                        successful_reads.append(
                            {
                                "path": path,
                                "start_line": lines[0],
                                "end_line": lines[1],
                                "body": body,
                                "ordinal": started_ordinals.get(
                                    str(item.get("id")), action_ordinal
                                ),
                            }
                        )
            output = json.dumps(payload, sort_keys=True, ensure_ascii=False) if payload is not None else ""
            paths = source_paths_from_text(output)
            cidx_paths.update(paths)
            visible_paths.update(paths)

    cited_paths: set[str] = set()
    citations: list[dict[str, Any]] = []
    try:
        final_value = read_json(final_path)
    except ScoreError:
        final_value = {}
    for evidence in final_value.get("evidence", []):
        if not isinstance(evidence, dict):
            continue
        path = evidence.get("path")
        if isinstance(path, str) and path.endswith(SOURCE_SUFFIXES):
            normalized_path = path.lstrip("./")
            cited_paths.add(normalized_path)
            lines = line_range(evidence)
            if lines:
                citations.append(
                    {
                        "path": normalized_path,
                        "start_line": lines[0],
                        "end_line": lines[1],
                    }
                )

    search_calls.sort(key=lambda item: item["ordinal"])
    read_attempts.sort(key=lambda item: item["ordinal"])
    locator_occurrences = [
        locator for call in search_calls for locator in call["locators"]
    ]
    unique_locators: list[dict[str, Any]] = []
    seen_locator_keys: set[tuple[Any, ...]] = set()
    for locator in locator_occurrences:
        key = locator_key(locator)
        if key not in seen_locator_keys:
            seen_locator_keys.add(key)
            unique_locators.append(locator)
    first_locators: list[dict[str, Any]] = []
    if search_calls:
        seen_first: set[tuple[Any, ...]] = set()
        for locator in search_calls[0]["locators"]:
            key = locator_key(locator)
            if key not in seen_first:
                seen_first.add(key)
                first_locators.append(locator)

    selected_locators = [
        locator
        for locator in unique_locators
        if any(ranges_overlap(locator, read) for read in successful_reads)
    ]
    cited_locators = [
        locator
        for locator in unique_locators
        if any(ranges_overlap(locator, citation) for citation in citations)
    ]
    selected_keys = {locator_key(item) for item in selected_locators}
    cited_keys = {locator_key(item) for item in cited_locators}
    navigation_false_leads = [
        item
        for item in selected_locators
        if not accepted_item(item, groups)
        and locator_key(item) not in cited_keys
    ]
    first_useful_locator_rank = next(
        (
            rank
            for rank, locator in enumerate(first_locators, start=1)
            if accepted_item(locator, groups)
        ),
        None,
    )

    unique_reads: list[dict[str, Any]] = []
    seen_reads: set[tuple[Any, ...]] = set()
    for read in successful_reads:
        key = range_key(read)
        if key not in seen_reads:
            seen_reads.add(key)
            unique_reads.append(read)
    accepted_reads = [read for read in successful_reads if accepted_item(read, groups)]
    cited_reads = [
        read
        for read in unique_reads
        if any(ranges_overlap(read, citation) for citation in citations)
    ]
    evidence_false_leads = [
        read
        for read in unique_reads
        if not accepted_item(read, groups)
        and not any(ranges_overlap(read, citation) for citation in citations)
    ]
    gross_evidence_bytes = sum(
        len(read["body"].encode("utf-8")) for read in successful_reads
    )
    source_cache: dict[str, bytes] = {}
    delivered_intervals: list[tuple[str, int, int]] = []
    for read in successful_reads:
        path = read["path"]
        try:
            if path not in source_cache:
                source_cache[path] = safe_source_path(root, path).read_bytes()
            interval = source_line_interval(
                source_cache[path], read["start_line"], read["end_line"]
            )
        except (OSError, ScoreError):
            interval = None
        if interval is not None:
            delivered_intervals.append((path, interval[0], interval[1]))
    unique_evidence_bytes = union_interval_bytes(delivered_intervals)
    all_groups_action: int | None = None
    accumulated_reads: list[dict[str, Any]] = []
    for read in sorted(successful_reads, key=lambda item: item["ordinal"]):
        accumulated_reads.append(read)
        if group_coverage(accumulated_reads, groups) == 1.0:
            all_groups_action = read["ordinal"] + 1
            break

    search_argument_keys = [
        key
        for key in (
            canonical_arguments(call.get("arguments")) for call in search_calls
        )
        if key is not None
    ]
    search_query_keys = [
        key
        for key in (search_query_key(call.get("arguments")) for call in search_calls)
        if key is not None
    ]
    read_range_keys = [
        range_key(read)
        for read in read_attempts
        if isinstance(read.get("path"), str)
        and isinstance(read.get("start_line"), int)
        and isinstance(read.get("end_line"), int)
    ]
    exact_locator_reads = []
    for read in read_attempts:
        earlier_locators = [
            locator
            for call in search_calls
            if call["ordinal"] < read["ordinal"]
            for locator in call["locators"]
        ]
        if any(read_matches_locator(read, locator) for locator in earlier_locators):
            exact_locator_reads.append(read)
    initial_reads: list[dict[str, Any]] = []
    seen_read_sources: set[tuple[Any, ...]] = set()
    for read in read_attempts:
        source_key = (read.get("path"), read.get("expected_sha256"))
        if source_key in seen_read_sources:
            continue
        seen_read_sources.add(source_key)
        initial_reads.append(read)
    exact_initial_locator_reads = []
    for read in initial_reads:
        earlier_locators = [
            locator
            for call in search_calls
            if call["ordinal"] < read["ordinal"]
            for locator in call["locators"]
        ]
        if any(read_matches_locator(read, locator) for locator in earlier_locators):
            exact_initial_locator_reads.append(read)
    invalid_range_reads = [
        read
        for read in read_attempts
        if read.get("status") == "failed" and read.get("result_code") == "INVALID_RANGE"
    ]
    repository_action_count = shell_actions + search_actions + read_span_actions
    actions_after_complete = (
        max(0, repository_action_count - all_groups_action)
        if all_groups_action is not None
        else None
    )
    first_search_max_inline_zero = (
        isinstance(search_calls[0].get("arguments"), dict)
        and search_calls[0]["arguments"].get("max_inline_bytes") == 0
        if search_calls
        else False
    )
    repeated_search_queries = len(search_query_keys) - len(set(search_query_keys))
    duplicate_search_arguments = len(search_argument_keys) - len(
        set(search_argument_keys)
    )
    duplicate_read_ranges = len(read_range_keys) - len(set(read_range_keys))
    nonexact_locator_reads = len(read_attempts) - len(exact_locator_reads)
    nonexact_initial_locator_reads = len(initial_reads) - len(
        exact_initial_locator_reads
    )
    stopped_after_complete = (
        actions_after_complete == 0 if actions_after_complete is not None else None
    )
    mechanically_adherent = (
        first_discovery == "cidx_search"
        and first_search_max_inline_zero
        and search_actions <= 2
        and repeated_search_queries == 0
        and duplicate_read_ranges == 0
        and not invalid_range_reads
        and nonexact_initial_locator_reads == 0
    )

    cited_from_cidx = sorted(cited_paths & cidx_paths)
    if read_span_actions and cited_from_cidx:
        usage_class = "read_span_cited"
    elif search_actions and cited_from_cidx:
        usage_class = "navigation"
    elif search_actions or read_span_actions:
        usage_class = "no_cited_path"
    else:
        usage_class = "no_use"
    return {
        "schema_version": 3,
        "first_repository_discovery_action": first_discovery,
        "first_discovery_is_cidx_search": first_discovery == "cidx_search",
        "repository_inspection_action_count": repository_action_count,
        "shell_inspection_action_count": shell_actions,
        "cidx_search_count": search_actions,
        "cidx_read_span_count": read_span_actions,
        "shell_event_output_bytes": shell_event_output_bytes,
        "shell_repository_output_bytes": shell_repository_output_bytes,
        "cidx_structured_bytes": cidx_structured_bytes,
        "cidx_text_bytes": cidx_text_bytes,
        "cidx_event_result_bytes": cidx_event_result_bytes,
        "cidx_both_representation_call_count": cidx_both_representation_calls,
        "repository_event_output_proxy_bytes": (
            shell_repository_output_bytes + cidx_event_result_bytes
        ),
        "visible_source_paths": sorted(visible_paths),
        "final_cited_source_paths": sorted(cited_paths),
        "visible_but_uncited_source_paths": sorted(visible_paths - cited_paths),
        "cidx_returned_source_paths": sorted(cidx_paths),
        "cidx_cited_source_paths": cited_from_cidx,
        "cidx_usage_class": usage_class,
        "orchestration_stage": {
            "first_search_max_inline_zero": first_search_max_inline_zero,
            "search_count_within_two": search_actions <= 2,
            "repeated_search_query_count": repeated_search_queries,
            "duplicate_search_argument_count": duplicate_search_arguments,
            "read_attempt_count": len(read_attempts),
            "duplicate_read_range_count": duplicate_read_ranges,
            "invalid_range_read_count": len(invalid_range_reads),
            "exact_locator_range_read_count": len(exact_locator_reads),
            "nonexact_locator_range_read_count": nonexact_locator_reads,
            "exact_locator_range_read_rate": ratio_or_none(
                len(exact_locator_reads), len(read_attempts)
            ),
            "initial_source_read_count": len(initial_reads),
            "exact_initial_locator_range_read_count": len(
                exact_initial_locator_reads
            ),
            "nonexact_initial_locator_range_read_count": (
                nonexact_initial_locator_reads
            ),
            "exact_initial_locator_range_read_rate": ratio_or_none(
                len(exact_initial_locator_reads), len(initial_reads)
            ),
            "inspection_actions_after_complete_evidence": actions_after_complete,
            "stopped_after_complete_evidence": stopped_after_complete,
            "mechanically_adherent": mechanically_adherent,
            "semantic_refinement_justification_reviewed": False,
        },
        "locator_stage": {
            "search_count": search_actions,
            "locator_occurrence_count": len(locator_occurrences),
            "unique_locator_count": len(unique_locators),
            "first_search_requirement_coverage_at_k": group_coverage(
                first_locators, groups
            ),
            "complete_first_search_locator_hit_at_k": (
                group_coverage(first_locators, groups) == 1.0
            ),
            "any_search_requirement_coverage": group_coverage(
                unique_locators, groups
            ),
            "first_useful_locator_rank": first_useful_locator_rank,
            "duplicate_locator_exposure_rate": ratio_or_none(
                len(locator_occurrences) - len(unique_locators),
                len(locator_occurrences),
            ),
            "locator_selection_utilization": ratio_or_none(
                len(selected_keys), len(unique_locators)
            ),
            "locator_citation_utilization": ratio_or_none(
                len(cited_keys), len(unique_locators)
            ),
            "navigation_false_lead_rate": ratio_or_none(
                len(navigation_false_leads), len(selected_locators)
            ),
            "structured_bytes": search_structured_bytes,
            "text_bytes": search_text_bytes,
            "event_result_bytes": search_event_result_bytes,
            "both_representation_call_count": search_both_representation_calls,
            "source_bytes": search_source_bytes,
            "bytes_per_unique_locator": ratio_or_none(
                search_event_result_bytes, len(unique_locators)
            ),
        },
        "evidence_stage": {
            "read_span_count": read_span_actions,
            "successful_read_span_count": len(successful_reads),
            "unique_read_range_count": len(unique_reads),
            "evidence_requirement_coverage": group_coverage(unique_reads, groups),
            "complete_evidence_hit": group_coverage(unique_reads, groups) == 1.0,
            "read_span_precision": ratio_or_none(
                len(accepted_reads), len(successful_reads)
            ),
            "evidence_citation_utilization": ratio_or_none(
                len(cited_reads), len(unique_reads)
            ),
            "evidence_false_lead_rate": ratio_or_none(
                len(evidence_false_leads), len(unique_reads)
            ),
            "gross_source_bytes": gross_evidence_bytes,
            "unique_source_bytes": unique_evidence_bytes,
            "redundant_evidence_ratio": ratio_or_none(
                max(0, gross_evidence_bytes - unique_evidence_bytes),
                gross_evidence_bytes,
            ),
            "first_inspection_action_with_complete_evidence": all_groups_action,
            "structured_bytes": read_structured_bytes,
            "text_bytes": read_text_bytes,
            "event_result_bytes": read_event_result_bytes,
            "both_representation_call_count": read_both_representation_calls,
        },
    }


def load_frozen_journey(run_root: Path) -> dict[str, dict[str, Any]]:
    path = run_root / "grading" / "journey-frozen.jsonl"
    freeze = read_json(run_root / "grading" / "journey-freeze.json")
    if freeze.get("journey_sha256") != sha256_file(path):
        raise ScoreError("frozen journey digest mismatch")
    result: dict[str, dict[str, Any]] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        value = json.loads(line)
        identifier = value.get("blind_id")
        if not isinstance(identifier, str) or identifier in result:
            raise ScoreError("invalid or duplicate frozen journey blind id")
        result[identifier] = value
    return result


def median(values: list[float | int]) -> float | None:
    return statistics.median(values) if values else None


def ratio(numerator: int | float, denominator: int | float) -> float | None:
    if denominator == 0:
        return None
    return numerator / denominator


def bootstrap_median_interval(values: list[float], seed: str) -> list[float] | None:
    if not values:
        return None
    rng = random.Random(int(hashlib.sha256(seed.encode()).hexdigest(), 16))
    samples = []
    for _ in range(10000):
        sample = [values[rng.randrange(len(values))] for _ in values]
        samples.append(statistics.median(sample))
    samples.sort()
    return [samples[249], samples[9749]]


def paired_group_summary(
    pairs: list[dict[str, Any]], labels: list[str]
) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for label in sorted(set(labels)):
        members = [
            pair
            for pair in pairs
            if label == pair["language"] or label in pair["cohorts"]
        ]
        dual = [
            pair
            for pair in members
            if pair["baseline"]["grade"]["outcome"] == "complete"
            and pair["cidx_fts"]["grade"]["outcome"] == "complete"
        ]
        model_ratios = [
            pair["paired"]["model_total_ratio"]
            for pair in dual
            if pair["paired"]["model_total_ratio"] is not None
        ]
        output_proxy_ratios = [
            ratio(
                pair["cidx_fts"]["journey"][
                    "repository_event_output_proxy_bytes"
                ],
                pair["baseline"]["journey"][
                    "repository_event_output_proxy_bytes"
                ],
            )
            for pair in dual
        ]
        result[label] = {
            "task_count": len(members),
            "baseline_complete": sum(
                pair["baseline"]["grade"]["outcome"] == "complete"
                for pair in members
            ),
            "cidx_complete": sum(
                pair["cidx_fts"]["grade"]["outcome"] == "complete"
                for pair in members
            ),
            "dual_complete": len(dual),
            "model_total_ratio_median": median(model_ratios),
            "model_total_non_increasing": sum(
                value <= 1 for value in model_ratios
            ),
            "inspection_action_difference_median": median(
                [
                    pair["paired"]["repository_inspection_action_difference"]
                    for pair in dual
                ]
            ),
            "event_output_proxy_ratio_median": median(
                [value for value in output_proxy_ratios if value is not None]
            ),
        }
    return result


def mean_present(values: list[float | int | None]) -> float | None:
    present = [float(value) for value in values if value is not None]
    return statistics.mean(present) if present else None


def assistant_stage_summary(pairs: list[dict[str, Any]]) -> dict[str, Any]:
    locator = [pair["cidx_fts"]["journey"]["locator_stage"] for pair in pairs]
    evidence = [pair["cidx_fts"]["journey"]["evidence_stage"] for pair in pairs]
    orchestration = [
        pair["cidx_fts"]["journey"]["orchestration_stage"] for pair in pairs
    ]
    return {
        "orchestration": {
            "task_count": len(orchestration),
            "mechanically_adherent_task_count": sum(
                item["mechanically_adherent"] for item in orchestration
            ),
            "first_search_max_inline_zero_count": sum(
                item["first_search_max_inline_zero"] for item in orchestration
            ),
            "search_count_within_two_count": sum(
                item["search_count_within_two"] for item in orchestration
            ),
            "repeated_search_query_count": sum(
                item["repeated_search_query_count"] for item in orchestration
            ),
            "duplicate_search_argument_count": sum(
                item["duplicate_search_argument_count"] for item in orchestration
            ),
            "duplicate_read_range_count": sum(
                item["duplicate_read_range_count"] for item in orchestration
            ),
            "invalid_range_read_count": sum(
                item["invalid_range_read_count"] for item in orchestration
            ),
            "exact_locator_range_read_count": sum(
                item["exact_locator_range_read_count"] for item in orchestration
            ),
            "nonexact_locator_range_read_count": sum(
                item["nonexact_locator_range_read_count"] for item in orchestration
            ),
            "exact_locator_range_read_rate_macro": mean_present(
                [item["exact_locator_range_read_rate"] for item in orchestration]
            ),
            "initial_source_read_count": sum(
                item["initial_source_read_count"] for item in orchestration
            ),
            "exact_initial_locator_range_read_count": sum(
                item["exact_initial_locator_range_read_count"]
                for item in orchestration
            ),
            "nonexact_initial_locator_range_read_count": sum(
                item["nonexact_initial_locator_range_read_count"]
                for item in orchestration
            ),
            "exact_initial_locator_range_read_rate_macro": mean_present(
                [
                    item["exact_initial_locator_range_read_rate"]
                    for item in orchestration
                ]
            ),
            "stopped_after_complete_evidence_count": sum(
                item["stopped_after_complete_evidence"] is True
                for item in orchestration
            ),
            "inspection_actions_after_complete_evidence": sum(
                item["inspection_actions_after_complete_evidence"] or 0
                for item in orchestration
            ),
        },
        "locator": {
            "task_count": len(locator),
            "search_count": sum(item["search_count"] for item in locator),
            "locator_occurrence_count": sum(
                item["locator_occurrence_count"] for item in locator
            ),
            "unique_locator_count_sum": sum(
                item["unique_locator_count"] for item in locator
            ),
            "complete_first_search_locator_hit_count": sum(
                item["complete_first_search_locator_hit_at_k"] for item in locator
            ),
            "first_search_requirement_coverage_macro": mean_present(
                [item["first_search_requirement_coverage_at_k"] for item in locator]
            ),
            "any_search_requirement_coverage_macro": mean_present(
                [item["any_search_requirement_coverage"] for item in locator]
            ),
            "first_useful_locator_rank_median": median(
                [
                    item["first_useful_locator_rank"]
                    for item in locator
                    if item["first_useful_locator_rank"] is not None
                ]
            ),
            "duplicate_locator_exposure_rate_macro": mean_present(
                [item["duplicate_locator_exposure_rate"] for item in locator]
            ),
            "locator_selection_utilization_macro": mean_present(
                [item["locator_selection_utilization"] for item in locator]
            ),
            "locator_citation_utilization_macro": mean_present(
                [item["locator_citation_utilization"] for item in locator]
            ),
            "navigation_false_lead_rate_macro": mean_present(
                [item["navigation_false_lead_rate"] for item in locator]
            ),
            "structured_bytes": sum(item["structured_bytes"] for item in locator),
            "text_bytes": sum(item["text_bytes"] for item in locator),
            "event_result_bytes": sum(item["event_result_bytes"] for item in locator),
            "both_representation_call_count": sum(
                item["both_representation_call_count"] for item in locator
            ),
            "source_bytes": sum(item["source_bytes"] for item in locator),
        },
        "evidence": {
            "task_count": len(evidence),
            "read_span_count": sum(item["read_span_count"] for item in evidence),
            "successful_read_span_count": sum(
                item["successful_read_span_count"] for item in evidence
            ),
            "complete_evidence_hit_count": sum(
                item["complete_evidence_hit"] for item in evidence
            ),
            "evidence_requirement_coverage_macro": mean_present(
                [item["evidence_requirement_coverage"] for item in evidence]
            ),
            "read_span_precision_macro": mean_present(
                [item["read_span_precision"] for item in evidence]
            ),
            "evidence_citation_utilization_macro": mean_present(
                [item["evidence_citation_utilization"] for item in evidence]
            ),
            "evidence_false_lead_rate_macro": mean_present(
                [item["evidence_false_lead_rate"] for item in evidence]
            ),
            "redundant_evidence_ratio_macro": mean_present(
                [item["redundant_evidence_ratio"] for item in evidence]
            ),
            "gross_source_bytes": sum(item["gross_source_bytes"] for item in evidence),
            "unique_source_bytes": sum(item["unique_source_bytes"] for item in evidence),
            "structured_bytes": sum(item["structured_bytes"] for item in evidence),
            "text_bytes": sum(item["text_bytes"] for item in evidence),
            "event_result_bytes": sum(item["event_result_bytes"] for item in evidence),
            "both_representation_call_count": sum(
                item["both_representation_call_count"] for item in evidence
            ),
        },
    }


def load_grades(context: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any]]:
    grading_root = context["run_root"] / "grading"
    key = read_json(grading_root / "blind-key.json")
    grades: dict[str, Any] = {}
    for corpus in context["manifest"]["corpora"]:
        corpus_id = corpus["corpus_id"]
        payload = read_json(grading_root / f"grades-{corpus_id}.json")
        if payload.get("schema_version") != 1 or payload.get("corpus_id") != corpus_id:
            raise ScoreError(f"wrong grade envelope for {corpus_id}")
        for grade in payload.get("grades", []):
            identifier = grade.get("blind_id")
            if identifier in grades or identifier not in key["entries"]:
                raise ScoreError(f"unknown or duplicate blind id: {identifier}")
            if grade.get("outcome") not in OUTCOMES:
                raise ScoreError(f"invalid outcome: {grade}")
            grades[identifier] = grade
    if set(grades) != set(key["entries"]):
        raise ScoreError("blind grading does not cover the full run")
    return key, grades


def aggregate(context: dict[str, Any]) -> None:
    run_root = context["run_root"]
    manifest = context["manifest"]
    sources = context["sources"]
    key, blind_grades = load_grades(context)
    frozen_journey = load_frozen_journey(run_root)
    if set(frozen_journey) != set(key["entries"]):
        raise ScoreError("frozen journey does not cover the full blind key")
    grade_by_cell: dict[tuple[str, str], dict[str, Any]] = {}
    journey_by_cell: dict[tuple[str, str], dict[str, Any]] = {}
    for identifier, mapping in key["entries"].items():
        grade = blind_grades[identifier]
        case = sources[
            next(
                task["question_source_index"]
                for task in manifest["tasks"]
                if task["task_id"] == mapping["task_id"]
            )
        ][mapping["task_id"]]
        expected_groups = {group["id"] for group in case["required_groups"]}
        actual_groups = {group["group_id"] for group in grade["required_groups"]}
        if actual_groups != expected_groups:
            raise ScoreError(f"grade group mismatch for {identifier}")
        grade_by_cell[(mapping["task_id"], mapping["arm"])] = grade
        journey_by_cell[(mapping["task_id"], mapping["arm"])] = frozen_journey[
            identifier
        ]

    pairs = []
    arm_counts = {arm: {outcome: 0 for outcome in OUTCOMES} for arm in ARMS}
    arm_tokens = {arm: [] for arm in ARMS}
    arm_uncached = {arm: [] for arm in ARMS}
    arm_model_total = {arm: [] for arm in ARMS}
    arm_paths = {arm: [] for arm in ARMS}
    cidx_calls = 0
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        case = sources[task["question_source_index"]][task_id]
        cells: dict[str, Any] = {}
        for arm in ARMS:
            observation = read_json(run_root / task_id / arm / "observation.json")
            usage = observation.get("usage", {})
            grade = grade_by_cell[(task_id, arm)]
            journey = journey_by_cell[(task_id, arm)]
            paths = journey["visible_source_paths"]
            input_tokens = int(usage.get("input_tokens", 0))
            uncached = int(usage.get("uncached_input_tokens", input_tokens - int(usage.get("cached_input_tokens", 0))))
            model_total = int(
                usage.get(
                    "model_total_tokens",
                    input_tokens + int(usage.get("output_tokens", 0)),
                )
            )
            arm_counts[arm][grade["outcome"]] += 1
            arm_tokens[arm].append(input_tokens)
            arm_uncached[arm].append(uncached)
            arm_model_total[arm].append(model_total)
            arm_paths[arm].append(len(paths))
            cidx_calls += int(observation.get("mcp_call_count", 0))
            cells[arm] = {
                "grade": grade,
                "usage": usage,
                "elapsed_seconds": observation.get("elapsed_seconds"),
                "command_count": observation.get("command_count"),
                "mcp_call_count": observation.get("mcp_call_count"),
                "visible_source_paths": paths,
                "journey": journey,
            }
        baseline = cells["baseline"]
        treatment = cells["cidx_fts"]
        base_input = int(baseline["usage"].get("input_tokens", 0))
        cidx_input = int(treatment["usage"].get("input_tokens", 0))
        base_uncached = int(baseline["usage"].get("uncached_input_tokens", 0))
        cidx_uncached = int(treatment["usage"].get("uncached_input_tokens", 0))
        base_model_total = int(baseline["usage"].get("model_total_tokens", 0))
        cidx_model_total = int(treatment["usage"].get("model_total_tokens", 0))
        pairs.append(
            {
                "sequence": task["sequence"],
                "task_id": task_id,
                "corpus_id": task["corpus_id"],
                "language": case["language"],
                "cohorts": case["cohorts"],
                "baseline": baseline,
                "cidx_fts": treatment,
                "paired": {
                    "input_difference": cidx_input - base_input,
                    "input_ratio": ratio(cidx_input, base_input),
                    "uncached_input_difference": cidx_uncached - base_uncached,
                    "uncached_input_ratio": ratio(cidx_uncached, base_uncached),
                    "model_total_difference": cidx_model_total - base_model_total,
                    "model_total_ratio": ratio(cidx_model_total, base_model_total),
                    "visible_source_path_difference": len(treatment["visible_source_paths"]) - len(baseline["visible_source_paths"]),
                    "repository_inspection_action_difference": (
                        treatment["journey"]["repository_inspection_action_count"]
                        - baseline["journey"]["repository_inspection_action_count"]
                    ),
                    "repository_event_output_proxy_bytes_difference": (
                        treatment["journey"][
                            "repository_event_output_proxy_bytes"
                        ]
                        - baseline["journey"][
                            "repository_event_output_proxy_bytes"
                        ]
                    ),
                },
            }
        )

    grading_root = run_root / "grading"
    with (run_root / "paired-results.jsonl").open("w", encoding="utf-8") as handle:
        for pair in pairs:
            handle.write(json.dumps(pair, sort_keys=True, ensure_ascii=False) + "\n")

    dual_complete = [
        pair
        for pair in pairs
        if pair["baseline"]["grade"]["outcome"] == "complete"
        and pair["cidx_fts"]["grade"]["outcome"] == "complete"
    ]
    uncached_ratios = [
        pair["paired"]["uncached_input_ratio"]
        for pair in dual_complete
        if pair["paired"]["uncached_input_ratio"] is not None
    ]
    uncached_differences = [
        pair["paired"]["uncached_input_difference"] for pair in dual_complete
    ]
    model_total_ratios = [
        pair["paired"]["model_total_ratio"]
        for pair in dual_complete
        if pair["paired"]["model_total_ratio"] is not None
    ]
    model_total_differences = [
        pair["paired"]["model_total_difference"] for pair in dual_complete
    ]
    path_ratios = [
        ratio(
            len(pair["cidx_fts"]["visible_source_paths"]),
            len(pair["baseline"]["visible_source_paths"]),
        )
        for pair in dual_complete
    ]
    path_ratios = [value for value in path_ratios if value is not None]
    conversions = sum(
        pair["baseline"]["grade"]["outcome"] != "complete"
        and pair["cidx_fts"]["grade"]["outcome"] == "complete"
        for pair in pairs
    )
    reversals = sum(
        pair["baseline"]["grade"]["outcome"] == "complete"
        and pair["cidx_fts"]["grade"]["outcome"] != "complete"
        for pair in pairs
    )
    interval = bootstrap_median_interval(
        model_total_ratios, context["run_manifest"]["run_id"]
    )
    enough = len(model_total_ratios) >= 8
    non_increasing_required = (
        (2 * len(model_total_ratios) + 2) // 3 if model_total_ratios else 0
    )
    directional = (
        enough
        and median(model_total_ratios) is not None
        and median(model_total_ratios) <= 0.85
        and sum(value <= 1 for value in model_total_ratios)
        >= non_increasing_required
    )
    probes = {}
    for arm in ARMS:
        probes[arm] = read_json(
            run_root / "_schema_probe" / arm / "observation.json"
        )["usage"]
    critical_labels = sorted(
        {
            cohort
            for pair in pairs
            for cohort in pair["cohorts"]
            if cohort.startswith("critical:")
        }
    )
    language_labels = sorted({pair["language"] for pair in pairs})
    aggregate_value = {
        "schema_version": 1,
        "run_id": context["run_manifest"]["run_id"],
        "manifest_sha256": context["run_manifest"]["experiment_manifest_sha256"],
        "identification": (
            "OPTIONAL_CIDX_NOT_ADOPTED" if cidx_calls == 0 else "CIDX_ADOPTED"
        ),
        "task_count": len(pairs),
        "cidx_tool_calls": cidx_calls,
        "cidx_adoption_tasks": sum(
            pair["cidx_fts"]["mcp_call_count"] > 0 for pair in pairs
        ),
        "outcomes": arm_counts,
        "conversions_to_complete": conversions,
        "reversals_from_complete": reversals,
        "dual_complete_count": len(dual_complete),
        "schema_probe": {
            "baseline": probes["baseline"],
            "cidx_fts": probes["cidx_fts"],
            "input_difference": int(probes["cidx_fts"].get("input_tokens", 0)) - int(probes["baseline"].get("input_tokens", 0)),
        },
        "tokens_all_tasks": {
            arm: {
                "input_sum": sum(arm_tokens[arm]),
                "input_median": median(arm_tokens[arm]),
                "uncached_sum": sum(arm_uncached[arm]),
                "uncached_median": median(arm_uncached[arm]),
                "model_total_sum": sum(arm_model_total[arm]),
                "model_total_median": median(arm_model_total[arm]),
            }
            for arm in ARMS
        },
        "dual_complete_efficiency": {
            "model_total_ratio_median": median(model_total_ratios),
            "model_total_difference_median": median(model_total_differences),
            "model_total_ratio_bootstrap_95": interval,
            "model_total_non_increasing_count": sum(
                value <= 1 for value in model_total_ratios
            ),
            "model_total_non_increasing_required": non_increasing_required,
            "uncached_input_ratio_median": median(uncached_ratios),
            "uncached_input_difference_median": median(uncached_differences),
            "non_increasing_count": sum(value <= 1 for value in uncached_ratios),
            "visible_source_path_ratio_median": median(path_ratios),
        },
        "diagnostic_labels": {
            "correctness_safe": arm_counts["cidx_fts"]["complete"] >= arm_counts["baseline"]["complete"] and arm_counts["cidx_fts"]["incorrect"] <= arm_counts["baseline"]["incorrect"],
            "accuracy_helpful": conversions >= 2 and reversals == 0,
            "token_reduction_directional": directional if enough else "INSUFFICIENT_DENOMINATOR",
            "token_reduction_supported": (
                directional and interval is not None and interval[1] < 1
            ) if enough else "INSUFFICIENT_DENOMINATOR",
            "optional_tool_value_observed": cidx_calls > 0,
        },
        "assistant_stages": assistant_stage_summary(pairs),
        "journey_freeze": read_json(grading_root / "journey-freeze.json"),
        "by_critical_cohort": paired_group_summary(pairs, critical_labels),
        "by_language": paired_group_summary(pairs, language_labels),
    }
    write_json(run_root / "aggregate.json", aggregate_value)
    manifest_id = manifest.get("manifest_id", "")
    version_match = re.search(r"v(\d+)$", manifest_id)
    version_number = int(version_match.group(1)) if version_match else 1
    version_label = f"Version {version_number}"
    report_lines = [
        f"# Paired Codex CLI Assistant A/B Result — {version_label}",
        "",
        f"- Run: `{aggregate_value['run_id']}`",
        f"- Identification: `{aggregate_value['identification']}`",
        f"- cidx adoption: `{aggregate_value['cidx_adoption_tasks']}/{len(pairs)}` tasks, `{cidx_calls}` calls",
        "- Scope: diagnostic only; not promotion evidence",
        "",
        "## Outcome summary",
        "",
        "| Arm | Complete | Partial | Incorrect | Ungradable | Model total | Input tokens | Uncached input |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for arm in ARMS:
        report_lines.append(
            f"| {arm} | {arm_counts[arm]['complete']} | {arm_counts[arm]['partial']} | {arm_counts[arm]['incorrect']} | {arm_counts[arm]['ungradable']} | {sum(arm_model_total[arm])} | {sum(arm_tokens[arm])} | {sum(arm_uncached[arm])} |"
        )
    report_lines.extend(
        [
            "",
            "## Paired tasks",
            "",
            "| Task | Baseline | cidx | Model-total ratio | Uncached ratio | cidx calls | Inspection delta | Visible source-path delta |",
            "| --- | --- | --- | ---: | ---: | ---: | ---: | ---: |",
        ]
    )
    for pair in pairs:
        value = pair["paired"]["uncached_input_ratio"]
        report_lines.append(
            f"| {pair['task_id']} | {pair['baseline']['grade']['outcome']} | {pair['cidx_fts']['grade']['outcome']} | {pair['paired']['model_total_ratio']:.3f} | {value:.3f} | {pair['cidx_fts']['mcp_call_count']} | {pair['paired']['repository_inspection_action_difference']} | {pair['paired']['visible_source_path_difference']} |"
        )
    report_lines.extend(
        [
            "",
            "## Critical cohorts",
            "",
            "| Cohort | Tasks | Baseline complete | cidx complete | Median model-total ratio | Non-increasing | Median inspection delta | Median event-output proxy ratio |",
            "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
        ]
    )
    for label, values in aggregate_value["by_critical_cohort"].items():
        report_lines.append(
            f"| {label} | {values['task_count']} | {values['baseline_complete']} | {values['cidx_complete']} | {values['model_total_ratio_median']:.3f} | {values['model_total_non_increasing']}/{values['dual_complete']} | {values['inspection_action_difference_median']:.1f} | {values['event_output_proxy_ratio_median']:.3f} |"
        )
    locator_summary = aggregate_value["assistant_stages"]["locator"]
    evidence_summary = aggregate_value["assistant_stages"]["evidence"]
    orchestration_summary = aggregate_value["assistant_stages"]["orchestration"]
    report_lines.extend(
        [
            "",
            "## Orchestration, locator, and evidence stages",
            "",
            f"- Mechanically prompt-adherent treatment tasks: {orchestration_summary['mechanically_adherent_task_count']}/{orchestration_summary['task_count']}",
            f"- First search with max_inline_bytes=0: {orchestration_summary['first_search_max_inline_zero_count']}/{orchestration_summary['task_count']}",
            f"- At most two searches: {orchestration_summary['search_count_within_two_count']}/{orchestration_summary['task_count']}",
            f"- Repeated search queries / duplicate read ranges / invalid ranges: {orchestration_summary['repeated_search_query_count']} / {orchestration_summary['duplicate_read_range_count']} / {orchestration_summary['invalid_range_read_count']}",
            f"- Exact / non-exact initial locator-range reads: {orchestration_summary['exact_initial_locator_range_read_count']} / {orchestration_summary['nonexact_initial_locator_range_read_count']}",
            f"- Stopped after complete cidx evidence: {orchestration_summary['stopped_after_complete_evidence_count']}/{orchestration_summary['task_count']}",
            f"- First-search complete locator hit: {locator_summary['complete_first_search_locator_hit_count']}/{locator_summary['task_count']}",
            f"- First-search requirement coverage (macro): {locator_summary['first_search_requirement_coverage_macro']:.3f}",
            f"- Any-search requirement coverage (macro): {locator_summary['any_search_requirement_coverage_macro']:.3f}",
            f"- Search payload: {locator_summary['structured_bytes']} structured bytes, {locator_summary['text_bytes']} text bytes, {locator_summary['source_bytes']} source bytes",
            f"- Complete cidx evidence hit: {evidence_summary['complete_evidence_hit_count']}/{evidence_summary['task_count']}",
            f"- Evidence requirement coverage (macro): {evidence_summary['evidence_requirement_coverage_macro']:.3f}",
            f"- Read-span precision (macro): {evidence_summary['read_span_precision_macro']:.3f}",
            f"- Read-span source: {evidence_summary['gross_source_bytes']} gross bytes, {evidence_summary['unique_source_bytes']} unique bytes",
        ]
    )
    interpretation = (
        f"This {version_label} batch estimates the effect of requiring one initial cidx FTS search when the tool is available. It remains a bounded diagnostic on the frozen 12-task panel, not a population estimate or release gate."
        if version_number >= 2
        else "Because the cidx arm made no cidx call, this run identifies the adoption effect of merely exposing the optional tool, not retrieval value."
    )
    report_lines.extend(
        [
            "",
            "## Interpretation boundary",
            "",
            interpretation,
            "",
            f"Frozen journey: `{grading_root / 'journey-frozen.jsonl'}`",
            f"Blind grading key: `{grading_root / 'blind-key.json'}`",
        ]
    )
    (run_root / "report.md").write_text("\n".join(report_lines) + "\n", encoding="utf-8")
    print(f"wrote aggregate and report in {run_root}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("prepare", "aggregate"))
    parser.add_argument("--run-root", required=True)
    parser.add_argument(
        "--manifest", default="testdata/retrieval/assistant-ab-chi-rhf-v3.json"
    )
    parser.add_argument("--bindings", default=".cidx/test/corpora.local.json")
    args = parser.parse_args()
    context = load_context(args)
    if args.command == "prepare":
        prepare(context)
    else:
        aggregate(context)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
