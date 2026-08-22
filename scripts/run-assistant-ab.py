#!/usr/bin/env python3
"""Run the frozen paired Codex CLI assistant diagnostic.

The runner owns execution and raw observation capture only. It never changes
retrieval settings, question truth, source files, or grading decisions.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Any

from assistant_session_trace import (
    PASSIVE_TRACE_PROTOCOL,
    TRACE_SCHEMA_VERSION,
    build_session_trace,
    resolve_trace_protocol,
)

try:
    from assistant_session_policy_trace import (
        POLICY_TRACE_PROTOCOL,
        TRACE_SCHEMA_VERSION as POLICY_TRACE_SCHEMA_VERSION,
        build_policy_trace as build_policy_session_trace,
        is_shell_cidx_attempt,
    )
except ModuleNotFoundError as exc:
    if exc.name != "assistant_session_policy_trace":
        raise
    POLICY_TRACE_PROTOCOL = "forced-cidx-policy-v2"
    POLICY_TRACE_SCHEMA_VERSION = None
    build_policy_session_trace = None
    is_shell_cidx_attempt = None


BASELINE_ARM = "baseline"
CIDX_ARM = "cidx_fts"
POLICY_NEUTRAL_ARM = "neutral_cidx"
POLICY_DIRECTED_ARM = "directed_cidx"
POLICY_ARM_IDS = (POLICY_NEUTRAL_ARM, POLICY_DIRECTED_ARM)
ARM_ID_PATTERN = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]*")
SHA256_PATTERN = re.compile(r"[0-9a-f]{64}")
SHELL_CIDX_PATTERN = re.compile(
    r"(?i)(?:^|[;&|()\s])cidx(?:\s|$)|(?:command\s+-v|which|type)\s+cidx"
)
EXPECTED_CIDX_TOOLS = ["read_span", "reindex", "search", "status"]
SOURCE_SUFFIXES = (".go", ".ts", ".tsx")
ENVIRONMENT_ALLOWLIST = (
    "CODEX_HOME",
    "HOME",
    "LANG",
    "LC_ALL",
    "LC_CTYPE",
    "LOGNAME",
    "PATH",
    "SHELL",
    "SSL_CERT_DIR",
    "SSL_CERT_FILE",
    "TERM",
    "TMPDIR",
    "USER",
)
SCHEMA_PROBE_TASK = "_schema_probe"
SCHEMA_PROBE_PROMPT = (
    "This is an input-accounting control. Do not call any tool. Return only "
    "this JSON object, preserving the values exactly: "
    '{"task_id":"_schema_probe","answer":"probe","evidence":'
    '[{"path":"probe","symbol":null,"start_line":null,'
    '"end_line":null,"supports":"probe"}],'
    '"uncertainties":[]}'
)


class ExperimentError(RuntimeError):
    pass


def experiment_protocol(manifest: dict[str, Any]) -> str:
    controls = manifest.get("controls")
    configured = controls.get("session_trace_protocol") if isinstance(controls, dict) else None
    if configured == POLICY_TRACE_PROTOCOL:
        if manifest.get("arms") is None:
            raise ExperimentError(
                "forced-cidx-policy-v2 requires manifest-defined neutral_cidx and directed_cidx arms"
            )
        return configured
    if manifest.get("arms") is not None:
        raise ExperimentError(
            "manifest-defined prompt-policy arms require forced-cidx-policy-v2"
        )
    try:
        return resolve_trace_protocol(manifest)
    except ValueError as exc:
        raise ExperimentError(str(exc)) from exc


def sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def require_sha256(value: Any, context: str) -> str:
    if not isinstance(value, str) or not SHA256_PATTERN.fullmatch(value):
        raise ExperimentError(f"{context} must be a lowercase SHA-256")
    return value


def legacy_arms() -> dict[str, dict[str, Any]]:
    """Return the historical two-arm contract without changing old manifests."""
    return {
        BASELINE_ARM: {
            "id": BASELINE_ARM,
            "cidx_exposed": False,
            "prompt_suffix": "",
            "mcp": {},
        },
        CIDX_ARM: {
            "id": CIDX_ARM,
            "cidx_exposed": True,
            "prompt_suffix": "",
            "mcp": {},
        },
    }


def manifest_arms(manifest: dict[str, Any]) -> tuple[dict[str, dict[str, Any]], bool]:
    """Resolve the historical fixed arms or a two-arm manifest-defined policy."""
    configured = manifest.get("arms")
    if configured is None:
        return legacy_arms(), True
    if isinstance(configured, dict):
        arm_items: list[dict[str, Any]] = []
        for arm_id, item in configured.items():
            if not isinstance(item, dict):
                raise ExperimentError("manifest arm must be an object")
            if "id" in item and item["id"] != arm_id:
                raise ExperimentError("manifest arm object id must match its arms key")
            arm_items.append({"id": arm_id} | item)
    elif isinstance(configured, list):
        arm_items = configured
    else:
        arm_items = []
    if len(arm_items) != 2:
        raise ExperimentError("manifest arms must define exactly two arm objects")

    arms: dict[str, dict[str, Any]] = {}
    for item in arm_items:
        if not isinstance(item, dict):
            raise ExperimentError("manifest arm must be an object")
        arm_id = item.get("id")
        if not isinstance(arm_id, str) or not ARM_ID_PATTERN.fullmatch(arm_id):
            raise ExperimentError("manifest arm id must be a safe non-empty identifier")
        if arm_id in arms:
            raise ExperimentError(f"duplicate manifest arm id: {arm_id}")
        cidx_exposed = item.get("cidx_exposed")
        if not isinstance(cidx_exposed, bool):
            raise ExperimentError(f"manifest arm {arm_id} must declare boolean cidx_exposed")
        suffix = item.get("prompt_suffix")
        if not isinstance(suffix, str):
            raise ExperimentError(f"manifest arm {arm_id} prompt_suffix must be a string")
        declared_suffix_hash = item.get("prompt_suffix_sha256")
        actual_suffix_hash = sha256_text(suffix)
        if declared_suffix_hash is not None and declared_suffix_hash != actual_suffix_hash:
            raise ExperimentError(f"manifest arm {arm_id} prompt suffix digest mismatch")
        mcp = item.get("mcp", {})
        if not isinstance(mcp, dict):
            raise ExperimentError(f"manifest arm {arm_id} mcp configuration must be an object")
        if not cidx_exposed and mcp:
            raise ExperimentError(f"manifest arm {arm_id} cannot configure MCP when cidx is hidden")
        arms[arm_id] = {
            "id": arm_id,
            "cidx_exposed": cidx_exposed,
            "prompt_suffix": suffix,
            "mcp": mcp,
        }

    if set(arms) != set(POLICY_ARM_IDS):
        raise ExperimentError(
            "manifest-defined arms must be exactly neutral_cidx and directed_cidx"
        )
    if not all(arm["cidx_exposed"] for arm in arms.values()):
        raise ExperimentError("manifest-defined prompt-policy arms must both expose cidx")
    return arms, False


def arm_mcp_config(arm: dict[str, Any]) -> tuple[str, str]:
    """Resolve the cidx server name and approval mode from one arm's config."""
    mcp = arm["mcp"]
    unknown_fields = set(mcp) - {
        "server",
        "server_name",
        "approval_mode",
        "default_tools_approval_mode",
    }
    if unknown_fields:
        raise ExperimentError(
            f"manifest arm {arm['id']} has unknown MCP fields: {sorted(unknown_fields)}"
        )
    server_name = mcp.get("server_name", mcp.get("server", "cidx"))
    approval_mode = mcp.get(
        "approval_mode", mcp.get("default_tools_approval_mode", "approve")
    )
    if server_name != "cidx":
        raise ExperimentError("only the frozen cidx MCP server may be exposed")
    if not isinstance(approval_mode, str) or not approval_mode:
        raise ExperimentError(f"manifest arm {arm['id']} has invalid MCP approval mode")
    return server_name, approval_mode


def rendered_prompt(template: str, task_id: str, question: str, arm: dict[str, Any]) -> str:
    return (
        template.replace("{{TASK_ID}}", task_id).replace("{{QUESTION}}", question)
        + arm["prompt_suffix"]
    )


