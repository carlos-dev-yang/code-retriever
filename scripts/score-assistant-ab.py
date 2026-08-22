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

import assistant_session_trace as session_trace
import assistant_session_policy_trace as policy_trace


ARMS = ("baseline", "cidx_fts")
OUTCOMES = {"complete", "partial", "incorrect", "ungradable"}
PACKET_SCHEMA_VERSION = 1
GRADE_ENVELOPE_VERSION = 2
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
    if passive_trace_enabled(manifest):
        trace_path = Path(__file__).resolve().with_name("assistant_session_trace.py")
        if run_manifest.get("session_trace_protocol") != session_trace.PASSIVE_TRACE_PROTOCOL or run_manifest.get("session_trace_schema_version") != session_trace.TRACE_SCHEMA_VERSION or run_manifest.get("session_trace_builder_sha256") != sha256_file(trace_path):
            raise ScoreError("passive trace identity does not match the runner manifest")
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
    context = {
        "project_root": project_root,
        "manifest": manifest,
        "manifest_path": manifest_path,
        "run_root": run_root,
        "run_manifest": run_manifest,
        "bindings": bindings,
        "sources": sources,
    }
    if policy_trace_enabled(manifest):
        validate_policy_trace_identity(context)
    return context


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


def passive_trace_enabled(manifest: dict[str, Any]) -> bool:
    return trace_protocol(manifest) == session_trace.PASSIVE_TRACE_PROTOCOL


def policy_trace_enabled(manifest: dict[str, Any]) -> bool:
    return trace_protocol(manifest) == policy_trace.POLICY_TRACE_PROTOCOL


def trace_protocol(manifest: dict[str, Any]) -> str:
    controls = manifest.get("controls")
    configured = (
        controls.get("session_trace_protocol")
        if isinstance(controls, dict)
        else None
    )
    if configured == policy_trace.POLICY_TRACE_PROTOCOL:
        return configured
    try:
        return session_trace.resolve_trace_protocol(manifest)
    except ValueError as exc:
        raise ScoreError(str(exc)) from exc


def policy_arms(manifest: dict[str, Any]) -> tuple[list[str], str, str]:
    """Resolve the policy-v2 pair from its manifest, never legacy constants."""
    configured = manifest.get("arms")
    if isinstance(configured, dict):
        items = []
        for arm_id, arm in configured.items():
            if not isinstance(arm, dict):
                raise ScoreError("policy-v2 arm must be an object")
            if "id" in arm and arm["id"] != arm_id:
                raise ScoreError("policy-v2 arm id does not match its manifest key")
            items.append({"id": arm_id, **arm})
    elif isinstance(configured, list):
        items = configured
    else:
        raise ScoreError("policy-v2 manifest must define two arms")
    if len(items) != 2 or any(not isinstance(item, dict) for item in items):
        raise ScoreError("policy-v2 manifest must define exactly two arm objects")
    arm_ids: list[str] = []
    reference: str | None = None
    treatment: str | None = None
    for arm in items:
        arm_id = arm.get("id")
        suffix = arm.get("prompt_suffix")
        if not isinstance(arm_id, str) or not arm_id or arm_id in arm_ids:
            raise ScoreError("policy-v2 arm IDs must be distinct non-empty strings")
        if arm.get("cidx_exposed") is not True or not isinstance(suffix, str):
            raise ScoreError("policy-v2 arms must expose cidx and declare prompt_suffix")
        arm_ids.append(arm_id)
        if suffix:
            if treatment is not None:
                raise ScoreError("policy-v2 requires exactly one directed prompt arm")
            treatment = arm_id
        else:
            if reference is not None:
                raise ScoreError("policy-v2 requires exactly one neutral prompt arm")
            reference = arm_id
    if reference is None or treatment is None:
        raise ScoreError("policy-v2 arms must resolve to neutral reference and directed treatment")
    if reference != "neutral_cidx" or treatment != "directed_cidx":
        raise ScoreError(
            "policy-v2 arms must be neutral_cidx and directed_cidx"
        )
    return arm_ids, reference, treatment


def validate_policy_trace_identity(context: dict[str, Any]) -> None:
    """Fail closed unless the run binds this exact policy trace implementation."""
    run_manifest = context["run_manifest"]
    trace_path = Path(__file__).resolve().with_name("assistant_session_policy_trace.py")
    delegated_path = Path(__file__).resolve().with_name("assistant_session_trace.py")
    if (
        run_manifest.get("session_trace_protocol")
        != policy_trace.POLICY_TRACE_PROTOCOL
        or run_manifest.get("session_trace_schema_version")
        != policy_trace.TRACE_SCHEMA_VERSION
        or run_manifest.get("session_trace_builder_module") != trace_path.name
        or run_manifest.get("session_trace_builder_sha256") != sha256_file(trace_path)
        or run_manifest.get("session_trace_delegated_builder_module")
        != delegated_path.name
        or run_manifest.get("session_trace_delegated_builder_schema_version")
        != session_trace.TRACE_SCHEMA_VERSION
        or run_manifest.get("session_trace_delegated_builder_sha256")
        != sha256_file(delegated_path)
    ):
        raise ScoreError("policy-v2 trace identity does not match the runner manifest")
    arm_ids, _, _ = policy_arms(context["manifest"])
    if run_manifest.get("arm_ids") != arm_ids:
        raise ScoreError("policy-v2 run arm IDs do not match the experiment manifest")
    snapshots = run_manifest.get("arms")
    if (
        not isinstance(snapshots, list)
        or [item.get("id") for item in snapshots if isinstance(item, dict)] != arm_ids
        or any(not isinstance(item, dict) or item.get("cidx_exposed") is not True for item in snapshots)
    ):
        raise ScoreError("policy-v2 run arm snapshots do not prove equal cidx exposure")


def policy_output_adapter_identity(manifest: dict[str, Any]) -> dict[str, str]:
    """Record the frozen generation adapter without treating it as a grade authority."""
    identity = manifest.get("blind_grade_output_adapter")
    if (
        not isinstance(identity, dict)
        or not isinstance(identity.get("path"), str)
        or not isinstance(identity.get("sha256"), str)
    ):
        raise ScoreError("policy-v2 manifest must record blind_grade_output_adapter identity")
    adapter_path = resolve(Path(__file__).resolve().parent.parent, identity["path"])
    if not adapter_path.is_file() or sha256_file(adapter_path) != identity["sha256"]:
        raise ScoreError("policy-v2 blind grade output adapter digest mismatch")
    return {"path": identity["path"], "sha256": identity["sha256"]}


def prepare(context: dict[str, Any]) -> None:
    if policy_trace_enabled(context["manifest"]):
        prepare_policy(context)
        return
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
            trace_path = run_root / task_id / arm / "session-trace.json"
            if passive_trace_enabled(manifest):
                if not trace_path.exists():
                    raise ScoreError(f"passive run is missing stored trace: {trace_path}")
                trace = read_json(trace_path)
                rebuilt = session_trace.build_session_trace(
                    run_root / task_id / arm / "events.jsonl",
                    run_root / task_id / arm / "final.json", root, frozen_fts_default=True,
                )
                if trace != rebuilt:
                    raise ScoreError(f"stored passive trace differs from deterministic rebuild: {identifier}")
            else:
                journey_records.append(
                    {"blind_id": identifier, **deterministic_journey(
                        run_root / task_id / arm / "events.jsonl",
                        run_root / task_id / arm / "final.json", root, case,
                    )}
                )
                trace = None
            if trace is not None:
                if trace.get("schema_version") != session_trace.TRACE_SCHEMA_VERSION:
                    raise ScoreError(f"unsupported session trace schema for {identifier}")
                journey_records.append({"blind_id": identifier, **trace})
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
            "schema_version": 3 if passive_trace_enabled(manifest) else 2,
            "record_count": len(journey_records),
            "journey_sha256": sha256_file(journey_path),
            "reducer_sha256": sha256_file(Path(__file__).resolve()),
            "arm_identity_present": False,
            "manual_fields_present": False,
            "trace_contains_frozen_truth": False if passive_trace_enabled(manifest) else None,
            "trace_contains_claim_grades": False if passive_trace_enabled(manifest) else None,
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
    if passive_trace_enabled(manifest):
        instructions += (
            f" The GRADING PACKET is input data with PACKET_SCHEMA_VERSION="
            f"{PACKET_SCHEMA_VERSION}: its root schema_version is not the response "
            "format. Return a new grade-result envelope with "
            f"GRADE_ENVELOPE_VERSION={GRADE_ENVELOPE_VERSION}: its root "
            f"schema_version must be the JSON integer {GRADE_ENVELOPE_VERSION}. Do "
            "not copy the packet object or its schema_version into the response."
            " For every material final-answer claim, provide one stable local "
            "claim_id, exact claim_text, observed|derived|unresolved classification, "
            "and evidence-index support references. Observed and derived claims must "
            "have at least one support reference; use unresolved when the cited "
            "evidence does not support the claim. Put claim_id values, not prose, in "
            "unsupported_claims and contradicted_claims; those two lists must be "
            "disjoint. This post-grade annotation remains separate from required-group "
            "coverage."
        )
    for corpus_id, entries in packets.items():
        entries.sort(key=lambda item: hashlib.sha256(item["blind_id"].encode()).hexdigest())
        packet = {
            "schema_version": PACKET_SCHEMA_VERSION,
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


def prepare_policy(context: dict[str, Any]) -> None:
    """Freeze arm-blind packets and independently rebuild every policy-v2 trace."""
    manifest = context["manifest"]
    policy_output_adapter_identity(manifest)
    run_root = context["run_root"]
    run_manifest = context["run_manifest"]
    sources = context["sources"]
    bindings = context["bindings"]
    arm_ids, _, _ = policy_arms(manifest)
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
        for arm in arm_ids:
            final_path = run_root / task_id / arm / "final.json"
            events_path = run_root / task_id / arm / "events.jsonl"
            trace_path = run_root / task_id / arm / "session-trace.json"
            identifier = blind_id(seed, task_id, arm)
            if not trace_path.is_file():
                raise ScoreError(f"policy-v2 run is missing stored trace: {trace_path}")
            trace = read_json(trace_path)
            rebuilt = policy_trace.build_policy_trace(
                events_path, final_path, root, frozen_fts_default=True
            )
            if trace != rebuilt:
                raise ScoreError(
                    f"stored policy-v2 trace differs from deterministic rebuild: {identifier}"
                )
            if (
                trace.get("protocol") != policy_trace.POLICY_TRACE_PROTOCOL
                or trace.get("schema_version") != policy_trace.TRACE_SCHEMA_VERSION
            ):
                raise ScoreError(f"unsupported policy-v2 trace schema for {identifier}")
            answer = read_json(final_path)
            observation = read_json(run_root / task_id / arm / "observation.json")
            journey_records.append({"blind_id": identifier, **trace})
            key["entries"][identifier] = {
                "task_id": task_id,
                "arm": arm,
                "corpus_id": corpus_id,
            }
            required_groups = []
            for group in case["required_groups"]:
                required_groups.append(
                    {
                        "group_id": group["id"],
                        "alternatives": [
                            {"spans": [byte_span(root, span) for span in alternative["spans"]]}
                            for alternative in group["alternatives"]
                        ],
                    }
                )
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
                        "hard_negatives": [
                            {"reason": item["reason"], "span": byte_span(root, item["span"])}
                            for item in case.get("hard_negatives", [])
                        ],
                    },
                    "cited_source_excerpts": [
                        {"evidence_index": index, **line_excerpt(root, evidence)}
                        for index, evidence in enumerate(answer.get("evidence", []))
                    ],
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
            handle.write(json.dumps(record, sort_keys=True, ensure_ascii=False) + "\n")
    write_json(
        grading_root / "journey-freeze.json",
        {
            "schema_version": 4,
            "record_count": len(journey_records),
            "journey_sha256": sha256_file(journey_path),
            "reducer_sha256": sha256_file(Path(__file__).resolve()),
            "trace_protocol": policy_trace.POLICY_TRACE_PROTOCOL,
            "trace_schema_version": policy_trace.TRACE_SCHEMA_VERSION,
            "trace_builder_module": "assistant_session_policy_trace.py",
            "trace_builder_sha256": run_manifest["session_trace_builder_sha256"],
            "trace_delegated_builder_module": run_manifest[
                "session_trace_delegated_builder_module"
            ],
            "trace_delegated_builder_sha256": run_manifest[
                "session_trace_delegated_builder_sha256"
            ],
            "arm_identity_present": False,
            "policy_or_tool_or_order_present": False,
            "manual_fields_present": False,
            "trace_contains_frozen_truth": False,
            "trace_contains_claim_grades": False,
        },
    )
    write_json(grading_root / "blind-key.json", key)
    instructions = (
        "Blindly grade every entry in the packet. Do not call tools. You are not "
        "shown arm identity, prompt policy, token use, tool use, execution order, "
        "or search journey. Use only the question, frozen required groups, cited "
        "excerpts, hard negatives, and assistant output. A complete answer covers "
        "every required group, answers the question, and has no material unsupported "
        "or contradicted claim. Partial is directionally correct with a material "
        "missing group or unsupported claim. Incorrect has a wrong main mechanism "
        "or materially relies on a hard negative. Ungradable is only for an "
        "operationally ungradable entry. Return exactly one grade for every blind_id "
        "and exactly one required_groups record for every frozen group. "
        "evidence_indices are zero-based indices from assistant_output.evidence. "
        "Do not infer missing evidence. Return only JSON matching the supplied schema. "
        f"The packet is input data with PACKET_SCHEMA_VERSION={PACKET_SCHEMA_VERSION}; "
        f"return a grade-result envelope with root schema_version={GRADE_ENVELOPE_VERSION}. "
        "For every material final-answer claim, provide one stable local claim_id, "
        "exact claim_text, observed|derived|unresolved classification, and evidence-index "
        "support references. Observed and derived claims must have at least one support "
        "reference; use unresolved when cited evidence does not support the claim. Put "
        "claim_id values, not prose, in unsupported_claims and contradicted_claims; "
        "those lists must be disjoint."
    )
    for corpus_id, entries in packets.items():
        entries.sort(key=lambda item: hashlib.sha256(item["blind_id"].encode()).hexdigest())
        packet = {
            "schema_version": PACKET_SCHEMA_VERSION,
            "corpus_id": corpus_id,
            "grading_instructions": instructions,
            "entries": entries,
        }
        write_json(grading_root / f"packet-{corpus_id}.json", packet)
        (grading_root / f"prompt-{corpus_id}.txt").write_text(
            instructions + "\n\nGRADING PACKET:\n" + json.dumps(packet, ensure_ascii=False, sort_keys=True),
            encoding="utf-8",
        )
    print(f"prepared policy-v2 arm-blind grading packets in {grading_root}")


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


