#!/usr/bin/env bash
# Verify one local archive without network access or user host configuration.
set -euo pipefail

[ $# -ge 1 ] || { printf 'usage: %s <archive> [checksums.txt]\n' "$0" >&2; exit 2; }
archive=$(CDPATH= cd -- "$(dirname -- "$1")" && pwd -P)/$(basename -- "$1")
checksums=${2:-"$(dirname -- "$archive")/checksums.txt"}
fail() { printf 'verify-local-release: %s\n' "$*" >&2; exit 1; }
[ "$(uname -s)" = Darwin ] || fail 'only darwin is locally verified'
[ "$(uname -m)" = arm64 ] || fail 'only arm64 is locally verified'
command -v sandbox-exec >/dev/null || fail 'sandbox-exec is unavailable; offline verification is unverified'
command -v git >/dev/null || fail 'git is required for the synthetic repository'
command -v sqlite3 >/dev/null || fail 'sqlite3 is required for the newer-schema negative check'
command -v codex >/dev/null || fail 'codex CLI is required for isolated project-config verification'
command -v python3 >/dev/null || fail 'python3 is required for structural transcript validation'
[ -f "$archive" ] && [ -f "$checksums" ] || fail 'archive or checksum manifest is absent'

profile='(version 1) (deny network*) (allow default)'
offline() { env -u VOYAGE_API_KEY GOPROXY=off sandbox-exec -p "$profile" "$@"; }
work=$(mktemp -d "${TMPDIR:-/tmp}/cidx-release.XXXXXX")
trap 'rm -rf "$work"' EXIT
archive_name=$(basename -- "$archive")
cp "$archive" "$work/$archive_name"
cp "$checksums" "$work/checksums.txt"
(cd "$work" && shasum -a 256 -c checksums.txt)
cp "$work/$archive_name" "$work/corrupt.tar.gz"
printf 'corrupt' >>"$work/corrupt.tar.gz"
if (cd "$work" && shasum -a 256 -c <(sed 's|cidx_.*\.tar\.gz|corrupt.tar.gz|' checksums.txt)); then
  fail 'corrupt archive unexpectedly verified'
fi
mkdir "$work/unpack"
tar -xzf "$work/$archive_name" -C "$work/unpack"
binary="$work/unpack/cidx"
[ -x "$binary" ] || fail 'archive lacks executable cidx'
chmod a-x "$binary"
if "$binary" version >/dev/null 2>&1; then
  fail 'non-executable artifact unexpectedly ran'
fi
chmod 0755 "$binary"
offline "$binary" version --json >"$work/version.json"
python3 - "$work/unpack/build-manifest.json" "$work/version.json" <<'PY'
import json,sys
m=json.load(open(sys.argv[1])); v=json.load(open(sys.argv[2])); r=v["runtime"]
assert m["build_info"] == v and m["archive_target"] == "darwin_arm64"
assert (v["target_os"],v["target_arch"],v["cgo_enabled"]) == ("darwin","arm64","1")
assert r["fts5_available"] and r["wal_available"] and r["registered_languages"] == ["go","typescript","tsx"]
assert r["production_schema_minimum"] == 1 and r["production_schema_maximum"] == v["production_schema_version"]
PY
for document in install.md hosts.md hooks.md upgrade.md; do
  [ -f "$work/unpack/docs/$document" ] || fail "archive missing docs/$document"
done

repo="$work/cidx phase14 한글"
mkdir -p "$repo"
offline git -C "$repo" init >/dev/null
printf 'package sample\nfunc Hello() string { return "hello" }\n' >"$repo/main.go"
printf 'export function hello(): string { return "hello" }\n' >"$repo/sample.ts"
printf 'export const View = () => <div>hello</div>\n' >"$repo/sample.tsx"
(cd "$repo" && offline "$binary" init --serving-dim 256)
(cd "$repo" && offline "$binary" index --reason manual >"$work/index.json")
(
  cd "$repo"
  offline "$binary" index --reason manual >"$work/index-a.json" &
  first=$!
  offline "$binary" index --reason manual >"$work/index-b.json" &
  second=$!
  wait "$first"
  wait "$second"
)
(cd "$repo" && offline "$binary" status --json >"$work/status.json")
[ ! -e "$repo/.cidx/lab/embeddings.db" ] || fail 'free runtime opened lab database'
[ ! -d "$repo/.cidx/lab" ] || fail 'free runtime created lab directory'
if find "$repo" -type f \( -name '*.wasm' -o -name '*.onnx' -o -name '*.gguf' \) -print -quit | grep -q .; then
  fail 'runtime created grammar or model download artifact'
fi

sha=$(shasum -a 256 "$repo/main.go" | awk '{print $1}')
fifo="$work/mcp.stdin"
mkfifo "$fifo"
offline "$binary" serve --root "$repo" <"$fifo" >"$work/mcp.jsonl" 2>"$work/mcp.stderr" &
server=$!
exec 3>"$fifo"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"local-release","version":"1"}}}' >&3
for _ in $(seq 1 50); do
  grep -q '"id":1' "$work/mcp.jsonl" && break
  sleep 0.1