def policy_manifest_contract(
    manifest: dict[str, Any],
    arms: dict[str, dict[str, Any]],
    *,
    session_trace_protocol: str,
    legacy_arms_mode: bool,
) -> dict[str, Any] | None:
    """Validate the sole-intervention policy contract for a new manifest."""
    if legacy_arms_mode:
        if session_trace_protocol == POLICY_TRACE_PROTOCOL:
            raise ExperimentError(
                "forced-cidx-policy-v2 rejects legacy asymmetric baseline/cidx_fts arms"
            )
        return None
    if session_trace_protocol != POLICY_TRACE_PROTOCOL:
        raise ExperimentError("manifest-defined arms require forced-cidx-policy-v2")
    prompt_policy = manifest.get("prompt_policy")
    if not isinstance(prompt_policy, dict):
        raise ExperimentError("policy manifest requires a top-level prompt_policy object")
    neutral_id = prompt_policy.get("neutral_arm_id")
    directed_id = prompt_policy.get("directed_arm_id")
    if neutral_id != POLICY_NEUTRAL_ARM or directed_id != POLICY_DIRECTED_ARM:
        raise ExperimentError(
            "prompt_policy must configure neutral_arm_id=neutral_cidx and directed_arm_id=directed_cidx"
        )
    neutral = arms[neutral_id]
    directed = arms[directed_id]
    if neutral["prompt_suffix"] != "":
        raise ExperimentError("neutral_cidx prompt_suffix must be empty")
    if not directed["prompt_suffix"]:
        raise ExperimentError("directed_cidx prompt_suffix must be non-empty")
    declared_suffix_hash = require_sha256(
        prompt_policy.get("directed_suffix_sha256"),
        "prompt_policy.directed_suffix_sha256",
    )
    if sha256_text(directed["prompt_suffix"]) != declared_suffix_hash:
        raise ExperimentError("directed_cidx prompt suffix differs from prompt_policy digest")
    neutral_mcp_hash = canonical_json_sha256(neutral["mcp"])
    directed_mcp_hash = canonical_json_sha256(directed["mcp"])
    if neutral_mcp_hash != directed_mcp_hash:
        raise ExperimentError("policy arm MCP objects must be canonical-identical")
    neutral_mcp = arm_mcp_config(neutral)
    directed_mcp = arm_mcp_config(directed)
    if neutral_mcp != directed_mcp:
        raise ExperimentError("policy arm resolved MCP settings must be identical")
    return {
        "neutral_arm_id": neutral_id,
        "directed_arm_id": directed_id,
        "directed_suffix_sha256": declared_suffix_hash,
        "mcp_sha256": neutral_mcp_hash,
        "mcp_server_name": neutral_mcp[0],
        "mcp_approval_mode": neutral_mcp[1],
    }


def schedule_value(manifest: dict[str, Any], name: str) -> Any:
    """Resolve one policy schedule declaration without allowing disagreement."""
    values = []
    if name in manifest:
        values.append(manifest[name])
    schedule = manifest.get("schedule")
    if isinstance(schedule, dict) and name in schedule:
        values.append(schedule[name])
    if len(values) == 2 and values[0] != values[1]:
        raise ExperimentError(f"conflicting top-level and schedule {name}")
    return values[0] if values else None


def paired_arm_order(
    arms: dict[str, dict[str, Any]], first_arm: str
) -> tuple[str, str]:
    if first_arm not in arms:
        raise ExperimentError(f"first_arm is not a declared arm: {first_arm!r}")
    other_arms = [arm_id for arm_id in arms if arm_id != first_arm]
    if len(other_arms) != 1 or other_arms[0] == first_arm:
        raise ExperimentError("each task must execute exactly two distinct arms")
    return first_arm, other_arms[0]


def validate_task_schedule(
    manifest: dict[str, Any],
    arms: dict[str, dict[str, Any]],
    *,
    legacy_arms_mode: bool,
) -> None:
    tasks = manifest.get("tasks", [])
    if not isinstance(tasks, list):
        raise ExperimentError("manifest tasks must be an array")
    task_ids: set[str] = set()
    sequences: set[int] = set()
    first_arm_counts = {arm_id: 0 for arm_id in arms}
    for task in tasks:
        if not isinstance(task, dict):
            raise ExperimentError("manifest task must be an object")
        task_id = task.get("task_id")
        if not isinstance(task_id, str) or not task_id:
            raise ExperimentError("manifest task_id must be a non-empty string")
        if task_id in task_ids:
            raise ExperimentError(f"duplicate task_id: {task_id}")
        task_ids.add(task_id)
        sequence = task.get("sequence")
        if not isinstance(sequence, int) or isinstance(sequence, bool) or sequence < 1:
            raise ExperimentError(f"task {task_id} sequence must be a positive integer")
        if sequence in sequences:
            raise ExperimentError(f"duplicate task sequence: {sequence}")
        sequences.add(sequence)
        first_arm, second_arm = paired_arm_order(arms, task.get("first_arm"))
        if first_arm == second_arm:
            raise ExperimentError(f"task {task_id} does not have two distinct arms")
        first_arm_counts[first_arm] += 1
    if sequences != set(range(1, len(tasks) + 1)):
        raise ExperimentError("task sequences must be contiguous from 1 through task count")
    if legacy_arms_mode:
        return
    schedule = manifest.get("schedule")
    if not isinstance(schedule, dict):
        raise ExperimentError("forced-cidx-policy-v2 requires a schedule object")
    required_schedule = {
        "pair_count": 30,
        "neutral_first_pairs": 15,
        "directed_first_pairs": 15,
    }
    for name, expected in required_schedule.items():
        if name not in schedule:
            raise ExperimentError(f"forced-cidx-policy-v2 schedule requires {name}")
        declared = schedule_value(manifest, name)
        if not isinstance(declared, int) or isinstance(declared, bool):
            raise ExperimentError(f"{name} must be an integer")
        if declared != expected:
            raise ExperimentError(
                f"forced-cidx-policy-v2 {name} must equal frozen value {expected}"
            )
    if len(tasks) != required_schedule["pair_count"]:
        raise ExperimentError("forced-cidx-policy-v2 requires exactly 30 tasks")
    expected_first_arm_counts = {
        POLICY_NEUTRAL_ARM: required_schedule["neutral_first_pairs"],
        POLICY_DIRECTED_ARM: required_schedule["directed_first_pairs"],
    }
    if first_arm_counts != expected_first_arm_counts:
        raise ExperimentError(
            "policy first-arm counts differ from the frozen 15/15 schedule"
        )


def policy_prompt_rendering_hashes(
    manifest: dict[str, Any],
    arms: dict[str, dict[str, Any]],
    sources: list[dict[str, dict[str, Any]]],
    policy_contract: dict[str, Any],
) -> dict[str, dict[str, Any]]:
    """Prove the directed suffix is the sole rendered-prompt difference."""
    neutral = arms[policy_contract["neutral_arm_id"]]
    directed = arms[policy_contract["directed_arm_id"]]
    suffix = directed["prompt_suffix"]
    renderings: dict[str, dict[str, Any]] = {}
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        question = sources[task["question_source_index"]][task_id]["text"]
        neutral_prompt = rendered_prompt(manifest["prompt_template"], task_id, question, neutral)
        directed_prompt = rendered_prompt(manifest["prompt_template"], task_id, question, directed)
        if not directed_prompt.endswith(suffix):
            raise ExperimentError(f"directed prompt does not end in its exact suffix: {task_id}")
        directed_without_suffix = directed_prompt[: -len(suffix)]
        if directed_without_suffix != neutral_prompt:
            raise ExperimentError(
                f"removing directed suffix does not yield the neutral prompt: {task_id}"
            )
        renderings[task_id] = {
            "neutral_rendered_prompt_sha256": sha256_text(neutral_prompt),
            "directed_rendered_prompt_sha256": sha256_text(directed_prompt),
            "directed_without_suffix_sha256": sha256_text(directed_without_suffix),
            "removing_directed_suffix_yields_neutral": True,
        }
    return renderings


def policy_trace_available() -> bool:
    return build_policy_session_trace is not None


def frozen_trace_builder_hashes(manifest: dict[str, Any]) -> dict[str, str]:
    """Collect declared policy/passive builder hashes from the frozen manifest."""
    prompt_policy = manifest.get("prompt_policy")
    freeze = manifest.get("freeze")
    containers = [
        prompt_policy if isinstance(prompt_policy, dict) else {},
        freeze if isinstance(freeze, dict) else {},
    ]
    if isinstance(freeze, dict) and isinstance(freeze.get("trace_builders"), dict):
        containers.append(freeze["trace_builders"])
    aliases = {
        "policy": (
            "policy_trace_builder_sha256",
            "policy_trace_module_sha256",
        ),
        "passive": (
            "passive_trace_builder_sha256",
            "passive_trace_module_sha256",
        ),
    }
    result: dict[str, str] = {}
    for role, names in aliases.items():
        values = [container[name] for container in containers for name in names if name in container]
        shorthand_names = {
            "policy": ("policy", "policy_v2", "forced_cidx_policy_v2"),
            "passive": ("passive", "passive_v1"),
        }[role]
        for container in containers:
            for name in shorthand_names:
                value = container.get(name)
                if isinstance(value, dict) and "sha256" in value:
                    values.append(value["sha256"])
                elif isinstance(value, str):
                    values.append(value)
        if not values:
            continue
        normalized = [require_sha256(value, f"declared {role} trace builder hash") for value in values]
        if len(set(normalized)) != 1:
            raise ExperimentError(f"conflicting declared {role} trace builder hashes")
        result[role] = normalized[0]
    return result