def trace_read_item(read: dict[str, Any]) -> dict[str, Any] | None:
    requested = read.get("requested")
    delivered = read.get("delivered")
    if not isinstance(requested, dict) or not isinstance(delivered, dict) or not isinstance(delivered.get("path"), str):
        return None
    start, end = requested.get("start_line"), requested.get("end_line")
    if not isinstance(start, int) or not isinstance(end, int) or start < 1 or end < start:
        return None
    return {"path": delivered["path"], "start_line": delivered.get("start_line"), "end_line": delivered.get("end_line"), "expected_sha256": delivered.get("indexed_sha256"), "ordinal": read.get("ordinal")}


def reduce_frozen_trace(
    trace: dict[str, Any], root: Path, case: dict[str, Any], *, require_first_cidx_search: bool
) -> dict[str, Any]:
    """Join frozen mechanics with frozen truth only after blind grades exist."""
    groups = frozen_line_groups(root, case)
    searches = [item for item in trace.get("search_observations", []) if isinstance(item, dict)]
    reads = [item for item in trace.get("read_observations", []) if isinstance(item, dict)]
    actions = [item for item in trace.get("actions", []) if isinstance(item, dict)]
    citations = [item for item in trace.get("final_citations", []) if isinstance(item, dict)]
    searches.sort(key=lambda item: item.get("ordinal", 0))
    reads.sort(key=lambda item: item.get("ordinal", 0))
    actions.sort(key=lambda item: item.get("ordinal", 0))
    locator_occurrences = [locator for search in searches for locator in search.get("locators", []) if isinstance(locator, dict)]
    unique_locators: list[dict[str, Any]] = []
    seen_locator_keys: set[str] = set()
    for locator in locator_occurrences:
        key = locator.get("key")
        if not isinstance(key, str) or key in seen_locator_keys:
            continue
        seen_locator_keys.add(key)
        unique_locators.append(locator)
    first_locators = []
    if searches:
        first_seen: set[str] = set()
        for locator in searches[0].get("locators", []):
            if isinstance(locator, dict) and isinstance(locator.get("key"), str) and locator["key"] not in first_seen:
                first_seen.add(locator["key"])
                first_locators.append(locator)
    read_attempts = [item for item in (trace_read_item(read) for read in reads) if item is not None]
    successful_reads = []
    for raw in reads:
        item = trace_read_item(raw)
        if item is not None and raw.get("success"):
            item["delivered_source_bytes"] = int(raw.get("delivered_source_bytes", 0))
            successful_reads.append(item)
    selected_locators = [locator for locator in unique_locators if any(ranges_overlap(locator, read) for read in successful_reads)]
    cited_locators = [locator for locator in unique_locators if any(ranges_overlap(locator, citation) for citation in citations)]
    selected_keys = {locator.get("key") for locator in selected_locators}
    cited_keys = {locator.get("key") for locator in cited_locators}
    navigation_false_leads = [locator for locator in selected_locators if not accepted_item(locator, groups) and locator.get("key") not in cited_keys]
    unique_reads: list[dict[str, Any]] = []
    seen_read_keys: set[tuple[Any, ...]] = set()
    for read in successful_reads:
        key = range_key(read)
        if key not in seen_read_keys:
            seen_read_keys.add(key)
            unique_reads.append(read)
    accepted_reads = [read for read in successful_reads if accepted_item(read, groups)]
    cited_reads = [read for read in unique_reads if any(ranges_overlap(read, citation) for citation in citations)]
    evidence_false_leads = [read for read in unique_reads if not accepted_item(read, groups) and not any(ranges_overlap(read, citation) for citation in citations)]
    gross_source_bytes = sum(read["delivered_source_bytes"] for read in successful_reads)
    source_cache: dict[str, bytes] = {}
    intervals: list[tuple[str, int, int]] = []
    for read in successful_reads:
        try:
            source_cache.setdefault(read["path"], safe_source_path(root, read["path"]).read_bytes())
            interval = source_line_interval(source_cache[read["path"]], read["start_line"], read["end_line"])
        except (OSError, ScoreError):
            interval = None
        if interval is not None:
            intervals.append((read["path"], interval[0], interval[1]))
    complete_evidence_ordinal: int | None = None
    first_complete_evidence_action: int | None = None
    accumulated: list[dict[str, Any]] = []
    for read in successful_reads:
        accumulated.append(read)
        if group_coverage(accumulated, groups) == 1.0:
            complete_evidence_ordinal = int(read.get("ordinal", 0))
            first_complete_evidence_action = complete_evidence_ordinal + 1
            break
    read_range_keys = [range_key(read) for read in read_attempts]
    search_argument_keys = [canonical_arguments(search.get("query") and {"query": search.get("query"), "mode": search.get("requested_mode"), "k": search.get("k")}) for search in searches]
    search_argument_keys = [key for key in search_argument_keys if key is not None]
    search_query_keys = [(search.get("query", "").strip(), search.get("requested_mode", "fts")) for search in searches if isinstance(search.get("query"), str)]
    first_discovery = actions[0].get("kind") if actions else None
    first_search_policy_met = first_discovery == "cidx_search" if require_first_cidx_search else None
    search_payloads = [item.get("payload_bytes", {}) for item in searches]
    read_payloads = [item.get("payload_bytes", {}) for item in reads]
    cidx_structured = sum(int(value.get("structured", 0)) for value in search_payloads + read_payloads if isinstance(value, dict))
    cidx_text = sum(int(value.get("text", 0)) for value in search_payloads + read_payloads if isinstance(value, dict))
    cidx_result = sum(int(value.get("event_result", 0)) for value in search_payloads + read_payloads if isinstance(value, dict))
    shells = [item for item in trace.get("shell_observations", []) if isinstance(item, dict)]
    ordinary_ranges = [value for shell in shells for value in shell.get("delivered_ranges", []) if isinstance(value, dict)]
    ordinary_discovered_paths = {
        path
        for shell in shells
        for path in shell.get("discovered_source_paths", [])
        if isinstance(path, str)
    }
    locator_paths = {
        locator.get("path")
        for locator in unique_locators
        if isinstance(locator.get("path"), str)
    }
    read_paths = {
        read.get("path")
        for read in successful_reads
        if isinstance(read.get("path"), str)
    }
    visible_paths = sorted(ordinary_discovered_paths | locator_paths | read_paths)
    ordinary_intervals: list[tuple[str, int, int]] = []
    for value in ordinary_ranges:
        try:
            path = value["path"]
            source_cache.setdefault(path, safe_source_path(root, path).read_bytes())
            interval = source_line_interval(source_cache[path], value["start_line"], value["end_line"])
            if interval is not None: ordinary_intervals.append((path, interval[0], interval[1]))
        except (KeyError, OSError, ScoreError):
            pass
    ordinary_unique = union_interval_bytes(ordinary_intervals)
    cidx_unique = union_interval_bytes(intervals)
    combined_unique = union_interval_bytes(intervals + ordinary_intervals)
    ordinary_before = (
        sum(shell.get("ordinal", 0) < complete_evidence_ordinal for shell in shells)
        if complete_evidence_ordinal is not None
        else None
    )
    ordinary_after = (
        sum(shell.get("ordinal", 0) > complete_evidence_ordinal for shell in shells)
        if complete_evidence_ordinal is not None
        else None
    )
    ordinary_output_bytes = sum(int(shell.get("output_bytes", 0)) for shell in shells)
    ordinary_attributed_output_bytes = sum(
        int(shell.get("attributed_output_bytes", 0)) for shell in shells
    )
    ordinary_unattributed_output_bytes = sum(
        int(shell.get("unattributed_output_bytes", 0)) for shell in shells
    )
    range_attribution = (
        "FULL"
        if ordinary_output_bytes > 0 and ordinary_unattributed_output_bytes == 0
        else "PARTIAL"
        if ordinary_attributed_output_bytes > 0
        else "NOT_OBSERVED"
    )
    first_cidx_action_position = next(
        (
            int(action.get("ordinal", 0)) + 1
            for action in actions
            if action.get("kind") in {"cidx_search", "cidx_read_span"}
        ),
        None,
    )
    return {
        "schema_version": 4,
        "first_repository_discovery_action": first_discovery,
        "first_cidx_action_position": first_cidx_action_position,
        "first_discovery_is_cidx_search": first_discovery == "cidx_search",
        "repository_inspection_action_count": len(actions),
        "shell_inspection_action_count": sum(item.get("kind") == "shell_repository_inspection" for item in actions),
        "cidx_search_count": len(searches),
        "cidx_read_span_count": len(reads),
        "cidx_structured_bytes": cidx_structured,
        "cidx_text_bytes": cidx_text,
        "cidx_event_result_bytes": cidx_result,
        "repository_event_output_proxy_bytes": sum(int(shell.get("output_bytes", 0)) for shell in shells) + cidx_result,
        "visible_source_paths": visible_paths,
        "final_cited_source_paths": sorted({citation.get("path") for citation in citations if isinstance(citation.get("path"), str)}),
        "cidx_usage_class": "read_span" if successful_reads else ("navigation" if searches else "no_use"),
        "orchestration_stage": {
            "first_cidx_search_required": require_first_cidx_search,
            "first_cidx_search_policy_met": first_search_policy_met,
            "repeated_search_query_count": len(search_query_keys) - len(set(search_query_keys)),
            "duplicate_search_argument_count": len(search_argument_keys) - len(set(search_argument_keys)),
            "read_attempt_count": len(read_attempts),
            "duplicate_read_range_count": len(read_range_keys) - len(set(read_range_keys)),
            "overlapping_successful_read_count": sum(bool(read.get("overlaps_prior_successful_read_keys")) for read in reads if read.get("success")),
            "effective_fts_search_count": sum(search.get("effective_mode") == "fts" for search in searches),
            "explicit_fts_search_count": sum(
                search.get("effective_mode_authority") == "explicit_request"
                for search in searches
            ),
            "frozen_default_fts_search_count": sum(
                search.get("effective_mode_authority") == "frozen_fts_default_config"
                for search in searches
            ),
            "effective_mode_not_observed_count": sum(search.get("effective_mode") is None for search in searches),
            "read_failure_count": sum(not read.get("success") for read in reads),
            "read_identity_mismatch_count": sum(
                read.get("conformance_error") == "IDENTITY_MISMATCH" for read in reads
            ),
            "read_source_conformance_failure_count": sum(
                not read.get("success")
                and read.get("conformance_error") != "READ_FAILURE"
                for read in reads
            ),
            "total_repository_tool_actions": len(actions),
        },
        "locator_stage": {
            "search_count": len(searches),
            "refined_search_count": max(0, len(searches) - 1),
            "refinement_reason_observability": (
                "NOT_OBSERVED" if len(searches) > 1 else "NOT_APPLICABLE"
            ),
            "locator_occurrence_count": len(locator_occurrences),
            "unique_locator_count": len(unique_locators),
            "first_search_requirement_coverage_at_k": group_coverage(first_locators, groups),
            "complete_first_search_locator_hit_at_k": group_coverage(first_locators, groups) == 1.0,
            "any_search_requirement_coverage": group_coverage(unique_locators, groups),
            "first_useful_locator_rank": next((rank for rank, locator in enumerate(first_locators, start=1) if accepted_item(locator, groups)), None),
            "duplicate_locator_exposure_rate": ratio_or_none(len(locator_occurrences) - len(unique_locators), len(locator_occurrences)),
            "locator_selection_utilization": ratio_or_none(len(selected_keys), len(unique_locators)),
            "locator_citation_utilization": ratio_or_none(len(cited_keys), len(unique_locators)),
            "navigation_false_lead_rate": ratio_or_none(len(navigation_false_leads), len(selected_locators)),
            "structured_bytes": sum(int(value.get("structured", 0)) for value in search_payloads if isinstance(value, dict)),
            "text_bytes": sum(int(value.get("text", 0)) for value in search_payloads if isinstance(value, dict)),
            "event_result_bytes": sum(int(value.get("event_result", 0)) for value in search_payloads if isinstance(value, dict)),
            "source_bytes": 0,
        },
        "evidence_stage": {
            "read_span_count": len(reads),
            "successful_read_span_count": len(successful_reads),
            "unique_read_range_count": len(unique_reads),
            "evidence_requirement_coverage": group_coverage(unique_reads, groups),
            "complete_evidence_hit": group_coverage(unique_reads, groups) == 1.0,
            "read_span_precision": ratio_or_none(len(accepted_reads), len(successful_reads)),
            "evidence_citation_utilization": ratio_or_none(len(cited_reads), len(unique_reads)),
            "evidence_false_lead_rate": ratio_or_none(len(evidence_false_leads), len(unique_reads)),
            "gross_source_bytes": gross_source_bytes,
            "unique_source_bytes": union_interval_bytes(intervals),
            "redundant_evidence_ratio": ratio_or_none(max(0, gross_source_bytes - union_interval_bytes(intervals)), gross_source_bytes),
            "first_inspection_action_with_complete_evidence": first_complete_evidence_action,
            "structured_bytes": sum(int(value.get("structured", 0)) for value in read_payloads if isinstance(value, dict)),
            "text_bytes": sum(int(value.get("text", 0)) for value in read_payloads if isinstance(value, dict)),
            "event_result_bytes": sum(int(value.get("event_result", 0)) for value in read_payloads if isinstance(value, dict)),
        },
        "exploration_stage": {
            "observability": "PARTIAL",
            "ordinary_inspection_action_count": len(shells),
            "ordinary_output_bytes": ordinary_output_bytes,
            "ordinary_attributed_output_bytes": ordinary_attributed_output_bytes,
            "ordinary_unattributed_output_bytes": ordinary_unattributed_output_bytes,
            "ordinary_range_attribution": range_attribution,
            "ordinary_discovered_source_path_count": len(ordinary_discovered_paths),
            "ordinary_attributed_range_count": len(ordinary_ranges),
            "ordinary_unique_source_bytes": ordinary_unique,
            "cidx_unique_source_bytes": cidx_unique,
            "combined_unique_source_bytes": combined_unique,
            "cidx_ordinary_reacquired_bytes": cidx_unique + ordinary_unique - combined_unique,
            "ordinary_inspections_before_complete_cidx_evidence": ordinary_before,
            "ordinary_inspections_after_complete_cidx_evidence": ordinary_after,
            "total_repository_tool_actions": len(actions),
            "named_parent_attribution": "NOT_OBSERVED",
            "named_parent_reason": "ordinary_tool_output_has_no_canonical_parent_identity",
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


def display_number(value: float | int | None, digits: int = 3) -> str:
    return f"{value:.{digits}f}" if value is not None else "NOT_OBSERVED"


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
    pairs: list[dict[str, Any]],
    labels: list[str],
    protocol: str,
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
        if protocol == "legacy-v3":
            output_proxy_ratios = [
                ratio(
                    pair["cidx_fts"]["journey"]["model_visible_repository_output_bytes"],
                    pair["baseline"]["journey"]["model_visible_repository_output_bytes"],
                )
                for pair in dual
            ]
            output_ratio_key = "visible_output_bytes_ratio_median"
        else:
            output_proxy_ratios = [
                ratio(
                    pair["cidx_fts"]["journey"].get("repository_event_output_proxy_bytes", 0),
                    pair["baseline"]["journey"].get("repository_event_output_proxy_bytes", 0),
                )
                for pair in dual
            ]
            output_ratio_key = "event_output_proxy_ratio_median"
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
            output_ratio_key: median(
                [value for value in output_proxy_ratios if value is not None]
            ),
        }
        if protocol == session_trace.PASSIVE_TRACE_PROTOCOL:
            first_positions = [
                pair["cidx_fts"]["journey"].get("first_cidx_action_position")
                for pair in members
                if pair["cidx_fts"]["journey"].get("first_cidx_action_position")
                is not None
            ]
            result[label].update(
                {
                    "cidx_adoption_tasks": sum(
                        pair["cidx_fts"]["mcp_call_count"] > 0 for pair in members
                    ),
                    "cidx_search_adoption_tasks": sum(
                        pair["cidx_fts"]["journey"]["cidx_search_count"] > 0
                        for pair in members
                    ),
                    "first_cidx_action_position_median": median(first_positions),
                }
            )
    return result


def mean_present(values: list[float | int | None]) -> float | None:
    present = [float(value) for value in values if value is not None]
    return statistics.mean(present) if present else None


def exploration_arm_summary(journeys: list[dict[str, Any]]) -> dict[str, Any]:
    stages = [journey.get("exploration_stage", {}) for journey in journeys]
    before = [
        item.get("ordinary_inspections_before_complete_cidx_evidence")
        for item in stages
        if item.get("ordinary_inspections_before_complete_cidx_evidence") is not None
    ]
    after = [
        item.get("ordinary_inspections_after_complete_cidx_evidence")
        for item in stages
        if item.get("ordinary_inspections_after_complete_cidx_evidence") is not None
    ]
    return {
        "task_count": len(stages),
        "ordinary_inspection_action_count": sum(
            int(item.get("ordinary_inspection_action_count", 0)) for item in stages
        ),
        "ordinary_output_bytes": sum(
            int(item.get("ordinary_output_bytes", 0)) for item in stages
        ),
        "ordinary_attributed_output_bytes": sum(
            int(item.get("ordinary_attributed_output_bytes", 0)) for item in stages
        ),
        "ordinary_unattributed_output_bytes": sum(
            int(item.get("ordinary_unattributed_output_bytes", 0)) for item in stages
        ),
        "ordinary_full_range_attribution_task_count": sum(
            item.get("ordinary_range_attribution") == "FULL" for item in stages
        ),
        "ordinary_partial_range_attribution_task_count": sum(
            item.get("ordinary_range_attribution") == "PARTIAL" for item in stages
        ),
        "ordinary_range_not_observed_task_count": sum(
            item.get("ordinary_range_attribution") == "NOT_OBSERVED"
            for item in stages
        ),
        "ordinary_discovered_source_path_count": sum(
            int(item.get("ordinary_discovered_source_path_count", 0))
            for item in stages
        ),
        "ordinary_attributed_range_count": sum(
            int(item.get("ordinary_attributed_range_count", 0)) for item in stages
        ),
        "ordinary_unique_source_bytes": sum(
            int(item.get("ordinary_unique_source_bytes", 0)) for item in stages
        ),
        "cidx_unique_source_bytes": sum(
            int(item.get("cidx_unique_source_bytes", 0)) for item in stages
        ),
        "combined_unique_source_bytes": sum(
            int(item.get("combined_unique_source_bytes", 0)) for item in stages
        ),
        "cidx_ordinary_reacquired_bytes": sum(
            int(item.get("cidx_ordinary_reacquired_bytes", 0)) for item in stages
        ),
        "total_repository_tool_actions": sum(
            int(item.get("total_repository_tool_actions", 0)) for item in stages
        ),
        "complete_cidx_evidence_timing_observed_count": len(before),
        "ordinary_inspections_before_complete_cidx_evidence": sum(before),
        "ordinary_inspections_after_complete_cidx_evidence": sum(after),
        "visible_source_path_count_task_sum": sum(
            len(journey.get("visible_source_paths", [])) for journey in journeys
        ),
        "named_parent_attribution": "NOT_OBSERVED",
    }


def assistant_stage_summary(
    pairs: list[dict[str, Any]],
    protocol: str,
) -> dict[str, Any]:
    journeys = [pair["cidx_fts"]["journey"] for pair in pairs]
    if protocol == "legacy-v4":
        compatible = []
        for pair in pairs:
            clone = dict(pair)
            treatment = dict(pair["cidx_fts"])
            journey = dict(treatment["journey"])
            journey["orchestration_stage"] = {
                "mechanically_adherent": False,
                "first_search_max_inline_zero": False,
                "search_count_within_two": False,
                "repeated_search_query_count": 0,
                "duplicate_search_argument_count": 0,
                "duplicate_read_range_count": 0,
                "invalid_range_read_count": 0,
                "exact_locator_range_read_count": 0,
                "nonexact_locator_range_read_count": 0,
                "exact_locator_range_read_rate": None,
                "initial_source_read_count": 0,
                "exact_initial_locator_range_read_count": 0,
                "nonexact_initial_locator_range_read_count": 0,
                "exact_initial_locator_range_read_rate": None,
                "stopped_after_complete_evidence": None,
                "inspection_actions_after_complete_evidence": None,
            }
            treatment["journey"] = journey
            clone["cidx_fts"] = treatment
            compatible.append(clone)
        result = legacy_assistant_stage_summary(compatible)
        result.pop("orchestration", None)
        return result
    if protocol in {"legacy-v5", "legacy-v6"}:
        return legacy_assistant_stage_summary(pairs)
    if protocol != session_trace.PASSIVE_TRACE_PROTOCOL:
        raise ScoreError(f"stage summary is unavailable for protocol: {protocol}")
    locator = [journey["locator_stage"] for journey in journeys if journey["locator_stage"]["search_count"] > 0]
    evidence = [journey["evidence_stage"] for journey in journeys if journey["evidence_stage"]["read_span_count"] > 0]
    baseline_journeys = [pair["baseline"]["journey"] for pair in pairs]
    orchestration = [journey["orchestration_stage"] for journey in journeys]
    return {
        "orchestration": {
            "treatment_task_count": len(orchestration),
            "first_cidx_search_required": all(item["first_cidx_search_required"] for item in orchestration),
            "first_cidx_search_policy_eligible_task_count": sum(
                item["first_cidx_search_required"] for item in orchestration
            ),
            "first_cidx_search_policy_met_count": sum(item["first_cidx_search_policy_met"] is True for item in orchestration),
            "repeated_search_query_count": sum(
                item["repeated_search_query_count"] for item in orchestration
            ),
            "duplicate_search_argument_count": sum(
                item["duplicate_search_argument_count"] for item in orchestration
            ),
            "duplicate_read_range_count": sum(
                item["duplicate_read_range_count"] for item in orchestration
            ),
            "overlapping_successful_read_count": sum(item["overlapping_successful_read_count"] for item in orchestration),
            "effective_fts_search_count": sum(item["effective_fts_search_count"] for item in orchestration),
            "explicit_fts_search_count": sum(
                item["explicit_fts_search_count"] for item in orchestration
            ),
            "frozen_default_fts_search_count": sum(
                item["frozen_default_fts_search_count"] for item in orchestration
            ),
            "effective_mode_not_observed_count": sum(item["effective_mode_not_observed_count"] for item in orchestration),
            "read_failure_count": sum(item["read_failure_count"] for item in orchestration),
            "read_identity_mismatch_count": sum(
                item["read_identity_mismatch_count"] for item in orchestration
            ),
            "read_source_conformance_failure_count": sum(
                item["read_source_conformance_failure_count"] for item in orchestration
            ),
        },
        "locator": {
            "cidx_search_task_count": len(locator),
            "no_search_treatment_task_count": len(journeys) - len(locator),
            "search_count": sum(item["search_count"] for item in locator),
            "refined_search_count": sum(item["refined_search_count"] for item in locator),
            "refinement_reason_observability": (
                "NOT_OBSERVED"
                if any(item["refined_search_count"] > 0 for item in locator)
                else "NOT_APPLICABLE"
            ),
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
            "source_bytes": sum(item["source_bytes"] for item in locator),
        },
        "evidence": {
            "read_span_task_count": len(evidence),
            "no_read_treatment_task_count": len(journeys) - len(evidence),
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
        },
        "exploration": {
            "baseline": exploration_arm_summary(baseline_journeys),
            "cidx_fts": exploration_arm_summary(journeys),
            "paired": {
                "combined_unique_source_bytes_difference_sum": sum(
                    pair["cidx_fts"]["journey"]["exploration_stage"]["combined_unique_source_bytes"]
                    - pair["baseline"]["journey"]["exploration_stage"]["combined_unique_source_bytes"]
                    for pair in pairs
                ),
                "combined_unique_source_bytes_difference_median": median(
                    [
                        pair["cidx_fts"]["journey"]["exploration_stage"]["combined_unique_source_bytes"]
                        - pair["baseline"]["journey"]["exploration_stage"]["combined_unique_source_bytes"]
                        for pair in pairs
                    ]
                ),
                "repository_tool_action_difference_sum": sum(
                    pair["paired"]["repository_inspection_action_difference"]
                    for pair in pairs
                ),
                "repository_tool_action_difference_median": median(
                    [
                        pair["paired"]["repository_inspection_action_difference"]
                        for pair in pairs
                    ]
                ),
                "visible_source_path_difference_sum": sum(
                    pair["paired"]["visible_source_path_difference"] for pair in pairs
                ),
                "visible_source_path_difference_median": median(
                    [pair["paired"]["visible_source_path_difference"] for pair in pairs]
                ),
                "repository_event_output_proxy_bytes_difference_sum": sum(
                    pair["paired"]["repository_event_output_proxy_bytes_difference"]
                    for pair in pairs
                ),
                "repository_event_output_proxy_bytes_difference_median": median(
                    [
                        pair["paired"]["repository_event_output_proxy_bytes_difference"]
                        for pair in pairs
                    ]
                ),
            },
        },
    }


def legacy_assistant_stage_summary(pairs: list[dict[str, Any]]) -> dict[str, Any]:
    """Frozen V3-V6 presentation, retained byte-for-byte in metric meaning."""
    locator = [pair["cidx_fts"]["journey"]["locator_stage"] for pair in pairs]
    evidence = [pair["cidx_fts"]["journey"]["evidence_stage"] for pair in pairs]
    orchestration = [pair["cidx_fts"]["journey"]["orchestration_stage"] for pair in pairs]
    return {
        "orchestration": {
            "task_count": len(orchestration),
            "mechanically_adherent_task_count": sum(item["mechanically_adherent"] for item in orchestration),
            "first_search_max_inline_zero_count": sum(item["first_search_max_inline_zero"] for item in orchestration),
            "search_count_within_two_count": sum(item["search_count_within_two"] for item in orchestration),
            "repeated_search_query_count": sum(item["repeated_search_query_count"] for item in orchestration),
            "duplicate_search_argument_count": sum(item["duplicate_search_argument_count"] for item in orchestration),
            "duplicate_read_range_count": sum(item["duplicate_read_range_count"] for item in orchestration),
            "invalid_range_read_count": sum(item["invalid_range_read_count"] for item in orchestration),
            "exact_locator_range_read_count": sum(item["exact_locator_range_read_count"] for item in orchestration),
            "nonexact_locator_range_read_count": sum(item["nonexact_locator_range_read_count"] for item in orchestration),
            "exact_locator_range_read_rate_macro": mean_present([item["exact_locator_range_read_rate"] for item in orchestration]),
            "initial_source_read_count": sum(item["initial_source_read_count"] for item in orchestration),
            "exact_initial_locator_range_read_count": sum(item["exact_initial_locator_range_read_count"] for item in orchestration),
            "nonexact_initial_locator_range_read_count": sum(item["nonexact_initial_locator_range_read_count"] for item in orchestration),
            "exact_initial_locator_range_read_rate_macro": mean_present([item["exact_initial_locator_range_read_rate"] for item in orchestration]),
            "stopped_after_complete_evidence_count": sum(item["stopped_after_complete_evidence"] is True for item in orchestration),
            "inspection_actions_after_complete_evidence": sum(item["inspection_actions_after_complete_evidence"] or 0 for item in orchestration),
        },
        "locator": {
            "task_count": len(locator), "search_count": sum(item["search_count"] for item in locator),
            "locator_occurrence_count": sum(item["locator_occurrence_count"] for item in locator),
            "unique_locator_count_sum": sum(item["unique_locator_count"] for item in locator),
            "complete_first_search_locator_hit_count": sum(item["complete_first_search_locator_hit_at_k"] for item in locator),
            "first_search_requirement_coverage_macro": mean_present([item["first_search_requirement_coverage_at_k"] for item in locator]),
            "any_search_requirement_coverage_macro": mean_present([item["any_search_requirement_coverage"] for item in locator]),
            "first_useful_locator_rank_median": median([item["first_useful_locator_rank"] for item in locator if item["first_useful_locator_rank"] is not None]),
            "duplicate_locator_exposure_rate_macro": mean_present([item["duplicate_locator_exposure_rate"] for item in locator]),
            "locator_selection_utilization_macro": mean_present([item["locator_selection_utilization"] for item in locator]),
            "locator_citation_utilization_macro": mean_present([item["locator_citation_utilization"] for item in locator]),
            "navigation_false_lead_rate_macro": mean_present([item["navigation_false_lead_rate"] for item in locator]),
            "structured_bytes": sum(item["structured_bytes"] for item in locator), "text_bytes": sum(item["text_bytes"] for item in locator),
            "event_result_bytes": sum(item["event_result_bytes"] for item in locator),
            "both_representation_call_count": sum(item["both_representation_call_count"] for item in locator), "source_bytes": sum(item["source_bytes"] for item in locator),
        },
        "evidence": {
            "task_count": len(evidence), "read_span_count": sum(item["read_span_count"] for item in evidence),
            "successful_read_span_count": sum(item["successful_read_span_count"] for item in evidence),
            "complete_evidence_hit_count": sum(item["complete_evidence_hit"] for item in evidence),
            "evidence_requirement_coverage_macro": mean_present([item["evidence_requirement_coverage"] for item in evidence]),
            "read_span_precision_macro": mean_present([item["read_span_precision"] for item in evidence]),
            "evidence_citation_utilization_macro": mean_present([item["evidence_citation_utilization"] for item in evidence]),
            "evidence_false_lead_rate_macro": mean_present([item["evidence_false_lead_rate"] for item in evidence]),
            "redundant_evidence_ratio_macro": mean_present([item["redundant_evidence_ratio"] for item in evidence]),
            "gross_source_bytes": sum(item["gross_source_bytes"] for item in evidence), "unique_source_bytes": sum(item["unique_source_bytes"] for item in evidence),
            "structured_bytes": sum(item["structured_bytes"] for item in evidence), "text_bytes": sum(item["text_bytes"] for item in evidence),
            "event_result_bytes": sum(item["event_result_bytes"] for item in evidence),
            "both_representation_call_count": sum(item["both_representation_call_count"] for item in evidence),
        },
    }


def valid_source_evidence_indices(
    root: Path,
    answer_evidence: list[Any],
) -> set[int]:
    """Return only citations that resolve to a valid source excerpt."""
    return {
        index
        for index, evidence in enumerate(answer_evidence)
        if isinstance(evidence, dict)
        and "source_excerpt" in line_excerpt(root, evidence)
    }


def validate_passive_grade(
    context: dict[str, Any],
    key: dict[str, Any],
    identifier: str,
    grade: dict[str, Any],
) -> None:
    expected_grade_keys = {
        "blind_id",
        "outcome",
        "required_groups",
        "unsupported_claims",
        "contradicted_claims",
        "rationale",
        "material_claims",
    }
    if set(grade) != expected_grade_keys:
        raise ScoreError(f"invalid grade fields for {identifier}")

    claims = grade.get("material_claims")
    required_groups = grade.get("required_groups")
    rationale = grade.get("rationale")
    if (
        not isinstance(claims, list)
        or not isinstance(required_groups, list)
        or not isinstance(rationale, str)
        or not rationale
    ):
        raise ScoreError(f"missing material claims for {identifier}")

    mapping = key["entries"][identifier]
    answer_evidence = read_json(
        context["run_root"]
        / mapping["task_id"]
        / mapping["arm"]
        / "final.json"
    ).get("evidence", [])
    if not isinstance(answer_evidence, list):
        raise ScoreError(f"invalid final evidence for {identifier}")
    valid_evidence = valid_source_evidence_indices(
        context["bindings"][mapping["corpus_id"]],
        answer_evidence,
    )

    group_ids: set[str] = set()
    for group in required_groups:
        evidence_indices = (
            group.get("evidence_indices") if isinstance(group, dict) else None
        )
        if (
            not isinstance(group, dict)
            or set(group) != {"group_id", "status", "evidence_indices"}
            or not isinstance(group.get("group_id"), str)
            or not group["group_id"]
            or group["group_id"] in group_ids
            or group.get("status")
            not in {"covered", "missing", "invalid_evidence"}
            or not isinstance(evidence_indices, list)
            or any(
                type(index) is not int
                or index < 0
                or index >= len(answer_evidence)
                or index not in valid_evidence
                for index in evidence_indices
            )
            or len(set(evidence_indices)) != len(evidence_indices)
        ):
            raise ScoreError(f"invalid required-group grade for {identifier}")
        group_ids.add(group["group_id"])

    matching_tasks = [
        task
        for task in context["manifest"]["tasks"]
        if task.get("task_id") == mapping["task_id"]
        and task.get("corpus_id") == mapping["corpus_id"]
    ]
    if len(matching_tasks) != 1:
        raise ScoreError(f"blind entry does not map to one task: {identifier}")
    task = matching_tasks[0]
    case = context["sources"][task["question_source_index"]].get(
        mapping["task_id"]
    )
    if not isinstance(case, dict):
        raise ScoreError(f"blind entry lacks frozen question truth: {identifier}")
    expected_group_ids = {
        group.get("id")
        for group in case.get("required_groups", [])
        if isinstance(group, dict) and isinstance(group.get("id"), str)
    }
    if group_ids != expected_group_ids:
        raise ScoreError(f"grade group mismatch for {identifier}")

    claim_ids: set[str] = set()
    for claim in claims:
        if (
            not isinstance(claim, dict)
            or set(claim)
            != {
                "claim_id",
                "claim_text",
                "classification",
                "support_references",
            }
            or not isinstance(claim.get("claim_id"), str)
            or not claim["claim_id"]
            or claim["claim_id"] in claim_ids
            or not isinstance(claim.get("claim_text"), str)
            or not claim["claim_text"]
            or claim.get("classification")
            not in {"observed", "derived", "unresolved"}
            or not isinstance(claim.get("support_references"), list)
        ):
            raise ScoreError(f"invalid material claim for {identifier}")
        if (
            claim["classification"] in {"observed", "derived"}
            and not claim["support_references"]
        ):
            raise ScoreError(
                f"supported material claim has no reference for {identifier}"
            )
        reference_indices: list[int] = []
        for reference in claim["support_references"]:
            if (
                not isinstance(reference, dict)
                or set(reference) != {"evidence_index"}
                or type(reference["evidence_index"]) is not int
                or reference["evidence_index"] < 0
                or reference["evidence_index"] >= len(answer_evidence)
                or reference["evidence_index"] not in valid_evidence
            ):
                raise ScoreError(
                    f"invalid material claim support reference for {identifier}"
                )
            reference_indices.append(reference["evidence_index"])
        if len(set(reference_indices)) != len(reference_indices):
            raise ScoreError(
                f"duplicate material claim support reference for {identifier}"
            )
        claim_ids.add(claim["claim_id"])

    unsupported = grade.get("unsupported_claims")
    contradicted = grade.get("contradicted_claims")
    if not isinstance(unsupported, list) or not isinstance(contradicted, list):
        raise ScoreError(f"invalid claim finding lists for {identifier}")
    if (
        any(not isinstance(value, str) for value in unsupported + contradicted)
        or len(set(unsupported)) != len(unsupported)
        or len(set(contradicted)) != len(contradicted)
        or not set(unsupported + contradicted).issubset(claim_ids)
        or set(unsupported) & set(contradicted)
    ):
        raise ScoreError(
            f"claim finding does not map to material claims for {identifier}"
        )


def load_grades(context: dict[str, Any]) -> tuple[dict[str, Any], dict[str, Any]]:
    grading_root = context["run_root"] / "grading"
    key = read_json(grading_root / "blind-key.json")
    grades: dict[str, Any] = {}
    for corpus in context["manifest"]["corpora"]:
        corpus_id = corpus["corpus_id"]
        payload = read_json(grading_root / f"grades-{corpus_id}.json")
        strict_v2 = passive_trace_enabled(context["manifest"]) or policy_trace_enabled(
            context["manifest"]
        )
        allowed_versions = {2} if strict_v2 else {1}
        if (
            payload.get("schema_version") not in allowed_versions
            or payload.get("corpus_id") != corpus_id
        ):
            raise ScoreError(f"wrong grade envelope for {corpus_id}")
        for grade in payload.get("grades", []):
            identifier = grade.get("blind_id")
            if identifier in grades or identifier not in key["entries"]:
                raise ScoreError(f"unknown or duplicate blind id: {identifier}")
            if grade.get("outcome") not in OUTCOMES:
                raise ScoreError(f"invalid outcome: {grade}")
            if payload.get("schema_version") == 2:
                validate_passive_grade(context, key, identifier, grade)
            grades[identifier] = grade
    if set(grades) != set(key["entries"]):
        raise ScoreError("blind grading does not cover the full run")
    return key, grades


def policy_truth_shapes(context: dict[str, Any]) -> dict[str, str]:
    """Load only the frozen task-to-question-shape mapping for reporting."""
    spec = context["manifest"].get("truth_source")
    if not isinstance(spec, dict) or not isinstance(spec.get("path"), str):
        raise ScoreError("policy-v2 manifest is missing its frozen truth source")
    path = resolve(context["project_root"], spec["path"])
    if not isinstance(spec.get("sha256"), str) or sha256_file(path) != spec["sha256"]:
        raise ScoreError("policy-v2 truth source digest mismatch")
    payload = read_json(path)
    shapes: dict[str, str] = {}
    for entry in payload.get("entries", []):
        if not isinstance(entry, dict):
            continue
        task_id, shape = entry.get("task_id"), entry.get("primary_shape")
        if not isinstance(task_id, str) or not isinstance(shape, str) or task_id in shapes:
            raise ScoreError("invalid policy-v2 truth shape entry")
        shapes[task_id] = shape
    return shapes


def policy_claim_summary(grades: list[dict[str, Any]]) -> dict[str, Any]:
    claims = [
        claim
        for grade in grades
        for claim in grade.get("material_claims", [])
        if isinstance(claim, dict)
    ]
    unsupported = sum(len(grade.get("unsupported_claims", [])) for grade in grades)
    contradicted = sum(len(grade.get("contradicted_claims", [])) for grade in grades)
    return {
        "material_claim_count": len(claims),
        "observed_count": sum(claim.get("classification") == "observed" for claim in claims),
        "derived_count": sum(claim.get("classification") == "derived" for claim in claims),
        "unresolved_count": sum(claim.get("classification") == "unresolved" for claim in claims),
        "claims_with_support_references": sum(
            bool(claim.get("support_references")) for claim in claims
        ),
        "support_reference_count": sum(
            len(claim.get("support_references", [])) for claim in claims
        ),
        "unsupported_claim_count": unsupported,
        "contradicted_claim_count": contradicted,
        "unsupported_claim_rate": ratio_or_none(unsupported, len(claims)),
    }


def policy_required_group_summary(grades: list[dict[str, Any]]) -> dict[str, Any]:
    groups = [
        group
        for grade in grades
        for group in grade.get("required_groups", [])
        if isinstance(group, dict)
    ]
    covered = sum(group.get("status") == "covered" for group in groups)
    return {
        "group_count": len(groups),
        "covered_count": covered,
        "missing_count": sum(group.get("status") == "missing" for group in groups),
        "invalid_evidence_count": sum(
            group.get("status") == "invalid_evidence" for group in groups
        ),
        "coverage": ratio_or_none(covered, len(groups)),
        "complete_task_count": sum(
            all(group.get("status") == "covered" for group in grade["required_groups"])
            for grade in grades
        ),
    }


def policy_tool_summary(journeys: list[dict[str, Any]]) -> dict[str, Any]:
    stages = [journey["policy_stage"] for journey in journeys]
    actions = [
        action
        for journey in journeys
        for action in journey.get("policy_actions", [])
        if isinstance(action, dict)
    ]
    families = sorted(
        {
            family
            for stage in stages
            for family in stage.get("ordinary_discovery_by_family", {})
        }
    )
    ambiguous_families = sorted(
        {
            family
            for stage in stages
            for family in stage.get("ambiguous_discovery_by_family", {})
        }
    )
    return {
        "task_count": len(journeys),
        "cidx_tool_call_count": sum(
            isinstance(action.get("kind"), str)
            and action["kind"].startswith("cidx_")
            for action in actions
        ),
        "cidx_other_tool_call_count": sum(
            action.get("kind") == "cidx_other" for action in actions
        ),
        "cidx_search_count": sum(int(stage.get("cidx_search_count", 0)) for stage in stages),
        "cidx_search_attempt_count": sum(
            int(stage.get("cidx_search_attempt_count", 0)) for stage in stages
        ),
        "incomplete_cidx_search_attempt_count": sum(
            int(stage.get("incomplete_cidx_search_attempt_count", 0))
            for stage in stages
        ),
        "cidx_read_span_count": sum(int(stage.get("cidx_read_span_count", 0)) for stage in stages),
        "cidx_read_span_attempt_count": sum(
            int(stage.get("cidx_read_span_attempt_count", 0)) for stage in stages
        ),
        "incomplete_cidx_read_span_attempt_count": sum(
            int(stage.get("incomplete_cidx_read_span_attempt_count", 0))
            for stage in stages
        ),
        "cidx_search_task_count": sum(bool(stage.get("cidx_search_count")) for stage in stages),
        "cidx_read_span_task_count": sum(bool(stage.get("cidx_read_span_count")) for stage in stages),
        "selected_locator_read_span_count": sum(
            int(stage.get("selected_locator_read_span_count", 0)) for stage in stages
        ),
        "exact_cidx_read_span_count": sum(
            int(stage.get("exact_cidx_read_span_count", 0)) for stage in stages
        ),
        "overlapping_successful_cidx_read_count": sum(
            int(stage.get("overlapping_successful_cidx_read_count", 0)) for stage in stages
        ),
        "ordinary_discovery_action_count": sum(
            int(stage.get("ordinary_discovery_action_count", 0)) for stage in stages
        ),
        "ordinary_discovery_by_family": {
            family: sum(
                int(stage.get("ordinary_discovery_by_family", {}).get(family, 0))
                for stage in stages
            )
            for family in families
        },
        "ambiguous_discovery_action_count": sum(
            int(stage.get("ambiguous_discovery_action_count", 0)) for stage in stages
        ),
        "ambiguous_discovery_by_family": {
            family: sum(
                int(stage.get("ambiguous_discovery_by_family", {}).get(family, 0))
                for stage in stages
            )
            for family in ambiguous_families
        },
        "shell_cidx_attempt_count": sum(
            int(stage.get("shell_cidx_attempt_count", 0)) for stage in stages
        ),
        "known_file_read_action_count": sum(
            int(stage.get("known_file_read_action_count", 0)) for stage in stages
        ),
        "non_search_repository_action_count": sum(
            int(stage.get("non_search_repository_action_count", 0)) for stage in stages
        ),
        "ordinary_discovery_before_first_cidx_search_count": sum(
            int(stage.get("ordinary_discovery_before_first_cidx_search_count") or 0)
            for stage in stages
        ),
        "ordinary_discovery_after_first_cidx_search_count": sum(
            int(stage.get("ordinary_discovery_after_first_cidx_search_count") or 0)
            for stage in stages
        ),
        "ordinary_discovery_without_cidx_search_count": sum(
            int(stage.get("ordinary_discovery_without_cidx_search_count", 0))
            for stage in stages
        ),
        "ambiguous_discovery_without_cidx_search_count": sum(
            int(stage.get("ambiguous_discovery_without_cidx_search_count", 0))
            for stage in stages
        ),
        "cidx_unique_source_bytes": sum(
            int(stage.get("cidx_unique_source_bytes", 0)) for stage in stages
        ),
        "ordinary_unique_source_bytes": sum(
            int(stage.get("ordinary_unique_source_bytes", 0)) for stage in stages
        ),
        "combined_unique_source_bytes": sum(
            int(stage.get("combined_unique_source_bytes", 0)) for stage in stages
        ),
        "cidx_ordinary_reacquired_source_bytes": sum(
            int(stage.get("cidx_ordinary_reacquired_source_bytes", 0))
            for stage in stages
        ),
        "cidx_ordinary_overlap_source_bytes": sum(
            int(stage.get("cidx_ordinary_overlap_source_bytes", 0))
            for stage in stages
        ),
    }


def policy_slice_summary(
    pairs: list[dict[str, Any]], reference_arm: str, treatment_arm: str
) -> dict[str, Any]:
    reference_grades = [pair["arms"][reference_arm]["grade"] for pair in pairs]
    treatment_grades = [pair["arms"][treatment_arm]["grade"] for pair in pairs]
    dual_complete = [
        pair
        for pair in pairs
        if pair["arms"][reference_arm]["grade"]["outcome"] == "complete"
        and pair["arms"][treatment_arm]["grade"]["outcome"] == "complete"
    ]
    ratios = [
        pair["paired"]["model_total_ratio"]
        for pair in dual_complete
        if pair["paired"]["model_total_ratio"] is not None
    ]
    all_ratios = [
        pair["paired"]["model_total_ratio"]
        for pair in pairs
        if pair["paired"]["model_total_ratio"] is not None
    ]
    return {
        "task_count": len(pairs),
        "outcomes": {
            arm: {
                outcome: sum(pair["arms"][arm]["grade"]["outcome"] == outcome for pair in pairs)
                for outcome in sorted(OUTCOMES)
            }
            for arm in (reference_arm, treatment_arm)
        },
        "required_groups": {
            reference_arm: policy_required_group_summary(reference_grades),
            treatment_arm: policy_required_group_summary(treatment_grades),
        },
        "paired": {
            "conversions_to_complete": sum(
                pair["arms"][reference_arm]["grade"]["outcome"] != "complete"
                and pair["arms"][treatment_arm]["grade"]["outcome"] == "complete"
                for pair in pairs
            ),
            "regressions_from_complete": sum(
                pair["arms"][reference_arm]["grade"]["outcome"] == "complete"
                and pair["arms"][treatment_arm]["grade"]["outcome"] != "complete"
                for pair in pairs
            ),
            "dual_complete_count": len(dual_complete),
            "all_pair_model_total_ratio_median": median(all_ratios),
            "all_pair_model_total_non_increasing_count": sum(
                value <= 1 for value in all_ratios
            ),
            "model_total_ratio_median": median(ratios),
            "model_total_non_increasing_count": sum(value <= 1 for value in ratios),
        },
    }


def aggregate_policy(context: dict[str, Any]) -> None:
    """Aggregate forced-cidx-policy-v2 without importing legacy arm semantics."""
    run_root = context["run_root"]
    manifest = context["manifest"]
    sources = context["sources"]
    arm_ids, reference_arm, treatment_arm = policy_arms(manifest)
    key, blind_grades = load_grades(context)
    frozen_journey = load_frozen_journey(run_root)
    freeze = read_json(run_root / "grading" / "journey-freeze.json")
    if (
        set(frozen_journey) != set(key["entries"])
        or freeze.get("trace_protocol") != policy_trace.POLICY_TRACE_PROTOCOL
        or freeze.get("trace_schema_version") != policy_trace.TRACE_SCHEMA_VERSION
        or freeze.get("trace_builder_sha256")
        != context["run_manifest"].get("session_trace_builder_sha256")
        or freeze.get("trace_delegated_builder_sha256")
        != context["run_manifest"].get("session_trace_delegated_builder_sha256")
    ):
        raise ScoreError("policy-v2 journey freeze does not match the bound trace")
    shapes = policy_truth_shapes(context)
    journeys: dict[tuple[str, str], dict[str, Any]] = {}
    grades: dict[tuple[str, str], dict[str, Any]] = {}
    for identifier, mapping in key["entries"].items():
        task_matches = [task for task in manifest["tasks"] if task["task_id"] == mapping["task_id"]]
        if len(task_matches) != 1 or mapping["arm"] not in arm_ids:
            raise ScoreError(f"policy-v2 blind entry does not map to one scheduled cell: {identifier}")
        task = task_matches[0]
        case = sources[task["question_source_index"]][task["task_id"]]
        grade = blind_grades[identifier]
        if {group["group_id"] for group in grade["required_groups"]} != {
            group["id"] for group in case["required_groups"]
        }:
            raise ScoreError(f"policy-v2 grade group mismatch for {identifier}")
        trace = frozen_journey[identifier]
        if (
            trace.get("protocol") != policy_trace.POLICY_TRACE_PROTOCOL
            or trace.get("schema_version") != policy_trace.TRACE_SCHEMA_VERSION
            or not isinstance(trace.get("policy_stage"), dict)
        ):
            raise ScoreError(f"invalid frozen policy-v2 trace: {identifier}")
        journey = reduce_frozen_trace(trace, context["bindings"][mapping["corpus_id"]], case, require_first_cidx_search=False)
        journey["policy_stage"] = trace["policy_stage"]
        journey["policy_actions"] = trace["actions"]
        journeys[(mapping["task_id"], mapping["arm"])] = journey
        grades[(mapping["task_id"], mapping["arm"])] = grade
    pairs: list[dict[str, Any]] = []
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        case = sources[task["question_source_index"]][task_id]
        if task_id not in shapes:
            raise ScoreError(f"policy-v2 truth lacks question shape for {task_id}")
        cells: dict[str, Any] = {}
        for arm in arm_ids:
            observation = read_json(run_root / task_id / arm / "observation.json")
            usage = observation.get("usage")
            if not isinstance(usage, dict):
                raise ScoreError(f"policy-v2 observation lacks usage: {task_id}/{arm}")
            cells[arm] = {
                "grade": grades[(task_id, arm)],
                "usage": usage,
                "elapsed_seconds": observation.get("elapsed_seconds"),
                "command_count": observation.get("command_count"),
                "mcp_call_count": observation.get("mcp_call_count"),
                "journey": journeys[(task_id, arm)],
            }
        reference = cells[reference_arm]
        treatment = cells[treatment_arm]
        reference_usage, treatment_usage = reference["usage"], treatment["usage"]
        reference_total = int(reference_usage.get("model_total_tokens", 0))
        treatment_total = int(treatment_usage.get("model_total_tokens", 0))
        pairs.append(
            {
                "sequence": task["sequence"],
                "task_id": task_id,
                "corpus_id": task["corpus_id"],
                "language": case["language"],
                "cohorts": case["cohorts"],
                "question_shape": shapes[task_id],
                "reference_arm": reference_arm,
                "treatment_arm": treatment_arm,
                "arms": cells,
                "paired": {
                    "input_difference": int(treatment_usage.get("input_tokens", 0)) - int(reference_usage.get("input_tokens", 0)),
                    "input_ratio": ratio(int(treatment_usage.get("input_tokens", 0)), int(reference_usage.get("input_tokens", 0))),
                    "uncached_input_difference": int(treatment_usage.get("uncached_input_tokens", 0)) - int(reference_usage.get("uncached_input_tokens", 0)),
                    "uncached_input_ratio": ratio(int(treatment_usage.get("uncached_input_tokens", 0)), int(reference_usage.get("uncached_input_tokens", 0))),
                    "model_total_difference": treatment_total - reference_total,
                    "model_total_ratio": ratio(treatment_total, reference_total),
                },
            }
        )
    with (run_root / "paired-results.jsonl").open("w", encoding="utf-8") as handle:
        for pair in pairs:
            handle.write(json.dumps(pair, sort_keys=True, ensure_ascii=False) + "\n")
    arm_grades = {
        arm: [pair["arms"][arm]["grade"] for pair in pairs] for arm in arm_ids
    }
    arm_journeys = {
        arm: [pair["arms"][arm]["journey"] for pair in pairs] for arm in arm_ids
    }
    token_fields = ("input_tokens", "cached_input_tokens", "uncached_input_tokens", "output_tokens", "model_total_tokens")
    policy_summary = policy_slice_summary(pairs, reference_arm, treatment_arm)
    directed_stages = [journey["policy_stage"]["directed_policy"] for journey in arm_journeys[treatment_arm]]
    failure_reasons = sorted(
        {reason for stage in directed_stages for reason in stage.get("failure_reasons", [])}
    )
    aggregate_value = {
        "schema_version": 1,
        "run_id": context["run_manifest"]["run_id"],
        "manifest_sha256": context["run_manifest"]["experiment_manifest_sha256"],
        "trace_protocol": policy_trace.POLICY_TRACE_PROTOCOL,
        "scope": "non_promotion_forced_cidx_prompt_diagnostic",
        "promotion_inference": "NOT_APPLICABLE",
        "arm_ids": arm_ids,
        "reference_arm": reference_arm,
        "treatment_arm": treatment_arm,
        "task_count": len(pairs),
        "answer_quality": {
            "outcomes": policy_summary["outcomes"],
            "required_groups": policy_summary["required_groups"],
            "material_claims": {arm: policy_claim_summary(arm_grades[arm]) for arm in arm_ids},
            "paired": policy_summary["paired"],
        },
        "strict_output_adapter_identity": policy_output_adapter_identity(manifest),
        "grade_semantic_authority": "validate_passive_grade",
        "official_tokens": {
            arm: {
                field: {
                    "sum": sum(int(pair["arms"][arm]["usage"].get(field, 0)) for pair in pairs),
                    "median": median([int(pair["arms"][arm]["usage"].get(field, 0)) for pair in pairs]),
                }
                for field in token_fields
            }
            for arm in arm_ids
        },
        "cidx_and_ordinary_behavior": {arm: policy_tool_summary(arm_journeys[arm]) for arm in arm_ids},
        "directed_policy_compliance": {
            "arm": treatment_arm,
            "task_count": len(directed_stages),
            "fully_compliant_count": sum(bool(stage.get("would_mechanically_comply_if_directed")) for stage in directed_stages),
            "failure_reasons": {
                reason: sum(reason in stage.get("failure_reasons", []) for stage in directed_stages)
                for reason in failure_reasons
            },
        },
        "stage_metrics": {
            arm: {
                "locator": {
                    key: value
                    for key, value in policy_tool_summary(arm_journeys[arm]).items()
                    if key.startswith("cidx_") or key.startswith("selected_locator") or key.startswith("exact_") or key.startswith("overlapping_")
                },
                "evidence": {
                    "complete_evidence_hit_count": sum(bool(journey["evidence_stage"]["complete_evidence_hit"]) for journey in arm_journeys[arm]),
                    "evidence_requirement_coverage_macro": mean_present([journey["evidence_stage"]["evidence_requirement_coverage"] for journey in arm_journeys[arm]]),
                    "read_span_precision_macro": mean_present([journey["evidence_stage"]["read_span_precision"] for journey in arm_journeys[arm]]),
                    "gross_source_bytes": sum(journey["evidence_stage"]["gross_source_bytes"] for journey in arm_journeys[arm]),
                    "unique_source_bytes": sum(journey["evidence_stage"]["unique_source_bytes"] for journey in arm_journeys[arm]),
                },
            }
            for arm in arm_ids
        },
        "by_language": {
            label: policy_slice_summary([pair for pair in pairs if pair["language"] == label], reference_arm, treatment_arm)
            for label in sorted({pair["language"] for pair in pairs})
        },
        "by_question_shape": {
            label: policy_slice_summary([pair for pair in pairs if pair["question_shape"] == label], reference_arm, treatment_arm)
            for label in sorted({pair["question_shape"] for pair in pairs})
        },
        "journey_freeze": freeze,
        "weighted_score": "NOT_REPORTED",
    }
    write_json(run_root / "aggregate.json", aggregate_value)
    report_lines = [
        "# Forced cidx Prompt Diagnostic V1 — Paired Result",
        "",
        f"- Run: `{aggregate_value['run_id']}`",
        f"- Arms: reference `{reference_arm}`, directed treatment `{treatment_arm}`",
        "- Scope: non-promotion prompt-policy diagnostic; no promotion inference.",
        "- Answer quality, policy compliance, navigation, evidence, source volume, and tokens are separate surfaces; no weighted score is reported.",
        "",
        "## Answer quality",
        "",
        "| Arm | Complete | Partial | Incorrect | Ungradable | Required groups covered | Unsupported claims | Contradicted claims |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for arm in arm_ids:
        outcomes = aggregate_value["answer_quality"]["outcomes"][arm]
        groups = aggregate_value["answer_quality"]["required_groups"][arm]
        claims = aggregate_value["answer_quality"]["material_claims"][arm]
        report_lines.append(f"| {arm} | {outcomes['complete']} | {outcomes['partial']} | {outcomes['incorrect']} | {outcomes['ungradable']} | {groups['covered_count']}/{groups['group_count']} | {claims['unsupported_claim_count']} | {claims['contradicted_claim_count']} |")
    compliance = aggregate_value["directed_policy_compliance"]
    report_lines.extend([
        "", "## Directed-policy compliance", "",
        f"- Full mechanical compliance: {compliance['fully_compliant_count']}/{compliance['task_count']}.",
        f"- Failure reasons: {json.dumps(compliance['failure_reasons'], sort_keys=True)}.",
        "", "## Paired outcomes", "",
        f"- Conversions to complete: {policy_summary['paired']['conversions_to_complete']}; regressions from complete: {policy_summary['paired']['regressions_from_complete']}; dual-complete pairs: {policy_summary['paired']['dual_complete_count']}.",
        f"- All-pair model-total ratio median: {display_number(policy_summary['paired']['all_pair_model_total_ratio_median'])}; non-increasing pairs: {policy_summary['paired']['all_pair_model_total_non_increasing_count']}/{len(pairs)}.",
        f"- Dual-complete model-total ratio median: {display_number(policy_summary['paired']['model_total_ratio_median'])}.",
        "", "## Tool and source behavior", "",
    ])
    for arm in arm_ids:
        values = aggregate_value["cidx_and_ordinary_behavior"][arm]
        report_lines.append(f"- `{arm}`: {values['cidx_search_count']} cidx searches, {values['cidx_read_span_count']} reads, {values['selected_locator_read_span_count']} exact selected-locator reads, {values['ordinary_discovery_action_count']} ordinary discovery actions, {values['cidx_ordinary_reacquired_source_bytes']} reacquired source bytes, and {values['combined_unique_source_bytes']} combined unique source bytes.")
    report_lines.extend([
        "", "## Interpretation boundary", "",
        "This measures behavior under an explicit host prompt policy. It preserves noncompliance in every denominator and does not establish voluntary adoption, optional-use marginal value, a mandatory role, or promotion readiness.",
        "", f"Frozen journey: `{run_root / 'grading' / 'journey-frozen.jsonl'}`", f"Blind grading key: `{run_root / 'grading' / 'blind-key.json'}`",
    ])
    (run_root / "report.md").write_text("\n".join(report_lines) + "\n", encoding="utf-8")
    print(f"wrote policy-v2 aggregate and report in {run_root}")


def aggregate(context: dict[str, Any]) -> None:
    if policy_trace_enabled(context["manifest"]):
        aggregate_policy(context)
        return
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
        if passive_trace_enabled(manifest):
            journey_by_cell[(mapping["task_id"], mapping["arm"])] = reduce_frozen_trace(
                frozen_journey[identifier],
                context["bindings"][mapping["corpus_id"]],
                case,
                require_first_cidx_search=context["manifest"].get("controls", {}).get(
                    "require_first_cidx_search", True
                ),
            )
        else:
            journey_by_cell[(mapping["task_id"], mapping["arm"])] = frozen_journey[identifier]

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
                        treatment["journey"].get("repository_event_output_proxy_bytes", 0)
                        - baseline["journey"].get("repository_event_output_proxy_bytes", 0)
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
    claim_support: dict[str, dict[str, Any]] = {}
    for arm in ARMS:
        grades = [pair[arm]["grade"] for pair in pairs]
        claim_rows = [claim for grade in grades for claim in grade.get("material_claims", []) if isinstance(claim, dict)]
        claim_support[arm] = {
            "status": "OBSERVED" if any("material_claims" in grade for grade in grades) else "NOT_OBSERVED",
            "material_claim_count": len(claim_rows),
            "observed_count": sum(claim.get("classification") == "observed" for claim in claim_rows),
            "derived_count": sum(claim.get("classification") == "derived" for claim in claim_rows),
            "unresolved_count": sum(claim.get("classification") == "unresolved" for claim in claim_rows),
            "claims_with_support_references": sum(bool(claim.get("support_references")) for claim in claim_rows),
            "support_reference_count": sum(len(claim.get("support_references", [])) for claim in claim_rows),
            "unsupported_claim_count": sum(len(grade.get("unsupported_claims", [])) for grade in grades),
            "contradicted_claim_count": sum(len(grade.get("contradicted_claims", [])) for grade in grades),
            "unsupported_claim_rate": ratio_or_none(sum(len(grade.get("unsupported_claims", [])) for grade in grades), len(claim_rows)),
            "unsupported_claim_rate_status": "OBSERVED" if claim_rows else "NOT_OBSERVED",
        }
    protocol = trace_protocol(manifest)
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
        "journey_freeze": read_json(grading_root / "journey-freeze.json"),
        "by_critical_cohort": paired_group_summary(pairs, critical_labels, protocol),
        "by_language": paired_group_summary(pairs, language_labels, protocol),
    }
    if protocol != "legacy-v3":
        aggregate_value["assistant_stages"] = assistant_stage_summary(pairs, protocol)
    if protocol == session_trace.PASSIVE_TRACE_PROTOCOL:
        aggregate_value["answer_claim_support"] = claim_support
        no_use_pairs = [
            pair for pair in pairs if pair["cidx_fts"]["mcp_call_count"] == 0
        ]
        aggregate_value["cidx_no_use_outcomes"] = {
            "task_count": len(no_use_pairs),
            **{
                outcome: sum(
                    pair["cidx_fts"]["grade"]["outcome"] == outcome
                    for pair in no_use_pairs
                )
                for outcome in sorted(OUTCOMES)
            },
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
            (
                "| Cohort | Tasks | Baseline complete | cidx complete | Median model-total ratio | Non-increasing | Median inspection delta | Median visible-output ratio |"
                if protocol == "legacy-v3"
                else "| Cohort | Tasks | Baseline complete | cidx complete | Median model-total ratio | Non-increasing | Median inspection delta | Median event-output proxy ratio |"
            ),
            "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
        ]
    )
    for label, values in aggregate_value["by_critical_cohort"].items():
        output_ratio = values[
            "visible_output_bytes_ratio_median"
            if protocol == "legacy-v3"
            else "event_output_proxy_ratio_median"
        ]
        report_lines.append(
            f"| {label} | {values['task_count']} | {values['baseline_complete']} | {values['cidx_complete']} | {display_number(values['model_total_ratio_median'])} | {values['model_total_non_increasing']}/{values['dual_complete']} | {display_number(values['inspection_action_difference_median'], 1)} | {display_number(output_ratio)} |"
        )
    if protocol == session_trace.PASSIVE_TRACE_PROTOCOL:
        report_lines.extend(
            [
                "",
                "## Cidx adoption by critical cohort",
                "",
                "| Cohort | Tasks | Any cidx use | Search use | Median first cidx action |",
                "| --- | ---: | ---: | ---: | ---: |",
            ]
        )
        for label, values in aggregate_value["by_critical_cohort"].items():
            report_lines.append(
                f"| {label} | {values['task_count']} | {values['cidx_adoption_tasks']} | {values['cidx_search_adoption_tasks']} | {display_number(values['first_cidx_action_position_median'], 1)} |"
            )
    stages = aggregate_value.get("assistant_stages")
    locator_summary = stages.get("locator") if isinstance(stages, dict) else None
    evidence_summary = stages.get("evidence") if isinstance(stages, dict) else None
    orchestration_summary = stages.get("orchestration") if isinstance(stages, dict) else None
    if passive_trace_enabled(manifest):
        exploration_summary = aggregate_value["assistant_stages"]["exploration"]
        baseline_exploration = exploration_summary["baseline"]
        cidx_exploration = exploration_summary["cidx_fts"]
        paired_exploration = exploration_summary["paired"]
        policy_eligible = orchestration_summary[
            "first_cidx_search_policy_eligible_task_count"
        ]
        policy_result = (
            f"{orchestration_summary['first_cidx_search_policy_met_count']}/"
            f"{policy_eligible}"
            if policy_eligible
            else "NOT_APPLICABLE"
        )
        report_lines.extend([
            "", "## Stage-separated navigation and evidence", "",
            f"- Availability/adoption: {aggregate_value['cidx_adoption_tasks']}/{len(pairs)} treatment tasks used cidx; {locator_summary['no_search_treatment_task_count']} made no cidx search.",
            f"- No-use outcomes: {aggregate_value['cidx_no_use_outcomes']['complete']} complete, {aggregate_value['cidx_no_use_outcomes']['partial']} partial, {aggregate_value['cidx_no_use_outcomes']['incorrect']} incorrect, and {aggregate_value['cidx_no_use_outcomes']['ungradable']} ungradable across {aggregate_value['cidx_no_use_outcomes']['task_count']} tasks.",
            f"- First-cidx-search policy required: {orchestration_summary['first_cidx_search_required']}; policy result: {policy_result}.",
            f"- Candidate navigation (search users only): {locator_summary['cidx_search_task_count']} tasks, {locator_summary['search_count']} searches, first-search complete locator hit {locator_summary['complete_first_search_locator_hit_count']}/{locator_summary['cidx_search_task_count']}.",
            f"- Refined searches: {locator_summary['refined_search_count']}; reason attribution is {locator_summary['refinement_reason_observability']} because the passive trace does not infer intent.",
            f"- Evidence acquisition (read users only): {evidence_summary['read_span_task_count']} tasks, {evidence_summary['successful_read_span_count']}/{evidence_summary['read_span_count']} successful reads, {evidence_summary['gross_source_bytes']} gross / {evidence_summary['unique_source_bytes']} unique source bytes.",
            f"- Read failures: {orchestration_summary['read_failure_count']}; identity mismatches: {orchestration_summary['read_identity_mismatch_count']}; source-conformance failures: {orchestration_summary['read_source_conformance_failure_count']}.",
            f"- Exploration scope, baseline / cidx: {baseline_exploration['total_repository_tool_actions']} / {cidx_exploration['total_repository_tool_actions']} repository actions and {baseline_exploration['combined_unique_source_bytes']} / {cidx_exploration['combined_unique_source_bytes']} task-summed unique source bytes; paired median differences are {display_number(paired_exploration['repository_tool_action_difference_median'], 1)} actions and {display_number(paired_exploration['combined_unique_source_bytes_difference_median'], 1)} bytes.",
            f"- cidx/ordinary overlap in treatment: {cidx_exploration['cidx_ordinary_reacquired_bytes']} bytes. Named-parent attribution is NOT_OBSERVED for both arms.",
            f"- Ordinary-output attribution, baseline / cidx: {baseline_exploration['ordinary_attributed_output_bytes']}/{baseline_exploration['ordinary_output_bytes']} and {cidx_exploration['ordinary_attributed_output_bytes']}/{cidx_exploration['ordinary_output_bytes']} bytes; unattributed bytes are {baseline_exploration['ordinary_unattributed_output_bytes']} / {cidx_exploration['ordinary_unattributed_output_bytes']}.",
            f"- Post-evidence exploration: complete cidx evidence timing was observed in {cidx_exploration['complete_cidx_evidence_timing_observed_count']} tasks; those tasks had {cidx_exploration['ordinary_inspections_before_complete_cidx_evidence']} ordinary inspections before and {cidx_exploration['ordinary_inspections_after_complete_cidx_evidence']} after that event.",
            f"- Effective FTS mode: {orchestration_summary['effective_fts_search_count']} authoritatively observed ({orchestration_summary['explicit_fts_search_count']} explicit, {orchestration_summary['frozen_default_fts_search_count']} frozen-default); {orchestration_summary['effective_mode_not_observed_count']} not observed.",
            f"- Baseline material claims: {claim_support['baseline']['material_claim_count']}; unsupported / contradicted: {claim_support['baseline']['unsupported_claim_count']} / {claim_support['baseline']['contradicted_claim_count']}; unsupported rate: {display_number(claim_support['baseline']['unsupported_claim_rate'])}.",
            f"- cidx material claims: {claim_support['cidx_fts']['material_claim_count']}; unsupported / contradicted: {claim_support['cidx_fts']['unsupported_claim_count']} / {claim_support['cidx_fts']['contradicted_claim_count']}; unsupported rate: {display_number(claim_support['cidx_fts']['unsupported_claim_rate'])}.",
            "- Answer quality, claim support, navigation, evidence, and exploration remain separate surfaces; no weighted total is reported.",
        ])
        interpretation = "This host-decided availability batch keeps no-use treatment tasks in the primary denominator. Candidate and evidence rates are conditional on the corresponding observed action."
    elif protocol != "legacy-v3":
        stage_heading = (
            "## Locator and evidence stages"
            if protocol == "legacy-v4"
            else "## Orchestration, locator, and evidence stages"
        )
        report_lines.extend([
            "", stage_heading, "",
            *( [f"- Mechanically prompt-adherent treatment tasks: {orchestration_summary['mechanically_adherent_task_count']}/{orchestration_summary['task_count']}", f"- First search with max_inline_bytes=0: {orchestration_summary['first_search_max_inline_zero_count']}/{orchestration_summary['task_count']}", f"- At most two searches: {orchestration_summary['search_count_within_two_count']}/{orchestration_summary['task_count']}"] if orchestration_summary is not None else []),
            *( [f"- Repeated search queries / duplicate read ranges / invalid ranges: {orchestration_summary['repeated_search_query_count']} / {orchestration_summary['duplicate_read_range_count']} / {orchestration_summary['invalid_range_read_count']}", f"- Exact / non-exact initial locator-range reads: {orchestration_summary['exact_initial_locator_range_read_count']} / {orchestration_summary['nonexact_initial_locator_range_read_count']}", f"- Stopped after complete cidx evidence: {orchestration_summary['stopped_after_complete_evidence_count']}/{orchestration_summary['task_count']}"] if orchestration_summary is not None else []),
            f"- First-search complete locator hit: {locator_summary['complete_first_search_locator_hit_count']}/{locator_summary['task_count']}",
            f"- First-search requirement coverage (macro): {locator_summary['first_search_requirement_coverage_macro']:.3f}",
            f"- Any-search requirement coverage (macro): {locator_summary['any_search_requirement_coverage_macro']:.3f}",
            f"- Search payload: {locator_summary['structured_bytes']} structured bytes, {locator_summary['text_bytes']} text bytes, {locator_summary['source_bytes']} source bytes",
            f"- Complete cidx evidence hit: {evidence_summary['complete_evidence_hit_count']}/{evidence_summary['task_count']}",
            f"- Evidence requirement coverage (macro): {evidence_summary['evidence_requirement_coverage_macro']:.3f}",
            f"- Read-span precision (macro): {evidence_summary['read_span_precision_macro']:.3f}",
            f"- Read-span source: {evidence_summary['gross_source_bytes']} gross bytes, {evidence_summary['unique_source_bytes']} unique bytes",
        ])
        interpretation = (f"This {version_label} batch estimates the effect of requiring one initial cidx FTS search when the tool is available. It remains a bounded diagnostic on the frozen 12-task panel, not a population estimate or release gate." if version_number >= 2 else "Because the cidx arm made no cidx call, this run identifies the adoption effect of merely exposing the optional tool, not retrieval value.")
    else:
        interpretation = "This Version 3 batch estimates the effect of requiring one initial cidx FTS search when the tool is available. It remains a bounded diagnostic on the frozen 12-task panel, not a population estimate or release gate."
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