done
grep -q '"id":1' "$work/mcp.jsonl" || fail 'initialize response missing'
if grep '"id":1' "$work/mcp.jsonl" | grep -q '"error":'; then fail 'initialize failed'; fi
{
  printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}'
  printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  printf '%s\n' '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"status","arguments":{}}}'
  printf '%s\n' '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search","arguments":{"query":"Hello","mode":"fts","max_inline_bytes":0}}}'
  printf '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"read_span","arguments":{"path":"main.go","start_line":1,"end_line":2,"expected_sha256":"%s"}}}\n' "$sha"
  printf '%s\n' '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"reindex","arguments":{"dry_run":true}}}'
} >&3
for _ in $(seq 1 50); do
  complete=true
  for request_id in 2 3 4 5 6; do
    grep -q "\"id\":$request_id" "$work/mcp.jsonl" || complete=false
  done
  "$complete" && break
  sleep 0.1
done
exec 3>&-
wait "$server"
awk 'substr($0, 1, 1) != "{" { exit 1 }' "$work/mcp.jsonl" || fail 'serve stdout contained nonprotocol output'
[ ! -s "$work/mcp.stderr" ] || fail 'successful serve wrote diagnostics to stderr'
for name in status search read_span reindex; do grep -q "\"name\":\"$name\"" "$work/mcp.jsonl" || fail "MCP tool missing: $name"; done
grep -q '"id":6' "$work/mcp.jsonl" || fail 'dry reindex response missing'
python3 - "$work/mcp.jsonl" <<'PY'
import json,sys
frames=[json.loads(x) for x in open(sys.argv[1]) if x.strip()]
assert all(isinstance(x,dict) and x.get("jsonrpc") == "2.0" for x in frames)
assert [x.get("id") for x in frames].count(None) == 0
assert {x.get("id") for x in frames} == set(range(1,7)) and len(frames) == 6
assert all(sum(x.get("id") == i for x in frames) == 1 for i in range(1,7))
by_id={x["id"]:x for x in frames}
for i in range(1,7):
  assert i in by_id and "error" not in by_id[i], f"missing/error id {i}"
tools=by_id[2]["result"]["tools"]
assert [x["name"] for x in tools] == ["status","search","read_span","reindex"]
for i in range(3,7):
  r=by_id[i]["result"]
  assert not r.get("isError",False), f"application error {i}"
search=by_id[4]["result"]["structuredContent"]
assert any(x["path"] == "main.go" and x["symbol"] == "Hello" for x in search["hits"])
span=by_id[5]["result"]["structuredContent"]
assert span["body"] == 'package sample\nfunc Hello() string { return "hello" }\n'
assert by_id[6]["result"]["structuredContent"]["dry_run"] is True
PY

