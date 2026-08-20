#!/usr/bin/env python3
"""Prepare blind grading packets and aggregate a frozen assistant A/B run."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import random
import re
import statistics
from typing import Any


ARMS = ("baseline", "cidx_fts")
OUTCOMES = {"complete", "partial", "incorrect", "ungradable"}
SOURCE_SUFFIXES = (".go", ".ts", ".tsx")
REPOSITORY_INSPECTION_RE = re.compile(
    r"(?i)(?:^|[;&|()\s])(?:rg|grep|find|fd|ls|tree|sed|cat|head|tail|awk|nl)(?:\s|$)|git\s+grep"
)


class ScoreError(RuntimeError):
    pass


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
            "schema_version": 1,
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


def mcp_visible_output(item: dict[str, Any]) -> str:
    hidden = {
        "id",
        "type",
        "server",
        "tool",
        "status",
        "arguments",
        "duration_ms",
    }
    visible = {key: value for key, value in item.items() if key not in hidden}
    return json.dumps(visible, sort_keys=True, ensure_ascii=False) if visible else ""


def deterministic_journey(events_path: Path, final_path: Path) -> dict[str, Any]:
    visible_paths: set[str] = set()
    cidx_paths: set[str] = set()
    shell_output_bytes = 0
    cidx_output_bytes = 0
    shell_actions = 0
    search_actions = 0
    read_span_actions = 0
    first_discovery: str | None = None
    for event in events(events_path):
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        kind = item.get("type")
        if event.get("type") == "item.started":
            if kind == "mcp_tool_call" and item.get("server") == "cidx":
                tool = item.get("tool")
                if tool in ("search", "read_span") and first_discovery is None:
                    first_discovery = f"cidx_{tool}"
            elif kind == "command_execution":
                command = str(item.get("command", ""))
                if REPOSITORY_INSPECTION_RE.search(command) and first_discovery is None:
                    first_discovery = "shell_repository_inspection"
            continue
        if event.get("type") != "item.completed":
            continue
        if kind == "command_execution":
            command = str(item.get("command", ""))
            output = item.get("aggregated_output")
            if REPOSITORY_INSPECTION_RE.search(command):
                shell_actions += 1
                if isinstance(output, str):
                    shell_output_bytes += len(output.encode("utf-8"))
                    visible_paths.update(source_paths_from_text(output))
                visible_paths.update(source_paths_from_text(command))
        elif kind == "mcp_tool_call" and item.get("server") == "cidx":
            tool = item.get("tool")
            if tool not in ("search", "read_span"):
                continue
            if tool == "search":
                search_actions += 1
            else:
                read_span_actions += 1
            output = mcp_visible_output(item)
            cidx_output_bytes += len(output.encode("utf-8"))
            paths = source_paths_from_text(output)
            cidx_paths.update(paths)
            visible_paths.update(paths)

    cited_paths: set[str] = set()
    try:
        final_value = read_json(final_path)
    except ScoreError:
        final_value = {}
    for evidence in final_value.get("evidence", []):
        if not isinstance(evidence, dict):
            continue
        path = evidence.get("path")
        if isinstance(path, str) and path.endswith(SOURCE_SUFFIXES):
            cited_paths.add(path.lstrip("./"))
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
        "schema_version": 1,
        "first_repository_discovery_action": first_discovery,
        "first_discovery_is_cidx_search": first_discovery == "cidx_search",
        "repository_inspection_action_count": (
            shell_actions + search_actions + read_span_actions
        ),
        "shell_inspection_action_count": shell_actions,
        "cidx_search_count": search_actions,
        "cidx_read_span_count": read_span_actions,
        "shell_visible_output_bytes": shell_output_bytes,
        "cidx_visible_output_bytes": cidx_output_bytes,
        "model_visible_repository_output_bytes": (
            shell_output_bytes + cidx_output_bytes
        ),
        "visible_source_paths": sorted(visible_paths),
        "final_cited_source_paths": sorted(cited_paths),
        "visible_but_uncited_source_paths": sorted(visible_paths - cited_paths),
        "cidx_returned_source_paths": sorted(cidx_paths),
        "cidx_cited_source_paths": cited_from_cidx,
        "cidx_usage_class": usage_class,
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
        output_ratios = [
            ratio(
                pair["cidx_fts"]["journey"][
                    "model_visible_repository_output_bytes"
                ],
                pair["baseline"]["journey"][
                    "model_visible_repository_output_bytes"
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
            "visible_output_bytes_ratio_median": median(
                [value for value in output_ratios if value is not None]
            ),
        }
    return result


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
                    "model_visible_repository_output_bytes_difference": (
                        treatment["journey"][
                            "model_visible_repository_output_bytes"
                        ]
                        - baseline["journey"][
                            "model_visible_repository_output_bytes"
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
        "journey_freeze": read_json(grading_root / "journey-freeze.json"),
        "by_critical_cohort": paired_group_summary(pairs, critical_labels),
        "by_language": paired_group_summary(pairs, language_labels),
    }
    write_json(run_root / "aggregate.json", aggregate_value)
    manifest_id = manifest.get("manifest_id", "")
    if manifest_id.endswith("v3"):
        version_label = "Version 3"
    elif manifest_id.endswith("v2"):
        version_label = "Version 2"
    else:
        version_label = "Version 1"
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
            "| Cohort | Tasks | Baseline complete | cidx complete | Median model-total ratio | Non-increasing | Median inspection delta | Median visible-output ratio |",
            "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
        ]
    )
    for label, values in aggregate_value["by_critical_cohort"].items():
        report_lines.append(
            f"| {label} | {values['task_count']} | {values['baseline_complete']} | {values['cidx_complete']} | {values['model_total_ratio_median']:.3f} | {values['model_total_non_increasing']}/{values['dual_complete']} | {values['inspection_action_difference_median']:.1f} | {values['visible_output_bytes_ratio_median']:.3f} |"
        )
    interpretation = (
        f"This {version_label} batch estimates the effect of requiring one initial cidx FTS search when the tool is available. It remains a bounded diagnostic on the frozen 12-task panel, not a population estimate or release gate."
        if version_label in ("Version 2", "Version 3")
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
