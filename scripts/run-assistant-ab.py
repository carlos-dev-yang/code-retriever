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


BASELINE_ARM = "baseline"
CIDX_ARM = "cidx_fts"
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
    if (
        search.get("default_mode") != "fts"
        or search.get("allow_paid_query_embedding") is not False
    ):
        raise ExperimentError(
            f"cidx state is not FTS-default for {corpus['corpus_id']}"
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


def verify_mcp_tools(
    mcp_binary: Path, source_root: Path, state_root: Path
) -> dict[str, Any]:
    process = subprocess.Popen(
        [
            str(mcp_binary),
            "--source-root",
            str(source_root),
            "--state-root",
            str(state_root),
        ],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
        env=isolated_environment(),
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
    process.stdin.write(
        json.dumps(
            {
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {"name": "status", "arguments": {}},
            }
        )
        + "\n"
    )
    process.stdin.flush()
    try:
        status_response = json.loads(process.stdout.readline())
    except json.JSONDecodeError as exc:
        process.kill()
        raise ExperimentError("invalid cidx status response") from exc
    process.stdin.close()
    try:
        process.wait(timeout=30)
    except subprocess.TimeoutExpired as exc:
        process.kill()
        raise ExperimentError("cidx MCP preflight did not exit") from exc
    if (
        process.returncode != 0
        or tools_response.get("id") != 2
        or status_response.get("id") != 3
        or "error" in status_response
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
    status = status_response.get("result", {}).get("structuredContent")
    if not isinstance(status, dict):
        raise ExperimentError("cidx status lacks structured content")
    for name in ("stale_count", "unindexed_count", "deleted_count"):
        if status.get(name) != 0:
            raise ExperimentError(f"isolated cidx status {name}={status.get(name)}")
    if status.get("dirty") is not False:
        raise ExperimentError("isolated cidx source is dirty")
    return {
        "names": tools,
        "tools": sorted(raw_tools, key=lambda item: item["name"]),
        "sha256": hashlib.sha256(canonical).hexdigest(),
        "isolated_status": status,
    }


def codex_command(
    codex_binary: str,
    controls: dict[str, Any],
    schema: Path,
    root: Path,
    final_path: Path,
    arm: str,
    mcp_binary: Path,
    state_root: Path | None,
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
    if arm == CIDX_ARM:
        if state_root is None:
            raise ExperimentError("treatment state root is required")
        command.extend(
            [
                "--config",
                f"mcp_servers.cidx.command={json.dumps(str(mcp_binary))}",
                "--config",
                "mcp_servers.cidx.args="
                + json.dumps(
                    [
                        "--source-root",
                        str(root),
                        "--state-root",
                        str(state_root),
                    ]
                ),
                "--config",
                'mcp_servers.cidx.default_tools_approval_mode="approve"',
            ]
        )
    elif arm != BASELINE_ARM:
        raise ExperimentError(f"unknown arm: {arm}")
    command.append("-")
    return command


def event_observation(events_path: Path, final_path: Path) -> dict[str, Any]:
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
    discovery_actions: list[dict[str, Any]] = []
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
        if event.get("type") == "item.started":
            if kind == "mcp_tool_call" and item.get("server") == "cidx":
                tool = item.get("tool")
                if tool in ("search", "read_span"):
                    discovery_actions.append(
                        {
                            "kind": f"cidx_{tool}",
                            "id": item.get("id"),
                            "ordinal": len(discovery_actions),
                        }
                    )
            elif kind == "command_execution":
                command_text = str(item.get("command", ""))
                if re.search(
                    r"(?i)(?:^|[;&|()\s])(?:rg|grep|find|fd|ls|tree|sed|cat|head|tail|awk|nl)(?:\s|$)|git\s+grep",
                    command_text,
                ):
                    discovery_actions.append(
                        {
                            "kind": "shell_repository_inspection",
                            "id": item.get("id"),
                            "ordinal": len(discovery_actions),
                        }
                    )
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
        "discovery_actions": discovery_actions,
        "first_repository_discovery_action": (
            discovery_actions[0]["kind"] if discovery_actions else None
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
    arm: str,
    task_id: str,
    prompt: str,
) -> dict[str, Any]:
    output_dir.mkdir(parents=True, exist_ok=False)
    prompt_path = output_dir / "prompt.txt"
    events_path = output_dir / "events.jsonl"
    stderr_path = output_dir / "stderr.txt"
    final_path = output_dir / "final.json"
    observation_path = output_dir / "observation.json"
    prompt_path.write_text(prompt, encoding="utf-8")
    command = codex_command(
        codex_binary, controls, schema, root, final_path, arm, mcp_binary, state_root
    )
    write_json(
        output_dir / "command.json",
        {
            "argv": command,
            "cwd": str(root),
            "task_id": task_id,
            "arm": arm,
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
    observation = event_observation(events_path, final_path)
    observation.update(
        {
            "task_id": task_id,
            "arm": arm,
            "exit_code": process.returncode,
            "timed_out": timed_out,
            "elapsed_seconds": elapsed,
            "valid_execution": process.returncode == 0 and not timed_out,
        }
    )
    write_json(observation_path, observation)
    return observation


def control_violations(
    arm: str,
    observation: dict[str, Any],
    *,
    require_first_cidx_search: bool,
) -> list[str]:
    violations: list[str] = []
    calls = observation.get("mcp_calls", [])
    if arm == BASELINE_ARM and calls:
        violations.append("baseline_mcp_call")
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
        if re.search(
            r"(?i)(?:^|[;&|()\s])cidx(?:\s|$)|(?:command\s+-v|which|type)\s+cidx",
            command_text,
        ):
            violations.append("shell_cidx_attempt")
    if arm == CIDX_ARM and require_first_cidx_search:
        first = observation.get("first_repository_discovery_action")
        if first is None:
            violations.append("missing_required_first_cidx_search")
        elif first != "cidx_search":
            violations.append("first_repository_discovery_not_cidx_search")
    return sorted(set(violations))


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
    arm: str,
    task_id: str,
    prompt: str,
    require_first_cidx_search: bool = True,
) -> dict[str, Any]:
    source_root: Path | None = None
    state_root: Path | None = None
    try:
        source_root = copy_source_worktree(original_root)
        source_before = verify_isolated_source(source_root, corpus)
        state_before: str | None = None
        if arm == CIDX_ARM:
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
        )
        observation.update(
            {
                "source_before": source_before,
                "source_after": source_after,
                "state_database_sha256_before": state_before,
                "state_database_sha256_after": state_after,
                "control_violations": violations,
                "valid_execution": observation["valid_execution"]
                and not violations,
                "isolated_source_removed_after_capture": True,
                "isolated_state_removed_after_capture": state_root is not None,
                "cidx_binary_sha256": sha256_file(cidx_binary),
                "mcp_binary_sha256": sha256_file(mcp_binary),
            }
        )
        write_json(output_dir / "observation.json", observation)
        if any(
            violation in {"baseline_mcp_call", "unexpected_mcp_server"}
            for violation in violations
        ):
            raise ExperimentError(
                f"global tool-exposure violation in {task_id}/{arm}: {violations}"
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
    args = parser.parse_args()

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
    manifest_status = manifest.get("status")
    if manifest_status != "frozen_for_execution" and not (
        args.preflight_only and manifest_status == "frozen_for_external_review"
    ):
        raise ExperimentError(
            f"manifest is not executable: {manifest_status}"
        )
    if not cidx_binary.is_file() or not os.access(cidx_binary, os.X_OK):
        raise ExperimentError(f"cidx binary is not executable: {cidx_binary}")
    if not mcp_binary.is_file() or not os.access(mcp_binary, os.X_OK):
        raise ExperimentError(f"MCP launcher is not executable: {mcp_binary}")
    if not schema.is_file():
        raise ExperimentError(f"answer schema is missing: {schema}")

    sources = question_sources(project_root, manifest)
    bindings = local_bindings(project_root, bindings_path)
    corpus_specs = {item["corpus_id"]: item for item in manifest["corpora"]}
    corpus_records: dict[str, Any] = {}
    for corpus_id, corpus in corpus_specs.items():
        if corpus_id not in bindings:
            raise ExperimentError(f"missing local corpus binding: {corpus_id}")
        corpus_records[corpus_id] = verify_corpus(
            corpus, bindings[corpus_id], cidx_binary
        )
    preflight_source: Path | None = None
    preflight_state: Path | None = None
    try:
        first_corpus_id = manifest["tasks"][0]["corpus_id"]
        preflight_source = copy_source_worktree(bindings[first_corpus_id])
        verify_isolated_source(preflight_source, corpus_specs[first_corpus_id])
        preflight_state, _ = copy_cidx_state(bindings[first_corpus_id])
        tool_schema = verify_mcp_tools(
            mcp_binary, preflight_source, preflight_state
        )
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

    controls = manifest["controls"]
    run_id = args.run_id or (
        "assistant-ab-v2-" + dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
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
        "cidx_tools": tool_schema["names"],
        "environment_variable_allowlist": sorted(isolated_environment()),
        "controls": controls,
        "corpora": corpus_records,
    }
    write_json(run_root / "run-manifest.json", run_manifest)
    write_json(run_root / "tool-schema.json", tool_schema)
    print(f"preflight passed; run={run_id}", flush=True)
    if args.preflight_only:
        return 0

    probe_corpus_id = manifest["tasks"][0]["corpus_id"]
    for arm in (BASELINE_ARM, CIDX_ARM):
        print(f"schema probe: {arm}", flush=True)
        probe = execute_isolated(
            codex_binary=args.codex_binary,
            mcp_binary=mcp_binary,
            cidx_binary=cidx_binary,
            controls=controls,
            schema=schema,
            original_root=bindings[probe_corpus_id],
            corpus=corpus_specs[probe_corpus_id],
            output_dir=run_root / SCHEMA_PROBE_TASK / arm,
            arm=arm,
            task_id=SCHEMA_PROBE_TASK,
            prompt=SCHEMA_PROBE_PROMPT,
            require_first_cidx_search=False,
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
            raise ExperimentError(f"invalid schema probe: {arm}")

    selected = set(args.only_task)
    for task in manifest["tasks"]:
        task_id = task["task_id"]
        if selected and task_id not in selected:
            continue
        case = sources[task["question_source_index"]][task_id]
        prompt = manifest["prompt_template"].replace("{{TASK_ID}}", task_id).replace(
            "{{QUESTION}}", case["text"]
        )
        second_arm = CIDX_ARM if task["first_arm"] == BASELINE_ARM else BASELINE_ARM
        for arm in (task["first_arm"], second_arm):
            print(f"task {task['sequence']:02d}/12 {task_id}: {arm}", flush=True)
            observation = execute_isolated(
                codex_binary=args.codex_binary,
                mcp_binary=mcp_binary,
                cidx_binary=cidx_binary,
                controls=controls,
                schema=schema,
                original_root=bindings[task["corpus_id"]],
                corpus=corpus_specs[task["corpus_id"]],
                output_dir=run_root / task_id / arm,
                arm=arm,
                task_id=task_id,
                prompt=prompt,
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