other="$work/other-root"
mkdir "$other"
offline git -C "$other" init >/dev/null
mkdir "$other/.cidx"
cp "$repo/.cidx/config.json" "$other/.cidx/config.json"
cp "$repo/.cidx/index.db" "$other/.cidx/index.db"
if offline "$binary" status --root "$other" >"$work/root-mismatch.stdout" 2>"$work/root-mismatch.stderr"; then fail 'different-root database unexpectedly opened'; fi
[ ! -s "$work/root-mismatch.stdout" ] || fail 'root-mismatch failure wrote protocol/data to stdout'
grep -Eiq 'different root|belongs to different root' "$work/root-mismatch.stderr" || fail 'root-mismatch diagnostic was not actionable'
schema_repo="$work/newer-schema-root"
mkdir "$schema_repo"
offline git -C "$schema_repo" init >/dev/null
(cd "$schema_repo" && offline "$binary" init --serving-dim 256)
offline sqlite3 "$schema_repo/.cidx/index.db" 'PRAGMA user_version=999;'
if offline "$binary" status --root "$schema_repo" >"$work/newer-schema.stdout" 2>"$work/newer-schema.stderr"; then fail 'newer schema unexpectedly opened'; fi
[ ! -s "$work/newer-schema.stdout" ] || fail 'newer-schema failure wrote protocol/data to stdout'
grep -Eiq 'newer than supported|schema version' "$work/newer-schema.stderr" || fail 'newer-schema diagnostic was not actionable'

invalid="$work/invalid-config"
mkdir "$invalid"
offline git -C "$invalid" init >/dev/null
mkdir "$invalid/.cidx"
printf '{"version":1,"unknown_phase14_field":true}\n' >"$invalid/.cidx/config.json"
if offline "$binary" status --root "$invalid" >"$work/invalid-config.stdout" 2>"$work/invalid-config.stderr"; then fail 'unsupported config unexpectedly opened'; fi
[ ! -s "$work/invalid-config.stdout" ] || fail 'invalid-config failure wrote protocol/data to stdout'
grep -Eiq 'unknown|config' "$work/invalid-config.stderr" || fail 'invalid-config diagnostic was not actionable'

project="$work/codex-project"
mkdir -p "$project/.codex" "$work/codex-home"
printf '[mcp_servers.cidx]\ncommand = "%s"\nargs = ["serve", "--root", "%s"]\ncwd = "%s"\nenv_vars = ["VOYAGE_API_KEY"]\n' "$binary" "$repo" "$repo" >"$project/.codex/config.toml"
mkdir -p "$work/home"
printf '[projects."%s"]\ntrust_level = "trusted"\n' "$project" >"$work/codex-home/config.toml"
HOME="$work/home" CODEX_HOME="$work/codex-home" offline codex -C "$project" mcp get cidx --json >"$work/codex-mcp.json"
HOME="$work/home" CODEX_HOME="$work/codex-home" offline codex -C "$project" mcp list --json >"$work/codex-mcp-list.json"
offline codex --version >"$work/codex-version.txt"
python3 - "$work/codex-mcp.json" "$work/codex-mcp-list.json" "$binary" "$repo" <<'PY'
import json,sys
g=json.load(open(sys.argv[1])); l=json.load(open(sys.argv[2])); binary,root=sys.argv[3:]
assert g["command"] == binary and g["args"] == ["serve","--root",root] and g.get("cwd") == root
assert g.get("env_vars") == ["VOYAGE_API_KEY"]
assert isinstance(l,list) and [x.get("name") for x in l] == ["cidx"]
PY
if [ -n "${CIDX_EVIDENCE_DIR:-}" ]; then
  mkdir -p "$CIDX_EVIDENCE_DIR"
  cp "$work/version.json" "$work/status.json" "$work/mcp.jsonl" "$work/mcp.stderr" "$work/codex-mcp.json" "$work/codex-mcp-list.json" "$work/codex-version.txt" \
    "$work/root-mismatch.stdout" "$work/root-mismatch.stderr" "$work/newer-schema.stdout" "$work/newer-schema.stderr" \
    "$work/invalid-config.stdout" "$work/invalid-config.stderr" "$CIDX_EVIDENCE_DIR/"
  printf 'local verifier transcripts copied to %s\n' "$CIDX_EVIDENCE_DIR"
fi
printf 'verified local darwin/arm64 archive; no model or assistant invocation was performed\n'