def verify_frozen_trace_builder_hashes(
    manifest: dict[str, Any],
    session_trace_protocol: str,
    identity: dict[str, Any],
) -> dict[str, str]:
    declared = frozen_trace_builder_hashes(manifest)
    if session_trace_protocol != POLICY_TRACE_PROTOCOL:
        return declared
    if set(declared) != {"policy", "passive"}:
        raise ExperimentError(
            "forced-cidx-policy-v2 requires frozen policy and passive trace hashes"
        )
    actual = {
        "policy": identity["sha256"],
        "passive": identity["delegated_sha256"],
    }
    for role, expected in declared.items():
        if actual[role] != expected:
            raise ExperimentError(
                f"frozen {role} trace builder hash differs from the live module"
            )
    return declared


def verify_frozen_execution_code(
    manifest: dict[str, Any], project_root: Path
) -> dict[str, dict[str, str]]:
    """Bind the policy diagnostic to its committed runner and scorer bytes."""
    freeze = manifest.get("freeze")
    configured = freeze.get("execution_code") if isinstance(freeze, dict) else None
    if not isinstance(configured, dict):
        raise ExperimentError("policy manifest requires freeze.execution_code")
    expected_paths = {
        "runner": Path(__file__).resolve(),
        "scorer": Path(__file__).resolve().with_name("score-assistant-ab.py"),
    }
    result: dict[str, dict[str, str]] = {}
    for role, expected_path in expected_paths.items():
        identity = configured.get(role)
        if (
            not isinstance(identity, dict)
            or not isinstance(identity.get("path"), str)
        ):
            raise ExperimentError(f"policy manifest requires {role} execution identity")
        declared_path = resolve_project_path(project_root, identity["path"])
        declared_hash = require_sha256(
            identity.get("sha256"), f"freeze.execution_code.{role}.sha256"
        )
        if declared_path != expected_path or not expected_path.is_file():
            raise ExperimentError(f"frozen {role} path is not the expected execution file")
        if sha256_file(expected_path) != declared_hash:
            raise ExperimentError(f"frozen {role} hash differs from the live file")
        result[role] = {"path": identity["path"], "sha256": declared_hash}
    return result


def trace_builder_identity(session_trace_protocol: str) -> dict[str, Any]:
    if session_trace_protocol == PASSIVE_TRACE_PROTOCOL:
        trace_path = Path(__file__).resolve().with_name("assistant_session_trace.py")
        return {
            "module": trace_path.name,
            "schema_version": TRACE_SCHEMA_VERSION,
            "sha256": sha256_file(trace_path),
            "available": True,
        }
    if session_trace_protocol == POLICY_TRACE_PROTOCOL:
        trace_path = Path(__file__).resolve().with_name("assistant_session_policy_trace.py")
        delegated_path = Path(__file__).resolve().with_name("assistant_session_trace.py")
        return {
            "module": trace_path.name,
            "schema_version": POLICY_TRACE_SCHEMA_VERSION,
            "sha256": sha256_file(trace_path) if trace_path.is_file() else None,
            "delegated_module": delegated_path.name,
            "delegated_schema_version": TRACE_SCHEMA_VERSION,
            "delegated_sha256": sha256_file(delegated_path),
            "available": policy_trace_available(),
        }
    return {
        "module": None,
        "schema_version": None,
        "sha256": None,
        "available": False,
    }


def read_json(path: Path) -> dict[str, Any]:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ExperimentError(f"cannot read JSON {path}: {exc}") from exc


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


def canonical_json_sha256(value: Any) -> str:
    encoded = json.dumps(
        value,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def run_checked(command: list[str], cwd: Path | None = None) -> str:
    completed = subprocess.run(
        command,
        cwd=cwd,
        check=False,
        capture_output=True,
        text=True,
    )
    if completed.returncode != 0:
        raise ExperimentError(
            f"command failed ({completed.returncode}): {shlex.join(command)}\n"
            f"stdout: {completed.stdout}\nstderr: {completed.stderr}"
        )
    return completed.stdout.strip()


def resolve_project_path(project_root: Path, value: str) -> Path:
    path = Path(value)
    if not path.is_absolute():
        path = project_root / path
    return path.resolve()


def local_bindings(project_root: Path, path: Path) -> dict[str, Path]:
    raw = read_json(path)
    result: dict[str, Path] = {}
    for corpus_id, value in raw.items():
        if not isinstance(corpus_id, str) or not isinstance(value, str):
            raise ExperimentError("corpus binding keys and values must be strings")
        result[corpus_id] = resolve_project_path(project_root, value)
    return result


def isolated_environment() -> dict[str, str]:
    environment = {
        key: os.environ[key]
        for key in ENVIRONMENT_ALLOWLIST
        if key in os.environ
    }
    environment.pop("VOYAGE_API_KEY", None)
    return environment


def copy_source_worktree(source: Path) -> Path:
    destination = Path(tempfile.mkdtemp(prefix="cidx-ab-source-"))

    def ignore(_directory: str, names: list[str]) -> set[str]:
        return {name for name in names if name == ".cidx"}

    shutil.copytree(source, destination, dirs_exist_ok=True, symlinks=True, ignore=ignore)
    if (destination / ".cidx").exists():
        raise ExperimentError("isolated source unexpectedly contains .cidx")
    return destination


def copy_cidx_state(source: Path) -> tuple[Path, str]:
    destination = Path(tempfile.mkdtemp(prefix="cidx-ab-state-"))
    config = source / ".cidx" / "config.json"
    database = source / ".cidx" / "db" / "index.db"
    (destination / "db").mkdir(mode=0o700)
    shutil.copy2(config, destination / "config.json")
    shutil.copy2(database, destination / "db" / "index.db")
    return destination, sha256_file(destination / "db" / "index.db")


def verify_isolated_source(root: Path, corpus: dict[str, Any]) -> dict[str, str]:
    commit = run_checked(["git", "rev-parse", "HEAD"], root)
    tree = run_checked(["git", "rev-parse", "HEAD^{tree}"], root)
    dirty = run_checked(
        ["git", "status", "--porcelain=v1", "--untracked-files=normal"], root
    )
    if commit != corpus["pinned_commit"] or tree != corpus["expected_tree_hash"]:
        raise ExperimentError("isolated source identity differs from frozen corpus")
    if dirty:
        raise ExperimentError(f"isolated source is dirty:\n{dirty}")
    return {"commit": commit, "tree": tree, "dirty": dirty}


def cleanup_isolated(path: Path | None, prefix: str) -> None:
    if path is None:
        return
    temporary = Path(tempfile.gettempdir()).resolve()
    resolved = path.resolve()
    if resolved.parent != temporary or not resolved.name.startswith(prefix):
        raise ExperimentError(f"refuse unsafe isolated cleanup: {resolved}")
    shutil.rmtree(resolved)


def question_sources(
    project_root: Path, manifest: dict[str, Any]
) -> list[dict[str, dict[str, Any]]]:
    for input_name in ("taxonomy", "truth_source"):
        spec = manifest[input_name]
        path = resolve_project_path(project_root, spec["path"])
        if sha256_file(path) != spec["sha256"]:
            raise ExperimentError(f"{input_name} digest mismatch: {path}")

    sources: list[dict[str, dict[str, Any]]] = []
    for spec in manifest["question_sources"]:
        path = resolve_project_path(project_root, spec["path"])
        if sha256_file(path) != spec["sha256"]:
            raise ExperimentError(f"question source digest mismatch: {path}")
        payload = read_json(path)
        cases = {case["id"]: case for case in payload.get("cases", [])}
        sources.append(cases)
    for task in manifest["tasks"]:
        index = task["question_source_index"]
        try:
            case = sources[index][task["task_id"]]
        except (IndexError, KeyError) as exc:
            raise ExperimentError(f"missing question case: {task['task_id']}") from exc
        if case.get("digest") != task["question_digest"]:
            raise ExperimentError(f"question digest mismatch: {task['task_id']}")

    identity = manifest["question_set_identity"]
    scheduled_questions = []
    for task in manifest["tasks"]:
        scheduled_questions.append(
            {
                key: task[key]
                for key in (
                    "sequence",
                    "task_id",
                    "question_source_index",
                    "question_digest",
                    "corpus_id",
                )
            }
            | {
                "language": sources[task["question_source_index"]][
                    task["task_id"]
                ]["language"]
            }
        )
    identity_input = {
        "id": identity["id"],
        "version": identity["version"],
        "taxonomy": manifest["taxonomy"],
        "question_sources": manifest["question_sources"],
        "scheduled_questions": scheduled_questions,
    }
    actual_identity = canonical_json_sha256(identity_input)
    if actual_identity != identity["sha256"]:
        raise ExperimentError(
            "question-set identity mismatch: "
            f"got {actual_identity}, want {identity['sha256']}"
        )
    if identity["case_count"] != len(manifest["tasks"]):
        raise ExperimentError("question-set case count differs from scheduled tasks")
    slice_counts: dict[str, int] = {}
    for task in manifest["tasks"]:
        language = sources[task["question_source_index"]][task["task_id"]][
            "language"
        ]
        slice_counts[language] = slice_counts.get(language, 0) + 1
    if slice_counts != identity["language_slice_counts"]:
        raise ExperimentError(
            "question-set language slices differ from frozen identity: "
            f"got {slice_counts}, want {identity['language_slice_counts']}"
        )
    return sources


def verify_corpus(
    corpus: dict[str, Any], root: Path, cidx_binary: Path
) -> dict[str, Any]:
    if not root.is_dir():
        raise ExperimentError(f"corpus binding is not a directory: {root}")
    commit = run_checked(["git", "rev-parse", "HEAD"], root)
    if commit != corpus["pinned_commit"]:
        raise ExperimentError(
            f"wrong corpus commit for {corpus['corpus_id']}: {commit}"
        )
    tree = run_checked(["git", "rev-parse", "HEAD^{tree}"], root)
    if tree != corpus["expected_tree_hash"]:
        raise ExperimentError(
            f"wrong corpus tree for {corpus['corpus_id']}: {tree}"
        )
    dirty = run_checked(
        ["git", "status", "--porcelain=v1", "--untracked-files=normal"], root
    )
    if dirty:
        raise ExperimentError(f"dirty corpus {corpus['corpus_id']}:\n{dirty}")

    state_root = root / ".cidx"
    config_path = state_root / "config.json"
    db_path = state_root / "db" / "index.db"
    if not config_path.is_file() or not db_path.is_file():
        raise ExperimentError(
            f"staged experiment state is missing for {corpus['corpus_id']}"
        )
    config = read_json(config_path)
    search = config.get("search", {})
    evaluation_state = corpus.get("evaluation_state", {})
    if sha256_file(config_path) != evaluation_state.get("config_sha256"):
        raise ExperimentError(
            f"cidx config differs from frozen state for {corpus['corpus_id']}"
        )
    if sha256_file(db_path) != evaluation_state.get("index_db_sha256"):
        raise ExperimentError(
            f"cidx index differs from frozen state for {corpus['corpus_id']}"
        )
    if search != evaluation_state.get("search"):
        raise ExperimentError(
            f"cidx search config differs from frozen state for {corpus['corpus_id']}"
        )
    if config.get("mcp") != evaluation_state.get("mcp"):
        raise ExperimentError(
            f"cidx MCP config differs from frozen state for {corpus['corpus_id']}"
        )
    if (
        search.get("default_mode") != "fts"
        or search.get("allow_paid_query_embedding") is not False
    ):
        raise ExperimentError(
            f"cidx state is not provider-free FTS for {corpus['corpus_id']}"
        )
    status_text = run_checked(
        [str(cidx_binary), "status", "--json", "--root", str(root)]
    )
    try:
        status = json.loads(status_text)
    except json.JSONDecodeError as exc:
        raise ExperimentError(f"invalid cidx status for {root}: {exc}") from exc
    for name in ("stale_count", "unindexed_count", "deleted_count"):
        if status.get(name) != 0:
            raise ExperimentError(
                f"cidx status {name}={status.get(name)} for {corpus['corpus_id']}"
            )
    return {
        "root": str(root),
        "commit": commit,
        "tree": tree,
        "config_sha256": sha256_file(config_path),
        "index_db_sha256": sha256_file(db_path),
        "status": status,
    }


def verify_frozen_tool_contract(
    manifest: dict[str, Any], tool_schema: dict[str, Any]
) -> None:
    frozen = manifest.get("freeze", {}).get("tool_contract_preflight", {})
    comparisons = {
        "tool_names": tool_schema["names"],
        "definition_sha256": tool_schema["sha256"],
        "description_sha256": tool_schema["description_sha256"],
        "input_schema_sha256": tool_schema["input_schema_sha256"],
        "functional_output_probe_sha256": tool_schema[
            "functional_output_probe_sha256"
        ],
        "provider_credentials_present": tool_schema[
            "provider_credentials_present"
        ],
        "result_representation": tool_schema["result_representation"],
    }
    for field, actual in comparisons.items():
        if frozen.get(field) != actual:
            raise ExperimentError(
                f"live cidx tool contract {field} differs from frozen manifest: "
                f"got {actual!r}, want {frozen.get(field)!r}"
            )


def verify_mcp_tools(
    mcp_binary: Path,
    source_root: Path,
    state_root: Path,
    result_representation: str,
    probe_query: str,
) -> dict[str, Any]:
    preflight_environment = isolated_environment()
    if "VOYAGE_API_KEY" in preflight_environment:
        raise ExperimentError("FTS preflight must not expose VOYAGE_API_KEY")
    process = subprocess.Popen(
        [
            str(mcp_binary),
            "--source-root",
            str(source_root),
            "--state-root",
            str(state_root),
            "--result-representation",
            result_representation,
        ],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
        env=preflight_environment,
    )
    assert process.stdin is not None
    assert process.stdout is not None
    initialize = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-11-25",
            "capabilities": {},
            "clientInfo": {"name": "cidx-assistant-ab-preflight", "version": "1"},
        },
    }
    process.stdin.write(json.dumps(initialize) + "\n")
    process.stdin.flush()
    try:
        initialize_response = json.loads(process.stdout.readline())
    except json.JSONDecodeError as exc:
        process.kill()
        raise ExperimentError("invalid cidx initialize response") from exc
    if initialize_response.get("id") != 1 or "error" in initialize_response:
        process.kill()
        raise ExperimentError(f"cidx initialize failed: {initialize_response}")
    process.stdin.write(
        json.dumps(
            {
                "jsonrpc": "2.0",
                "method": "notifications/initialized",
                "params": {},
            }
        )
        + "\n"
    )
    process.stdin.write(
        json.dumps(
            {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/list",
                "params": {},
            }
        )
        + "\n"
    )
    process.stdin.flush()
    try:
        tools_response = json.loads(process.stdout.readline())
    except json.JSONDecodeError as exc:
        process.kill()
        raise ExperimentError("invalid cidx tools/list response") from exc

    def call_tool(call_id: int, name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        process.stdin.write(
            json.dumps(
                {
                    "jsonrpc": "2.0",
                    "id": call_id,
                    "method": "tools/call",
                    "params": {"name": name, "arguments": arguments},
                }
            )
            + "\n"
        )
        process.stdin.flush()
        try:
            response = json.loads(process.stdout.readline())
        except json.JSONDecodeError as exc:
            process.kill()
            raise ExperimentError(f"invalid cidx {name} response") from exc
        if response.get("id") != call_id or "error" in response:
            process.kill()
            raise ExperimentError(f"cidx {name} call failed: {response}")
        return response

    def structured_result(response: dict[str, Any], name: str) -> dict[str, Any]:
        tool_result = response.get("result", {})
        value = tool_result.get("structuredContent")
        if not isinstance(value, dict):
            for block in tool_result.get("content", []):
                if not isinstance(block, dict) or not isinstance(block.get("text"), str):
                    continue
                try:
                    value = json.loads(block["text"])
                except json.JSONDecodeError:
                    continue
                if isinstance(value, dict):
                    break
        if not isinstance(value, dict):
            raise ExperimentError(f"cidx {name} lacks a usable result representation")
        return value

    status_response = call_tool(3, "status", {})
    status = structured_result(status_response, "status")
    status_fields = {
        "applied_active_serving_profile",
        "applied_canonical_text_profile",
        "applied_index_profile",
        "applied_source_profile",
        "applied_vector_space_profile",
        "applied_vector_storage_profile",
        "chunk_count",
        "deleted_count",
        "desired_active_serving_profile",
        "desired_canonical_text_profile",
        "desired_index_profile",
        "desired_source_profile",
        "desired_vector_space_profile",
        "desired_vector_storage_profile",
        "dirty",
        "embed_attempted_at",
        "embed_succeeded_at",
        "failed_count",
        "file_count",
        "generation_changed_during_status",
        "index_attempted_at",
        "index_error_count",
        "index_succeeded_at",
        "manifest_sha256",
        "observed_generation",
        "pending_count",
        "ready_count",
        "segment_count",
        "stale_count",
        "unindexed_count",
        "vector_coverage_denominator",
        "vector_coverage_numerator",
    }
    if set(status) != status_fields:
        raise ExperimentError(f"cidx status output contract mismatch: {status}")
    status_count_fields = {
        "chunk_count",
        "deleted_count",
        "failed_count",
        "file_count",
        "index_error_count",
        "observed_generation",
        "pending_count",
        "ready_count",
        "segment_count",
        "stale_count",
        "unindexed_count",
        "vector_coverage_denominator",
        "vector_coverage_numerator",
    }
    if any(
        not isinstance(status[field], int)
        or isinstance(status[field], bool)
        or status[field] < 0
        for field in status_count_fields
    ):
        raise ExperimentError(f"cidx status count contract mismatch: {status}")
    if (
        status["dirty"] is not False
        or status["generation_changed_during_status"] is not False
    ):
        raise ExperimentError(f"cidx status freshness contract mismatch: {status}")
    search_response = call_tool(
        4,
        "search",
        {"query": probe_query, "k": 1, "mode": "fts", "max_inline_bytes": 0},
    )
    search_result = structured_result(search_response, "search")
    if set(search_result) != {"results"} or not isinstance(
        search_result["results"], list
    ):
        raise ExperimentError("cidx search output contract mismatch")
    if not search_result["results"]:
        raise ExperimentError("cidx search output probe returned no locator")
    locator = search_result["results"][0]
    locator_fields = {
        "chunk_id",
        "path",
        "language",
        "kind",
        "qualified_symbol",
        "start_line",
        "end_line",
        "indexed_sha256",
        "match_sources",
    }
    if not isinstance(locator, dict) or set(locator) != locator_fields:
        raise ExperimentError(f"cidx search locator contract mismatch: {locator}")
    integer_fields = ("chunk_id", "start_line", "end_line")
    if any(
        not isinstance(locator[field], int) or isinstance(locator[field], bool)
        for field in integer_fields
    ):
        raise ExperimentError(f"cidx search locator integer type mismatch: {locator}")
    if (
        locator["chunk_id"] < 1
        or locator["start_line"] < 1
        or locator["end_line"] < locator["start_line"]
    ):
        raise ExperimentError(f"cidx search locator range mismatch: {locator}")
    for field in ("path", "language", "kind", "qualified_symbol"):
        if not isinstance(locator[field], str) or not locator[field]:
            raise ExperimentError(f"cidx search locator {field} mismatch: {locator}")
    if locator["language"] not in ("go", "typescript", "tsx"):
        raise ExperimentError(f"cidx search locator language mismatch: {locator}")
    if locator["kind"] not in ("function", "method", "type"):
        raise ExperimentError(f"cidx search locator kind mismatch: {locator}")
    if not isinstance(locator["indexed_sha256"], str) or not re.fullmatch(
        r"[0-9a-f]{64}", locator["indexed_sha256"]
    ):
        raise ExperimentError(f"cidx search locator SHA-256 mismatch: {locator}")
    allowed_match_sources = {"symbol", "path", "descriptive_fts", "fts"}
    if (
        not isinstance(locator["match_sources"], list)
        or not locator["match_sources"]
        or any(
            not isinstance(value, str) or value not in allowed_match_sources
            for value in locator["match_sources"]
        )
    ):
        raise ExperimentError(f"cidx search match_sources mismatch: {locator}")
    read_response = call_tool(
        5,
        "read_span",
        {
            "path": locator["path"],
            "start_line": locator["start_line"],
            "end_line": locator["end_line"],
            "expected_sha256": locator["indexed_sha256"],
        },
    )
    read_result = structured_result(read_response, "read_span")
    read_fields = {"path", "start_line", "end_line", "body", "indexed_sha256"}
    if (
        set(read_result) != read_fields
        or not isinstance(read_result["body"], str)
        or not read_result["body"]
    ):
        raise ExperimentError(f"cidx read_span output contract mismatch: {read_result}")
    if any(
        read_result[field] != locator[field]
        for field in ("path", "start_line", "end_line", "indexed_sha256")
    ):
        raise ExperimentError("cidx read_span output identity differs from locator")
    reindex_response = call_tool(6, "reindex", {"dry_run": True})
    reindex_result = structured_result(reindex_response, "reindex")
    reindex_fields = {
        "dry_run",
        "planned_files_updated",
        "planned_files_deleted",
        "planned_chunks",
        "planned_embeddings_reused",
        "planned_embeddings_pending",
    }
    if (
        set(reindex_result) != reindex_fields
        or reindex_result.get("dry_run") is not True
    ):
        raise ExperimentError(f"cidx reindex dry-run contract mismatch: {reindex_result}")
    if any(
        not isinstance(reindex_result[field], int)
        or isinstance(reindex_result[field], bool)
        or reindex_result[field] < 0
        for field in reindex_fields - {"dry_run"}
    ):
        raise ExperimentError(f"cidx reindex count contract mismatch: {reindex_result}")
    process.stdin.close()
    try:
        process.wait(timeout=30)
    except subprocess.TimeoutExpired as exc:
        process.kill()
        raise ExperimentError("cidx MCP preflight did not exit") from exc
    if (
        process.returncode != 0
        or tools_response.get("id") != 2
        or "error" in tools_response
    ):
        stderr = process.stderr.read() if process.stderr is not None else ""
        raise ExperimentError(
            "cidx MCP preflight failed: "
            f"tools={tools_response}; status={status_response}; stderr={stderr}"
        )
    raw_tools = tools_response["result"]["tools"]
    tools = sorted(tool["name"] for tool in raw_tools)
    if tools != EXPECTED_CIDX_TOOLS:
        raise ExperimentError(f"unexpected cidx MCP tools: {tools}")
    canonical = json.dumps(
        sorted(raw_tools, key=lambda item: item["name"]),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    for name in ("stale_count", "unindexed_count", "deleted_count"):
        if status.get(name) != 0:
            raise ExperimentError(f"isolated cidx status {name}={status.get(name)}")
    if status.get("dirty") is not False:
        raise ExperimentError("isolated cidx source is dirty")
    descriptions = {
        tool["name"]: tool.get("description")
        for tool in raw_tools
        if isinstance(tool.get("name"), str) and isinstance(tool.get("description"), str)
    }
    input_schemas = {
        tool["name"]: tool.get("inputSchema")
        for tool in raw_tools
        if isinstance(tool.get("name"), str)
    }
    if (
        sorted(descriptions) != EXPECTED_CIDX_TOOLS
        or sorted(input_schemas) != EXPECTED_CIDX_TOOLS
    ):
        raise ExperimentError("cidx tools must expose descriptions and input schemas")
    functional_probe = {
        "query": probe_query,
        "search_result_fields": sorted(search_result),
        "search_locator_fields": sorted(locator_fields),
        "read_span_fields": sorted(read_fields),
        "status_fields": sorted(status_fields),
        "reindex_dry_run_fields": sorted(reindex_fields),
    }
    return {
        "names": tools,
        "tools": sorted(raw_tools, key=lambda item: item["name"]),
        "sha256": hashlib.sha256(canonical).hexdigest(),
        "description_sha256": hashlib.sha256(
            json.dumps(descriptions, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
        ).hexdigest(),
        "input_schema_sha256": hashlib.sha256(
            json.dumps(input_schemas, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
        ).hexdigest(),
        "descriptions": descriptions,
        "provider_credentials_present": False,
        "isolated_status": status,
        "functional_output_probe": functional_probe,
        "functional_output_probe_sha256": canonical_json_sha256(functional_probe),
        "result_representation": result_representation,
    }


def codex_command(
    codex_binary: str,
    controls: dict[str, Any],
    schema: Path,
    root: Path,
    final_path: Path,
    arm: dict[str, Any],
    mcp_binary: Path,
    state_root: Path | None,
    result_representation: str,
) -> list[str]:
    command = [
        codex_binary,
        "exec",
        "--ignore-user-config",
        "--ignore-rules",
        "--strict-config",
        "--ephemeral",
        "--json",
        "--output-schema",
        str(schema),
        "--output-last-message",
        str(final_path),
        "--model",
        controls["model"],
        "--sandbox",
        controls["sandbox"],
        "--cd",
        str(root),
        "--config",
        f'model_reasoning_effort="{controls["reasoning_effort"]}"',
    ]
    for feature in controls.get("disabled_features", []):
        command.extend(["--disable", feature])
    if arm["cidx_exposed"]:
        if state_root is None:
            raise ExperimentError(f"cidx state root is required for arm {arm['id']}")
        server_name, approval_mode = arm_mcp_config(arm)
        command.extend(
            [
                "--config",
                f"mcp_servers.{server_name}.command={json.dumps(str(mcp_binary))}",
                "--config",
                f"mcp_servers.{server_name}.args="
                + json.dumps(
                    [
                        "--source-root",
                        str(root),
                        "--state-root",
                        str(state_root),
                        "--result-representation",
                        result_representation,
                    ]
                ),
                "--config",
                f"mcp_servers.{server_name}.default_tools_approval_mode="
                + json.dumps(approval_mode),
            ]
        )
    command.append("-")
    return command


def event_observation(
    events_path: Path,
    final_path: Path,
    trace: dict[str, Any],
) -> dict[str, Any]:
    parsed: list[dict[str, Any]] = []
    invalid_event_lines = 0
    if events_path.exists():
        for line in events_path.read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                invalid_event_lines += 1
                continue
            if isinstance(event, dict):
                parsed.append(event)

    usage: dict[str, Any] = {}
    commands: list[dict[str, Any]] = []
    mcp_calls: list[dict[str, Any]] = []
    source_paths: set[str] = set()
    for event in parsed:
        if event.get("type") == "turn.completed" and isinstance(
            event.get("usage"), dict
        ):
            usage = dict(event["usage"])
        if event.get("type") not in ("item.started", "item.completed"):
            continue
        item = event.get("item")
        if not isinstance(item, dict):
            continue
        kind = item.get("type")
        if kind == "command_execution" and event.get("type") == "item.completed":
            record = {
                key: item.get(key)
                for key in ("id", "command", "status", "exit_code")
                if key in item
            }
            commands.append(record)
            command_text = str(item.get("command", ""))
            for token in re.findall(r"[A-Za-z0-9_./@-]+", command_text):
                if token.lower().endswith(SOURCE_SUFFIXES) and not token.startswith("/"):
                    source_paths.add(token.lstrip("./"))
        if kind == "mcp_tool_call" and event.get("type") == "item.completed":
            mcp_calls.append(
                {
                    key: item.get(key)
                    for key in ("id", "server", "tool", "status", "arguments")
                    if key in item
                }
            )

    if "input_tokens" in usage:
        usage["uncached_input_tokens"] = max(
            0, usage.get("input_tokens", 0) - usage.get("cached_input_tokens", 0)
        )
        usage["model_total_tokens"] = usage.get("input_tokens", 0) + usage.get(
            "output_tokens", 0
        )

    final_value: Any = None
    final_error: str | None = None
    if final_path.exists():
        try:
            final_value = json.loads(final_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            final_error = str(exc)
    else:
        final_error = "missing final output"

    return {
        "event_count": len(parsed),
        "invalid_event_lines": invalid_event_lines,
        "usage": usage,
        "command_count": len(commands),
        "commands": commands,
        "mcp_call_count": len(mcp_calls),
        "mcp_calls": mcp_calls,
        "cidx_used": bool(mcp_calls),
        "discovery_actions": trace["actions"],
        "first_repository_discovery_action": (
            trace["actions"][0]["kind"] if trace["actions"] else None
        ),
        "source_paths_from_commands": sorted(source_paths),
        "final": final_value,
        "final_error": final_error,
    }


def execute_one(
    *,
    codex_binary: str,
    mcp_binary: Path,
    state_root: Path | None,
    controls: dict[str, Any],
    schema: Path,
    root: Path,
    output_dir: Path,
    arm: dict[str, Any],
    task_id: str,
    prompt: str,
    result_representation: str,
    session_trace_protocol: str,
) -> dict[str, Any]:
    output_dir.mkdir(parents=True, exist_ok=False)
    prompt_path = output_dir / "prompt.txt"
    events_path = output_dir / "events.jsonl"
    stderr_path = output_dir / "stderr.txt"
    final_path = output_dir / "final.json"
    observation_path = output_dir / "observation.json"
    prompt_path.write_text(prompt, encoding="utf-8")
    command = codex_command(
        codex_binary,
        controls,
        schema,
        root,
        final_path,
        arm,
        mcp_binary,
        state_root,
        result_representation,
    )
    write_json(
        output_dir / "command.json",
        {
            "argv": command,
            "cwd": str(root),
            "task_id": task_id,
            "arm": arm["id"],
        },
    )

    environment = isolated_environment()
    started = time.monotonic()
    timed_out = False
    with events_path.open("wb") as stdout, stderr_path.open("wb") as stderr:
        process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=stdout,
            stderr=stderr,
            cwd=root,
            env=environment,
            start_new_session=True,
        )
        try:
            process.communicate(
                input=prompt.encode("utf-8"), timeout=controls["timeout_seconds"]
            )
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait(timeout=10)
    elapsed = time.monotonic() - started
    if session_trace_protocol == POLICY_TRACE_PROTOCOL:
        if build_policy_session_trace is None:
            raise ExperimentError(
                "policy-v2 trace requested but assistant_session_policy_trace.py is unavailable"
            )
        trace = build_policy_session_trace(
            events_path,
            final_path,
            root,
            frozen_fts_default=True,
        )
    else:
        trace = build_session_trace(
            events_path,
            final_path,
            root,
            frozen_fts_default=True,
        )
    observation = event_observation(events_path, final_path, trace)
    if session_trace_protocol in (PASSIVE_TRACE_PROTOCOL, POLICY_TRACE_PROTOCOL):
        write_json(output_dir / "session-trace.json", trace)
    observation.update(
        {
            "task_id": task_id,
            "arm": arm["id"],
            "exit_code": process.returncode,
            "timed_out": timed_out,
            "elapsed_seconds": elapsed,
            "valid_execution": process.returncode == 0 and not timed_out,
        }
    )
    write_json(observation_path, observation)
    return observation


def control_violations(
    arm: dict[str, Any],
    observation: dict[str, Any],
    *,
    require_first_cidx_search: bool,
    session_trace_protocol: str,
) -> list[str]:
    violations: list[str] = []
    calls = observation.get("mcp_calls", [])
    if not arm["cidx_exposed"] and calls:
        violations.append(
            "baseline_mcp_call" if arm["id"] == BASELINE_ARM else "hidden_arm_mcp_call"
        )
    for call in calls:
        if call.get("server") != "cidx":
            violations.append("unexpected_mcp_server")
        if call.get("tool") == "reindex":
            violations.append("reindex_call")
        arguments = call.get("arguments")
        if (
            call.get("tool") == "search"
            and isinstance(arguments, dict)
            and arguments.get("mode") == "hybrid"
        ):
            violations.append("hybrid_search_call")
    for command in observation.get("commands", []):
        command_text = str(command.get("command", ""))
        if (
            session_trace_protocol != POLICY_TRACE_PROTOCOL
            and SHELL_CIDX_PATTERN.search(command_text)
        ):
            violations.append("shell_cidx_attempt")
    if arm["cidx_exposed"] and require_first_cidx_search:
        first = observation.get("first_repository_discovery_action")
        if first is None:
            violations.append("missing_required_first_cidx_search")
        elif first != "cidx_search":
            violations.append("first_repository_discovery_not_cidx_search")
    return sorted(set(violations))


def invalidating_control_violations(
    violations: list[str], session_trace_protocol: str
) -> list[str]:
    """Keep a policy shell invocation observable without discarding its turn."""
    if session_trace_protocol == POLICY_TRACE_PROTOCOL:
        return [violation for violation in violations if violation != "shell_cidx_attempt"]
    return violations


def policy_noncompliance_violations(observation: dict[str, Any]) -> list[str]:
    return sorted(
        {
            "shell_cidx_attempt"
            for command in observation.get("commands", [])
            if is_shell_cidx_attempt is not None
            and is_shell_cidx_attempt(str(command.get("command", "")))
        }
    )


def execute_isolated(
    *,
    codex_binary: str,
    mcp_binary: Path,
    cidx_binary: Path,
    controls: dict[str, Any],
    schema: Path,
    original_root: Path,
    corpus: dict[str, Any],
    output_dir: Path,
    arm: dict[str, Any],
    task_id: str,
    prompt: str,
    result_representation: str,
    require_first_cidx_search: bool,
    session_trace_protocol: str,
) -> dict[str, Any]:
    source_root: Path | None = None
    state_root: Path | None = None
    try:
        source_root = copy_source_worktree(original_root)
        source_before = verify_isolated_source(source_root, corpus)
        state_before: str | None = None
        if arm["cidx_exposed"]:
            state_root, state_before = copy_cidx_state(original_root)
        observation = execute_one(
            codex_binary=codex_binary,
            mcp_binary=mcp_binary,
            state_root=state_root,
            controls=controls,
            schema=schema,
            root=source_root,
            output_dir=output_dir,
            arm=arm,
            task_id=task_id,
            prompt=prompt,
            result_representation=result_representation,
            session_trace_protocol=session_trace_protocol,
        )
        source_after = verify_isolated_source(source_root, corpus)
        state_after = (
            sha256_file(state_root / "db" / "index.db")
            if state_root is not None
            else None
        )
        violations = control_violations(
            arm,
            observation,
            require_first_cidx_search=require_first_cidx_search,
            session_trace_protocol=session_trace_protocol,
        )
        policy_noncompliance = (
            policy_noncompliance_violations(observation)
            if session_trace_protocol == POLICY_TRACE_PROTOCOL
            else []
        )
        invalidating_violations = invalidating_control_violations(
            violations, session_trace_protocol
        )
        updates = {
            "source_before": source_before,
            "source_after": source_after,
            "state_database_sha256_before": state_before,
            "state_database_sha256_after": state_after,
            "control_violations": violations,
            "valid_execution": observation["valid_execution"]
            and not invalidating_violations,
            "isolated_source_removed_after_capture": True,
            "isolated_state_removed_after_capture": state_root is not None,
            "cidx_binary_sha256": sha256_file(cidx_binary),
            "mcp_binary_sha256": sha256_file(mcp_binary),
        }
        if session_trace_protocol == POLICY_TRACE_PROTOCOL:
            updates["policy_noncompliance"] = policy_noncompliance
        observation.update(updates)
        write_json(output_dir / "observation.json", observation)
        if any(
            violation in {
                "baseline_mcp_call",
                "hidden_arm_mcp_call",
                "unexpected_mcp_server",
            }
            for violation in violations
        ):
            raise ExperimentError(
                f"global tool-exposure violation in {task_id}/{arm['id']}: {violations}"
            )
        return observation
    finally:
        cleanup_isolated(source_root, "cidx-ab-source-")
        cleanup_isolated(state_root, "cidx-ab-state-")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--manifest",
        default="testdata/retrieval/assistant-ab-chi-rhf-v3.json",
    )
    parser.add_argument(
        "--bindings", default=".cidx/test/corpora.local.json"
    )
    parser.add_argument("--cidx-binary", required=True)
    parser.add_argument("--mcp-binary", required=True)
    parser.add_argument("--codex-binary", default="codex")
    parser.add_argument(
        "--artifact-root", default=".cidx/test/assistant-ab/runs"
    )
    parser.add_argument("--run-id")
    parser.add_argument("--only-task", action="append", default=[])
    parser.add_argument("--preflight-only", action="store_true")
    parser.add_argument("--schema-probes-only", action="store_true")
    parser.add_argument(
        "--mcp-result-representation",
        choices=("dual", "text", "structured"),
        default="structured",
    )
    args = parser.parse_args()
    if args.preflight_only and args.schema_probes_only:
        parser.error("--preflight-only and --schema-probes-only are mutually exclusive")

    project_root = Path(__file__).resolve().parent.parent
    manifest_path = resolve_project_path(project_root, args.manifest)
    bindings_path = resolve_project_path(project_root, args.bindings)
    cidx_binary = resolve_project_path(project_root, args.cidx_binary)
    mcp_binary = resolve_project_path(project_root, args.mcp_binary)
    artifact_root = resolve_project_path(project_root, args.artifact_root)
    schema = resolve_project_path(
        project_root, "schemas/evaluation/assistant-answer.schema.json"
    )
    manifest = read_json(manifest_path)
    controls = manifest["controls"]
    arms, legacy_arms_mode = manifest_arms(manifest)
    session_trace_protocol = experiment_protocol(manifest)
    policy_contract = policy_manifest_contract(
        manifest,
        arms,
        session_trace_protocol=session_trace_protocol,
        legacy_arms_mode=legacy_arms_mode,
    )
    validate_task_schedule(
        manifest, arms, legacy_arms_mode=legacy_arms_mode
    )
    trace_identity = trace_builder_identity(session_trace_protocol)
    if session_trace_protocol == POLICY_TRACE_PROTOCOL and not trace_identity["available"]:
        raise ExperimentError(
            "policy-v2 trace requested but assistant_session_policy_trace.py is unavailable"
        )
    frozen_trace_hashes = verify_frozen_trace_builder_hashes(
        manifest, session_trace_protocol, trace_identity
    )
    frozen_execution_code = (
        verify_frozen_execution_code(manifest, project_root)
        if session_trace_protocol == POLICY_TRACE_PROTOCOL
        else None
    )
    manifest_status = manifest.get("status")
    if manifest_status != "frozen_for_execution" and not (
        args.preflight_only and manifest_status == "frozen_for_external_review"
    ):
        raise ExperimentError(
            f"manifest is not executable: {manifest_status}"
        )
    frozen_representation = manifest.get("controls", {}).get(
        "treatment_result_representation"
    )
    if (
        frozen_representation is not None
        and args.mcp_result_representation != frozen_representation
    ):
        raise ExperimentError(
            "MCP result representation differs from frozen manifest: "
            f"got {args.mcp_result_representation}, want {frozen_representation}"
        )
    if not cidx_binary.is_file() or not os.access(cidx_binary, os.X_OK):
        raise ExperimentError(f"cidx binary is not executable: {cidx_binary}")
    if not mcp_binary.is_file() or not os.access(mcp_binary, os.X_OK):
        raise ExperimentError(f"MCP launcher is not executable: {mcp_binary}")
    if not schema.is_file():
        raise ExperimentError(f"answer schema is missing: {schema}")

    sources = question_sources(project_root, manifest)
    policy_prompt_renderings = (
        policy_prompt_rendering_hashes(manifest, arms, sources, policy_contract)
        if policy_contract is not None
        else None
    )
    bindings = local_bindings(project_root, bindings_path)
    corpus_specs = {item["corpus_id"]: item for item in manifest["corpora"]}
    corpus_records: dict[str, Any] = {}
    for corpus_id, corpus in corpus_specs.items():
        if corpus_id not in bindings:
            raise ExperimentError(f"missing local corpus binding: {corpus_id}")
        corpus_records[corpus_id] = verify_corpus(
            corpus, bindings[corpus_id], cidx_binary
        )
        frozen_state = corpus["evaluation_state"]
        if controls["search_default_k"] != frozen_state["search"]["return_k"]:
            raise ExperimentError(
                f"controls.search_default_k differs from {corpus_id} config"
            )
        if (
            controls["mcp_hard_max_inline_bytes"]
            != frozen_state["mcp"]["hard_max_inline_bytes"]
        ):
            raise ExperimentError(
                f"controls.mcp_hard_max_inline_bytes differs from {corpus_id} config"
            )
    preflight_source: Path | None = None
    preflight_state: Path | None = None
    try:
        first_corpus_id = manifest["tasks"][0]["corpus_id"]
        preflight_source = copy_source_worktree(bindings[first_corpus_id])
        verify_isolated_source(preflight_source, corpus_specs[first_corpus_id])
        preflight_state, _ = copy_cidx_state(bindings[first_corpus_id])
        tool_schema = verify_mcp_tools(
            mcp_binary,
            preflight_source,
            preflight_state,
            args.mcp_result_representation,
            sources[manifest["tasks"][0]["question_source_index"]][
                manifest["tasks"][0]["task_id"]
            ]["text"],
        )
        verify_frozen_tool_contract(manifest, tool_schema)
    finally:
        cleanup_isolated(preflight_source, "cidx-ab-source-")
        cleanup_isolated(preflight_state, "cidx-ab-state-")

    codex_version = run_checked([args.codex_binary, "--version"])
    codex_path_value = shutil.which(args.codex_binary)
    if codex_path_value is None:
        raise ExperimentError(f"Codex CLI is not on PATH: {args.codex_binary}")
    codex_path = Path(codex_path_value).resolve()
    login_check = subprocess.run(
        [args.codex_binary, "login", "status"],
        check=False,
        capture_output=True,
        text=True,
    )
    if login_check.returncode != 0:
        raise ExperimentError(
            f"Codex CLI login check failed: {login_check.stdout}{login_check.stderr}"
        )
    login_status = (login_check.stdout + login_check.stderr).strip()
    if "logged in" not in login_status.lower():
        raise ExperimentError(f"Codex CLI is not logged in: {login_status}")

    require_first_cidx_search = controls.get(
        "require_first_cidx_search", True if legacy_arms_mode else False
    )
    if not isinstance(require_first_cidx_search, bool):
        raise ExperimentError("controls.require_first_cidx_search must be boolean")
    if not legacy_arms_mode and require_first_cidx_search:
        raise ExperimentError(
            "manifest-defined prompt-policy arms must record first-search noncompliance, not invalidate it"
        )
    run_id = args.run_id or (
        str(manifest.get("manifest_id", "assistant-ab"))
        + "-"
        + dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    )
    run_root = artifact_root / run_id
    if run_root.exists():
        raise ExperimentError(f"run directory already exists: {run_root}")
    run_root.mkdir(parents=True)
    run_manifest = {
        "schema_version": 1,
        "run_id": run_id,
        "started_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "experiment_manifest": str(manifest_path.relative_to(project_root)),
        "experiment_manifest_sha256": sha256_file(manifest_path),
        "answer_schema_sha256": sha256_file(schema),
        "runner_sha256": sha256_file(Path(__file__).resolve()),
        "plan_sha256": sha256_file(
            resolve_project_path(
                project_root,
                manifest.get(
                    "plan",
                    "docs/implementation/ASSISTANT-AB-TEST-PLAN-V2.md",
                ),
            )
        ),
        "codex_version": codex_version,
        "codex_binary_sha256": sha256_file(codex_path),
        "cidx_binary_sha256": sha256_file(cidx_binary),
        "mcp_binary_sha256": sha256_file(mcp_binary),
        "cidx_tool_schema_sha256": tool_schema["sha256"],
        "cidx_tool_description_sha256": tool_schema["description_sha256"],
        "cidx_tool_input_schema_sha256": tool_schema["input_schema_sha256"],
        "cidx_tool_output_probe_sha256": tool_schema[
            "functional_output_probe_sha256"
        ],
        "cidx_tool_descriptions": tool_schema["descriptions"],
        "cidx_tools": tool_schema["names"],
        "mcp_result_representation": args.mcp_result_representation,
        "environment_variable_allowlist": sorted(isolated_environment()),
        "controls": controls,
        "corpora": corpus_records,
    }
    if session_trace_protocol == PASSIVE_TRACE_PROTOCOL:
        run_manifest["session_trace_protocol"] = session_trace_protocol
        run_manifest["session_trace_schema_version"] = trace_identity["schema_version"]
        run_manifest["session_trace_builder_sha256"] = trace_identity["sha256"]
    elif session_trace_protocol == POLICY_TRACE_PROTOCOL:
        run_manifest.update(
            {
                "session_trace_protocol": session_trace_protocol,
                "session_trace_schema_version": trace_identity["schema_version"],
                "session_trace_builder_sha256": trace_identity["sha256"],
                "session_trace_builder_module": trace_identity["module"],
                "session_trace_delegated_builder_schema_version": trace_identity[
                    "delegated_schema_version"
                ],
                "session_trace_delegated_builder_sha256": trace_identity[
                    "delegated_sha256"
                ],
                "session_trace_delegated_builder_module": trace_identity[
                    "delegated_module"
                ],
                "frozen_trace_builder_hashes": frozen_trace_hashes,
                "frozen_execution_code": frozen_execution_code,
            }
        )
    if not legacy_arms_mode:
        assert policy_contract is not None
        assert policy_prompt_renderings is not None
        prompt_renderings = policy_prompt_renderings
        run_manifest["arm_ids"] = list(arms)
        run_manifest["arms"] = [
            {
                "id": arm["id"],
                "cidx_exposed": arm["cidx_exposed"],
                "mcp": arm["mcp"],
                "prompt_suffix_sha256": sha256_text(arm["prompt_suffix"]),
            }
            for arm in arms.values()
        ]
        run_manifest["prompt_policy"] = {
            **policy_contract,
            "rendering_verification": prompt_renderings,
        }
        run_manifest["rendered_prompt_sha256"] = {
            task_id: {
                policy_contract["neutral_arm_id"]: values[
                    "neutral_rendered_prompt_sha256"
                ],
                policy_contract["directed_arm_id"]: values[
                    "directed_rendered_prompt_sha256"
                ],
            }
            for task_id, values in prompt_renderings.items()
        }
        run_manifest["base_prompt_sha256"] = {
            task_id: values["directed_without_suffix_sha256"]
            for task_id, values in prompt_renderings.items()
        }
    write_json(run_root / "run-manifest.json", run_manifest)
    write_json(run_root / "tool-schema.json", tool_schema)
    print(f"preflight passed; run={run_id}", flush=True)
    if args.preflight_only:
        return 0

    probe_corpus_id = manifest["tasks"][0]["corpus_id"]
    for arm in arms.values():
        print(f"schema probe: {arm['id']}", flush=True)
        probe = execute_isolated(
            codex_binary=str(codex_path),
            mcp_binary=mcp_binary,
            cidx_binary=cidx_binary,
            controls=controls,
            schema=schema,
            original_root=bindings[probe_corpus_id],
            corpus=corpus_specs[probe_corpus_id],
            output_dir=run_root / SCHEMA_PROBE_TASK / arm["id"],
            arm=arm,
            task_id=SCHEMA_PROBE_TASK,
            prompt=SCHEMA_PROBE_PROMPT,
            result_representation=args.mcp_result_representation,
            require_first_cidx_search=False,
            session_trace_protocol=session_trace_protocol,
        )
        if (
            not probe["valid_execution"]
            or probe["command_count"] != 0
            or probe["mcp_call_count"] != 0
            or probe["final"]
            != {
                "task_id": SCHEMA_PROBE_TASK,
                "answer": "probe",
                "evidence": [
                    {
                        "path": "probe",
                        "symbol": None,
                        "start_line": None,
                        "end_line": None,
                        "supports": "probe",
                    }
                ],
                "uncertainties": [],
            }
        ):
            raise ExperimentError(f"invalid schema probe: {arm['id']}")

    if args.schema_probes_only:
        run_manifest["finished_at"] = dt.datetime.now(dt.timezone.utc).isoformat()
        write_json(run_root / "run-manifest.json", run_manifest)
        print(f"schema probes complete: {run_root}", flush=True)
        return 0

    selected = set(args.only_task)
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        if selected and task_id not in selected:
            continue
        case = sources[task["question_source_index"]][task_id]
        for arm_id in paired_arm_order(arms, task["first_arm"]):
            arm = arms[arm_id]
            prompt = rendered_prompt(
                manifest["prompt_template"], task_id, case["text"], arm
            )
            print(
                f"task {task['sequence']:02d}/{len(manifest['tasks'])} "
                f"{task_id}: {arm_id}",
                flush=True,
            )
            observation = execute_isolated(
                codex_binary=str(codex_path),
                mcp_binary=mcp_binary,
                cidx_binary=cidx_binary,
                controls=controls,
                schema=schema,
                original_root=bindings[task["corpus_id"]],
                corpus=corpus_specs[task["corpus_id"]],
                output_dir=run_root / task_id / arm_id,
                arm=arm,
                task_id=task_id,
                prompt=prompt,
                result_representation=args.mcp_result_representation,
                require_first_cidx_search=require_first_cidx_search,
                session_trace_protocol=session_trace_protocol,
            )
            print(
                f"  exit={observation['exit_code']} timeout={observation['timed_out']} "
                f"input={observation['usage'].get('input_tokens')} "
                f"mcp_calls={observation['mcp_call_count']} "
                f"valid={observation['valid_execution']}",
                flush=True,
            )
        root = bindings[task["corpus_id"]]
        dirty = run_checked(
            ["git", "status", "--porcelain=v1", "--untracked-files=normal"], root
        )
        if dirty:
            raise ExperimentError(f"source tree changed after pair {task_id}:\n{dirty}")

    run_manifest["finished_at"] = dt.datetime.now(dt.timezone.utc).isoformat()
    write_json(run_root / "run-manifest.json", run_manifest)
    print(f"execution complete: {run_root}", flush=True)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ExperimentError as exc:
        print(f"assistant A/B error: {exc}", file=sys.stderr)
        raise SystemExit(2)
